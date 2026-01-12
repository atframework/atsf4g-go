package lobbysvr_logic_quest_internal

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	config "github.com/atframework/atsf4g-go/component-config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	logic_time "github.com/atframework/atsf4g-go/component-logical_time"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	. "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest/data"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
)

var userManagerReflectType reflect.Type

func init() {
	var _ logic_quest.UserQuestManager = (*UserQuestManager)(nil)
	userManagerReflectType = lu.GetStaticReflectType[UserQuestManager]()
	data.RegisterUserModuleManagerCreator[logic_quest.UserQuestManager](func(_ cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserQuestManager(owner)
	})

	data.RegisterUserItemManagerCreator([]data.UserItemTypeIdRange{
		data.MakeUserItemTypeIdRange(
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_QUEST_BEGIN),
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_QUEST_END)),
	}, func(ctx cd.RpcContext, owner *data.User) data.UserItemManagerImpl {
		mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](owner)
		if mgr == nil {
			ctx.LogError("can not find user quest manager")
			return nil
		}
		convert, ok := mgr.(data.UserItemManagerImpl)
		if !ok || convert == nil {
			ctx.LogError("user quest manager does not implement UserItemManagerImpl")
			return nil
		}
		return convert
	})

	registerCondition()
}

func (m *UserQuestManager) GetReflectType() reflect.Type {
	return userManagerReflectType
}

type UserQuestManager struct {
	data.UserModuleManagerBase
	data.UserItemManagerBase

	// quests private_protocol_pbdesc.UserQuestData

	quests UserQuestListData
	// 二级索引
	// innerProgressType -> param_1 -> []*ProgressKey
	progressKeyIndex UserProgreesKeyIndex

	dirtyQuestEvent      map[int32]*public_protocol_pbdesc.DQuestData
	dirtyExpriedQuestIDs []int32
	dirtyAutoReward      map[int32][]*public_protocol_common.DItemBasic

	questResetList   QuestTimePointEntrySortQueue
	questExpriedList QuestTimePointEntrySortQueue
	// quest_exsit_list   []*QuestStatusEntry

	existQuestIDs map[int32]public_protocol_common.EnQuestStatus

	eventQueue      []EventQueueItem
	eventQueueGuard bool
}

func CreateUserQuestManager(owner *data.User) *UserQuestManager {
	mgr := &UserQuestManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
		progressKeyIndex:      make(UserProgreesKeyIndex),
		existQuestIDs:         make(map[int32]public_protocol_common.EnQuestStatus),
		dirtyQuestEvent:       make(map[int32]*public_protocol_pbdesc.DQuestData),
		dirtyExpriedQuestIDs:  make([]int32, 0),
	}

	// 初始化 UserQuestListData 的 map 字段
	mgr.quests.ProgressingQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	mgr.quests.CompletedQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	mgr.quests.ReceivedQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	mgr.quests.DeleteCache = make(map[int32]*private_protocol_pbdesc.QuestDeleteCache)
	mgr.quests.ExpiredQuestsID = make([]int32, 0)

	return mgr
}

func (m *UserQuestManager) GetOwner() *data.User {
	return m.UserItemManagerBase.GetOwner()
}

func registerCondition() {
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_QuestStatus(), nil, checkRuleQuestStatus)

}

// 道具相关api实现.
func (m *UserQuestManager) AddItem(ctx cd.RpcContext, itemAddGuard []*data.ItemAddGuard, _ *data.ItemFlowReason) data.Result {
	for _, addGuard := range itemAddGuard {
		if addGuard == nil {
			continue
		}

		questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(addGuard.Item.ItemBasic.TypeId)
		if questCfg == nil {
			continue
		}
		m.AddQuest(ctx, questCfg)
	}
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) CheckAddItem(_ cd.RpcContext,
	itemOffset []*public_protocol_common.DItemInstance,
) ([]*data.ItemAddGuard, data.Result) {
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"),
			public_protocol_pbdesc.EnErrorCode(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
	}

	for _, item := range itemOffset {
		_, ok := m.existQuestIDs[item.ItemBasic.TypeId]
		if ok {
			return nil, cd.CreateRpcResultError(fmt.Errorf("item type id %d is a quest id", item.ItemBasic.TypeId),
				public_protocol_pbdesc.EnErrorCode(public_protocol_pbdesc.EnErrorCode_EN_ERR_QUEST_ALREADY_ACCEPTED))
		}
	}
	return m.CreateItemAddGuard(itemOffset)
}

func (m *UserQuestManager) SubItem(_ cd.RpcContext, _ []*data.ItemSubGuard, _ *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultError(fmt.Errorf("item unsupport sub"),
		public_protocol_pbdesc.EnErrorCode(public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID))
}

func (m *UserQuestManager) CheckSubItem(_ cd.RpcContext,
	_ []*public_protocol_common.DItemBasic,
) ([]*data.ItemSubGuard, data.Result) {
	return m.CreateItemSubGuard(nil)
}

func (m *UserQuestManager) ForeachItem(_ func(item *public_protocol_common.DItemInstance) bool) {
}

func (m *UserQuestManager) GenerateItemInstanceFromBasic(_ cd.RpcContext,
	_ *public_protocol_common.DItemBasic,
) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GenerateItemInstanceFromCfgOffset(_ cd.RpcContext,
	_ *public_protocol_common.Readonly_DItemOffset,
) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GenerateItemInstanceFromOffset(_ cd.RpcContext,
	_ *public_protocol_common.DItemOffset,
) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GetTypeStatistics(ctx cd.RpcContext, _ int32) *data.ItemTypeStatistics {
	return nil
}

func (m *UserQuestManager) GetItemFromBasic(ctx cd.RpcContext, _ *public_protocol_common.DItemBasic) (
	*public_protocol_common.DItemInstance, data.Result,
) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GetNotEnoughErrorCode(_ int32) int32 {
	return 0
}

func (m *UserQuestManager) CheckTypeIDValid(_ int32) bool {
	return true
}

// db load & save

func (m *UserQuestManager) InitFromDB(ctx cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	// m.quests = *dbUser.GetQuestData().Clone()
	m.existQuestIDs = make(map[int32]public_protocol_common.EnQuestStatus)

	m.quests.ProgressingQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	for _, questData := range dbUser.GetQuestData().GetUserQuestList().GetProgressingQuests() {
		m.quests.ProgressingQuests[questData.GetQuestId()] = questData.Clone()
		m.existQuestIDs[questData.GetQuestId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING
	}

	m.quests.CompletedQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	for _, questData := range dbUser.GetQuestData().GetUserQuestList().GetCompletedQuests() {
		m.quests.CompletedQuests[questData.GetQuestId()] = questData.Clone()
		m.existQuestIDs[questData.GetQuestId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE
	}

	m.quests.ReceivedQuests = make(map[int32]*public_protocol_pbdesc.DQuestData)
	for _, questData := range dbUser.GetQuestData().GetUserQuestList().GetReceivedQuests() {
		m.quests.ReceivedQuests[questData.GetQuestId()] = questData.Clone()
		m.existQuestIDs[questData.GetQuestId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE
	}

	m.quests.ExpiredQuestsID = make([]int32, 0, len(dbUser.GetQuestData().GetUserQuestList().GetExpiredQuests()))
	for _, expiredQuestID := range dbUser.GetQuestData().GetUserQuestList().GetExpiredQuests() {
		m.quests.ExpiredQuestsID = append(m.quests.ExpiredQuestsID, expiredQuestID)
		m.existQuestIDs[expiredQuestID] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED
	}

	m.quests.DeleteCache = make(map[int32]*private_protocol_pbdesc.QuestDeleteCache)
	for _, expiredQuestData := range dbUser.GetQuestData().GetUserQuestList().GetDeleteCache() {
		m.quests.DeleteCache[expiredQuestData.GetDeleteCache().GetQuestId()] = expiredQuestData.Clone()
		// m.existQuestIDs[expiredQuestData.GetQuestId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_DELETE_CACHE
	}

	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if functionUnlockManager != nil {
		functionUnlockManager.RegisterFunctionUnlockEvent(ctx, public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_QUEST, m)
	}

	// 重建各种索引
	m.buildUserQuestIndexes(ctx)

	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) DumpToDB(_ cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	if dbUser == nil {
		return cd.CreateRpcResultOk()
	}
	userQuestData := dbUser.MutableQuestData().MutableUserQuestList()

	// 清空现有数据并重新填充
	userQuestData.ProgressingQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.ProgressingQuests))
	for _, questData := range m.quests.ProgressingQuests {
		userQuestData.ProgressingQuests = append(userQuestData.ProgressingQuests, questData)
	}

	userQuestData.CompletedQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.CompletedQuests))
	for _, questData := range m.quests.CompletedQuests {
		userQuestData.CompletedQuests = append(userQuestData.CompletedQuests, questData)
	}

	userQuestData.ReceivedQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.ReceivedQuests))
	for _, questData := range m.quests.ReceivedQuests {
		userQuestData.ReceivedQuests = append(userQuestData.ReceivedQuests, questData)
	}

	userQuestData.ExpiredQuests = make([]int32, len(m.quests.ExpiredQuestsID))
	copy(userQuestData.ExpiredQuests, m.quests.ExpiredQuestsID)

	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) DumpQuestInfo(questData *public_protocol_pbdesc.DUserQuestsData) {
	questData.ProgressingQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.ProgressingQuests))
	for _, quest := range m.quests.ProgressingQuests {
		questData.ProgressingQuests = append(questData.ProgressingQuests, quest)
	}

	questData.CompletedQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.CompletedQuests))
	for _, quest := range m.quests.CompletedQuests {
		questData.CompletedQuests = append(questData.CompletedQuests, quest)
	}

	questData.ReceivedQuests = make([]*public_protocol_pbdesc.DQuestData, 0, len(m.quests.ReceivedQuests))
	for _, quest := range m.quests.ReceivedQuests {
		questData.ReceivedQuests = append(questData.ReceivedQuests, quest)
	}
}

func (m *UserQuestManager) buildUserQuestIndexes(ctx cd.RpcContext) {
	// Implementation for rebuilding various indexes related to user quests
	m.progressKeyIndex = make(UserProgreesKeyIndex)
	m.questExpriedList = QuestTimePointEntrySortQueue{}
	m.questResetList = QuestTimePointEntrySortQueue{}

	for _, questData := range m.quests.ProgressingQuests {
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questData.GetQuestId())
		if questCfg == nil {
			ctx.LogError("quest config not found when build indexes",
				"quest_id", questData.GetQuestId())
			continue
		}

		m.insertUserQuestIndexes(ctx, questCfg)

		// 插入过期队列
		m.insertExpriedQuestList(questData.GetQuestId(), questData.ExpiredTime)
		m.insertResetQuestList(questData.GetQuestId(), questData.ResetTime)
	}

	for _, questData := range m.quests.CompletedQuests {
		m.insertExpriedQuestList(questData.GetQuestId(), questData.ExpiredTime)
		m.insertResetQuestList(questData.GetQuestId(), questData.ResetTime)
	}

	for _, questData := range m.quests.ReceivedQuests {
		// m.insertExpriedQuestList(questData.GetQuestId(), questData.ExpiredTime)
		m.insertResetQuestList(questData.GetQuestId(), questData.ResetTime)
	}
}

func (m *UserQuestManager) insertUserQuestIndexes(_ cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) {
	for _, progressCfg := range questCfg.GetProgress() {
		progressType := progressCfg.GetProgressParamOneofCase()
		param1 := GetProgressCfgParam1Value(progressCfg)
		if _, ok := m.progressKeyIndex[progressType]; !ok {
			m.progressKeyIndex[progressType] = make(map[int32]map[int32]*ProgressKey)
		}
		if _, ok := m.progressKeyIndex[progressType][param1]; !ok {
			m.progressKeyIndex[progressType][param1] = make(map[int32]*ProgressKey)
		}
		if _, ok := m.progressKeyIndex[progressType][param1][questCfg.GetId()]; !ok {
			m.progressKeyIndex[progressType][param1][questCfg.GetId()] = &ProgressKey{
				QuestID:          questCfg.GetId(),
				ProgressUniqueID: progressCfg.GetUniqueId(),
			}
		}
	}
}

func (m *UserQuestManager) removeUserQuestIndexes(_ cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) {
	for _, progressCfg := range questCfg.GetProgress() {
		progressType := progressCfg.GetProgressParamOneofCase()
		param1 := GetProgressCfgParam1Value(progressCfg)
		progressKeyMap, ok := m.progressKeyIndex[progressType][param1]
		if !ok {
			if len(m.progressKeyIndex[progressType]) == 0 {
				delete(m.progressKeyIndex, progressType)
			}
			continue
		}
		delete(progressKeyMap, questCfg.GetId())
		if len(progressKeyMap) == 0 {
			delete(m.progressKeyIndex[progressType], param1)
		}

		if len(m.progressKeyIndex[progressType]) == 0 {
			delete(m.progressKeyIndex, progressType)
		}
	}
}

func (m *UserQuestManager) insertExpriedQuestList(questID int32, expriedTime int64) {
	if expriedTime > 0 {
		m.questExpriedList.Insert(QuestTimePointEntry{
			QuestID:   questID,
			Timepoint: expriedTime,
		})
	}
}

func (m *UserQuestManager) insertResetQuestList(questID int32, resetTime int64) {
	if resetTime > 0 {
		m.questResetList.Insert(QuestTimePointEntry{
			QuestID:   questID,
			Timepoint: resetTime,
		})
	}
}

func (m *UserQuestManager) Rebuild(ctx cd.RpcContext) {
	m.OnResourceVersionChanged(ctx)
}

func (m *UserQuestManager) NotifyFunctionUnlock(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, unlockIDs []int32) {
	if len(unlockIDs) == 0 {
		return
	}
	if functionID != public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_QUEST {
		return
	}
	for _, questID := range unlockIDs {
		ctx.LogDebug("trying to unlock quest", "quest_id", questID)

		// 正常来说这里的任务玩家不可能已经解锁
		// 但策划可以随便改表
		questStatus := m.existQuestIDs[questID]
		if questStatus != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK {
			continue
		}
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
		if m.CheckQuestIsUnlock(ctx, questCfg) {
			m.AddQuest(ctx, questCfg)
		}
	}
}

func (m *UserQuestManager) LoginInit(ctx cd.RpcContext) {
	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if functionUnlockManager != nil {
		functionUnlockManager.RegisterFunctionUnlockEvent(ctx, public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_QUEST, m)
	}
}

func (m *UserQuestManager) OnLogin(ctx cd.RpcContext) {
	m.OnResourceVersionChanged(ctx)
	m.RefreshLimitSecond(ctx)
	m.cleanUpDeleteCache(ctx)
}

func (m *UserQuestManager) RefreshLimitSecond(ctx cd.RpcContext) {
	// 需要重置的任务
	m.CleanUpExpiredQuests(ctx, ctx.GetNow())
	m.checkPeriodRestCondition(ctx, ctx.GetNow().Unix())
}

func (m *UserQuestManager) QueryQuestStatus(questID int32) public_protocol_common.EnQuestStatus {
	questStatus, ok := m.existQuestIDs[questID]
	if !ok {
		return public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK
	}
	return questStatus
}

func (m *UserQuestManager) QueryQuestIsFinish(questID int32) bool {
	status := m.QueryQuestStatus(questID)
	return status == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE ||
		status == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE
}

// 任务触发.
func (m *UserQuestManager) QuestTriggerEvent(ctx cd.RpcContext,
	triggerType private_protocol_pbdesc.QuestTriggerParams_EnParamID, param *private_protocol_pbdesc.QuestTriggerParams,
) {
	m.eventQueue = append(m.eventQueue, EventQueueItem{
		EventType: triggerType,
		Params:    param,
	})
	if m.eventQueueGuard {
		return
	}
	m.eventQueueGuard = true
	defer func() {
		m.eventQueueGuard = false
	}()

	// 清理可能无效的任务
	// now := ctx.GetNow()
	// m.CleanUpExpiredQuests(ctx, now)
	for len(m.eventQueue) > 0 {
		eventItem := m.eventQueue[0]
		m.eventQueue = m.eventQueue[1:]
		// 处理事件
		m.triggerEventInner(ctx, eventItem)
	}
}

func (m *UserQuestManager) triggerEventInner(ctx cd.RpcContext, eventItem EventQueueItem) {
	triggerCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestTriggerEventTypeById(
		int32(eventItem.EventType)) //nolint:gosec
	if triggerCfg == nil {
		ctx.LogError("quest trigger config not found",
			"trigger_type", eventItem.EventType)
		return
	}
	// 任务重置(周期任务)
	// 不可以在这里重置 过期任务，重置和过期流程要放
	// m.tryPeriodRest(ctx, ctx.GetNow())

	// 任务进度更新
	for _, pregressType := range triggerCfg.GetProgressTypes() {
		m.updateQuestProgressByType(ctx, eventItem.EventType, pregressType, eventItem.Params)
	}
}

func (m *UserQuestManager) updateQuestProgressByType(ctx cd.RpcContext,
	triggerType private_protocol_pbdesc.QuestTriggerParams_EnParamID, pregressType int32,
	params *private_protocol_pbdesc.QuestTriggerParams,
) {
	// 根据触发类型得到所有可能需要更新进度列表
	ProgressKeyList := m.GetProgressKeyList(ctx, pregressType, params)
	if len(ProgressKeyList) == 0 {
		return
	}

	pendingFinishQuestIDs := []int32{}
	for _, progressKey := range ProgressKeyList {
		ctx.LogDebug("updating quest progress",
			"progress_type", pregressType, "quest_id", progressKey.QuestID)

		questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(progressKey.QuestID)

		if questCfg == nil {
			ctx.LogError("quest config not found",
				"quest_id", progressKey.QuestID)
			continue
		}

		// 检查任务是否失效
		if m.CheckQuestInvalid(ctx, questCfg) {
			continue
		}

		questData, ok := m.quests.ProgressingQuests[progressKey.QuestID]
		if !ok || questData == nil {
			ctx.LogError("quest questData data not found", "quest_id", progressKey.QuestID)
			continue
		}

		// 检查是否需要重置
		if questData.ResetTime > 0 && questData.ResetTime <= ctx.GetNow().Unix() {
			m.tryPeriodRest(ctx, ctx.GetNow())
			questData, ok := m.quests.ProgressingQuests[progressKey.QuestID]
			// 删除重置队列
			m.questResetList.Remove(progressKey.QuestID)
			if !ok || questData == nil {
				ctx.LogError("quest questData data not found after reset", "quest_id", progressKey.QuestID)
				continue
			}
		}

		// 检查通用条件
		if !m.CheckQuestCommonCondition(ctx, questCfg) {
			continue
		}

		// 更新任务进度
		for _, progressData := range questData.GetProgress() {
			if progressData.GetUniqueId() != progressKey.ProgressUniqueID {
				continue
			}
			progressCfg := GetProgressCfgByUniqueId(progressData.GetUniqueId(), questCfg.GetProgress())
			if progressCfg == nil {
				ctx.LogError("quest progress config not found", "quest_id", progressKey.QuestID,
					"progress_unique_id", progressKey.ProgressUniqueID)
				continue
			}
			originValue := progressData.Value
			m.AddquestProgressInner(ctx, questCfg, progressData, pregressType, params, progressCfg)
			ctx.LogDebug("quest progress updated",
				"quest_id", progressData.GetUniqueId(), "progress_type", pregressType,
				"origin_value", originValue, "new_value", progressData.Value)

			if originValue != progressData.Value && m.CheckQuestProgressComplete(ctx, progressCfg, progressData) {
				// 任务已经完成
				pendingFinishQuestIDs = append(pendingFinishQuestIDs, questData.GetQuestId())
			}
		}
	}
	m.FinishQuests(ctx, pendingFinishQuestIDs, false)
}

func (m *UserQuestManager) AddquestProgressInner(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList,
	questProgress *public_protocol_pbdesc.DUserQuestProgressData,
	pregressType int32, params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
) {
	cfgMgr := config.GetConfigManager().GetCurrentConfigGroup()
	progressTypeCfg := cfgMgr.GetExcelQuestProgressTypeById(int32(pregressType))
	if progressTypeCfg == nil {
		ctx.LogError("quest progress type config not found",
			"progress_type", pregressType)
		return
	}

	// 检查通用条件
	if logic_condition.HasLimitData(questCfg.GetCommonCondition()) {
		conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
		if conditionMgr == nil {
			ctx.LogError("condition manager not found")
			return
		}
		ok := conditionMgr.CheckBasicLimit(ctx, questCfg.GetCommonCondition(), logic_condition.CreateEmptyRuleCheckerRuntime())
		if !ok.IsOK() {
			ctx.LogDebug("quest common condition not met",

				"quest_id", questCfg.GetId())
			return
		}
	}

	progressHander := GetQuestProgressHandler(pregressType)
	if progressHander == nil || progressHander.UpdateHandler == nil {
		ctx.LogError("quest progress handler UpdateHandler not found",
			"progress_type", pregressType)
		return
	}
	progressHander.UpdateHandler(ctx, params, progressCfg, questProgress)

	m.addDirtyQuestData(ctx, m.quests.ProgressingQuests[questCfg.GetId()])
}

func (m *UserQuestManager) CheckQuestCommonCondition(ctx cd.RpcContext,
	questCfg *public_protocol_config.Readonly_ExcelQuestList,
) bool {
	// 获取通用条件配置
	commonCondition := questCfg.GetCommonCondition()
	if commonCondition == nil {
		return true // 没有通用条件，默认通过
	}

	// 检查是否有限制数据
	if !logic_condition.HasLimitData(commonCondition) {
		return true // 没有限制数据，默认通过
	}

	// 获取条件管理器
	conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
	if conditionMgr == nil {
		ctx.LogError("failed to get UserConditionManager")
		return false
	}

	// 检查所有条件（包括静态和动态）
	rpcResult := conditionMgr.CheckBasicLimit(ctx, commonCondition, logic_condition.CreateEmptyRuleCheckerRuntime())
	return rpcResult.IsOK()
}

func (m *UserQuestManager) GetProgressKeyList(ctx cd.RpcContext,
	progressType int32,
	params *private_protocol_pbdesc.QuestTriggerParams,
) map[int32]*ProgressKey {
	// handler := GetQuestProgressHandler(public_protocol_config.DQuestConditionProgress_EnProgressParamID(progressType))
	handler := GetQuestProgressHandler(progressType)
	if handler == nil || handler.ProgreesKeyHandler == nil {
		ctx.LogError("quest progress handler GetProgressKeyListHandler not found",

			"progress_type", progressType)
		return nil
	}
	return handler.ProgreesKeyHandler(ctx, progressType, params, m.progressKeyIndex)
}

func (m *UserQuestManager) tryPeriodRest(ctx cd.RpcContext, now time.Time) {
	// 只能由时间触发
	m.checkPeriodRestCondition(ctx, now.Unix())
}

func (m *UserQuestManager) checkPeriodRestCondition(ctx cd.RpcContext, now int64) {
	for m.questResetList.Len() > 0 {
		resetEntry := m.questResetList.Top()
		if resetEntry == nil {
			break
		}
		if resetEntry.Timepoint > now {
			break
		}
		m.questResetList.Pop()
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(resetEntry.QuestID)
		if questCfg == nil {
			ctx.LogError("quest config not found",
				"quest_id", resetEntry.QuestID)
			continue
		}
		if questCfg.GetProgressResetPeriod().GetPeriodDays() == 0 {
			ctx.LogError("quest reset period days is zero",
				"quest_id", resetEntry.QuestID)
			continue
		}

		if m.CheckQuestInvalid(ctx, questCfg) {
			continue
		}

		// 开始重置
		m.StartPeriodQuestRest(ctx, questCfg, resetEntry, now)
	}
}

func (m *UserQuestManager) StartPeriodQuestRest(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList,
	resetEntry *QuestTimePointEntry, now int64,
) {
	// 重置任务
	m.resetQuest(ctx, questCfg, false)
}

func (m *UserQuestManager) QuestHasNoProgress(questCfg *public_protocol_config.Readonly_ExcelQuestList) bool {
	return len(questCfg.GetProgress()) == 0
}

func (m *UserQuestManager) CheckQuestInvalid(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) bool {
	// 判断任务是合法 CleanUpQuestIsInvalid 整合一下可以
	// 清理过去和下架的任务
	if !questCfg.GetOn() {
		return true
	}

	if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
		if ctx.GetNow().Unix() > questCfg.GetAvailablePeriod().GetSpecificPeriod().GetEnd().GetSeconds() {
			return true
		}
	}
	return false
}

// 资源版本变化时，检查任务解锁和完成状态.
func (m *UserQuestManager) OnResourceVersionChanged(ctx cd.RpcContext) {
	// 登录时资源变化需要重新判断未解锁的任务的解锁&&进行中任务的完成状态
	now := ctx.GetNow()
	pendingFinishQuestIDs := []int32{}
	AllQuest := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().QuestSequence
	for _, questCfg := range AllQuest {
		if questCfg == nil {
			continue
		}

		m.CleanUpQuestIsInvalid(ctx, questCfg) // 任务已经非法
		m.QueryQuestIsFinish(questCfg.GetId()) //  任务是否已经完成

		if !questCfg.GetOn() {
			continue
		}

		questStatus := m.existQuestIDs[questCfg.GetId()]

		if questStatus == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING {
			// 检查进行中任务是否完成
			if m.CheckQuestComplete(ctx, questCfg) {
				pendingFinishQuestIDs = append(pendingFinishQuestIDs, questCfg.GetId())
			}
			continue
		}

		if questStatus != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK {
			continue
		}
		m.existQuestIDs[questCfg.GetId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK

		// 时间条件
		if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
			start := questCfg.GetAvailablePeriod().GetSpecificPeriod().GetStart().GetSeconds()
			end := questCfg.GetAvailablePeriod().GetSpecificPeriod().GetEnd().GetSeconds()
			if now.Unix() < start || now.Unix() > end {
				ctx.LogDebug("quest is not in available period",
					"quest_id", questCfg.GetId())
				continue
			}
		}
		// 检查所有解锁条件
		isUnlock := m.CheckQuestIsUnlock(ctx, questCfg)

		if isUnlock {
			m.AddQuest(ctx, questCfg)
		}

		// TODO 版本号管理，现在还没版本号不记录
		// m.quests.UserQuestList.ExcelVersion = config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestList
	}

	if len(pendingFinishQuestIDs) > 0 {
		m.FinishQuests(ctx, pendingFinishQuestIDs, false)
	}

	// m.deleteExpriedDeletequestCache(ctx)
}

func (m *UserQuestManager) CheckQuestComplete(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) bool {
	if m.QuestHasNoProgress(questCfg) {
		return true
	}

	questData := m.quests.ProgressingQuests[questCfg.GetId()]
	for _, progressdata := range questData.Progress {
		progressCfg := GetProgressCfgByUniqueId(progressdata.UniqueId, questCfg.GetProgress())
		if progressCfg == nil {
			ctx.LogError("quest progress config not found",

				"quest_id", questCfg.GetId(), "progress_unique_id", progressdata.UniqueId)
			continue
		}
		if m.CheckQuestProgressComplete(ctx, progressCfg, progressdata) {
			return true
		}
	}
	return false
}

func (m *UserQuestManager) CheckQuestIsUnlock(ctx cd.RpcContext,
	questCfg *public_protocol_config.Readonly_ExcelQuestList,
) bool {
	if len(questCfg.GetUnlockConditions()) == 0 {
		// 无解锁条件，默认解锁
		return true
	}

	if len(questCfg.GetUnlockConditions()) == 1 && questCfg.GetUnlockConditions()[0] == nil {
		return true
	}
	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if functionUnlockManager == nil {
		ctx.LogError("functionUnlockManager is nil")
		return false
	}
	return functionUnlockManager.CheckFunctionUnlock(ctx, questCfg.GetUnlockConditions())
}

// 检查任务是否已经下架或者失效.
func (m *UserQuestManager) CleanUpQuestIsInvalid(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) {
	// 清理过去和下架的任务
	if !questCfg.GetOn() {
		m.ExpiredQuest(ctx, questCfg.GetId())
		ctx.LogInfo("quest is off, delete quest",
			"quest_id", questCfg.GetId())
	}

	// 删除过期任务
	if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
		if ctx.GetNow().Unix() > questCfg.GetAvailablePeriod().GetSpecificPeriod().GetEnd().GetSeconds() {
			m.ExpiredQuest(ctx, questCfg.GetId())
			ctx.LogInfo("quest is expired, delete quest",
				"quest_id", questCfg.GetId())
		}
	}
}

func (m *UserQuestManager) AddQuest(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) {
	questID := questCfg.GetId()
	if m.QuestHasNoProgress(questCfg) {
		// 任务默认解锁就完成
		m.FinishQuest(ctx, questID, true)
		return
	}

	// 判断下是否解锁就完成了
	// peddingAddProgress := map[int32]*public_protocol_pbdesc.DUserQuestProgressData{}
	questData := public_protocol_pbdesc.DQuestData{}
	isFinsh := false

	if deleteCahe, ok := m.quests.DeleteCache[questID]; ok {
		ctx.LogDebug("adding quest from delete cache", "quest_id", questID)
		// 如果任务在删除缓存中，按照数据恢复进度
		questData = *deleteCahe.GetDeleteCache().Clone()
		delete(m.quests.DeleteCache, questID)

		for _, progressData := range questData.GetProgress() {
			progressCfg := GetProgressCfgByUniqueId(progressData.GetUniqueId(), questCfg.GetProgress())
			if m.CheckQuestProgressComplete(ctx, progressCfg, progressData) {
				// 任务解锁就完成
				isFinsh = true
			}
		}
	} else {
		questData.QuestId = questID
		questData.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING
		questData.CreatedTime = ctx.GetNow().Unix()

		// penndingAddProgress := []*public_protocol_pbdesc.DUserQuestProgressData{}

		for _, progressCfg := range questCfg.GetProgress() {
			// 获得初始化赋值
			Progress := public_protocol_pbdesc.DUserQuestProgressData{}
			Progress.Status = public_protocol_common.EnQuestProgressStatusType_EN_QUEST_PROGRESS_STATUS_PROGRESSING
			questProgressHandler := GetQuestProgressHandler(int32(progressCfg.GetProgressParamOneofCase()))

			Progress.CreatedTime = ctx.GetNow().Unix()
			Progress.UniqueCount = make(map[int64]bool)
			Progress.UniqueId = progressCfg.GetUniqueId()

			if progressCfg.GetProgressParamOneofCase() == public_protocol_config.DQuestConditionProgress_EnProgressParamID_NONE {
				// 没有完成条件的任务 为了客户端展示所以添加一下progress
				Progress.Value = logic_quest.DQuestNoPeogressValue
				isFinsh = true
			} else {
				if questProgressHandler != nil && questProgressHandler.InitHandler != nil {
					rpcResult := questProgressHandler.InitHandler(ctx, progressCfg, &Progress, m.GetOwner())
					if !rpcResult.IsOK() {
						ctx.LogError("init quest progress value failed",

							"quest_id", questCfg.GetId(), "progress_type", progressCfg.GetProgressParamOneofCase(),
							"error", rpcResult.Error)
						return
					}
				}

				// penndingAddProgress = append(penndingAddProgress, &Progress)

				if m.CheckQuestProgressComplete(ctx, progressCfg, &Progress) {
					// 任务解锁就完成
					isFinsh = true
				}
			}
			questData.Progress = append(questData.Progress, &Progress)
		}
	}

	m.quests.ProgressingQuests[questID] = &questData

	if isFinsh {
		m.FinishQuest(ctx, questID, true)
		return
	}
	// for _, progressData := range penndingAddProgress {
	// 	questData.Progress = append(questData.Progress, progressData)
	// }

	m.insertUserQuestIndexes(ctx, questCfg)

	// 插入到各种索引里面
	questData.ResetTime = m.GetQuestResetTime(ctx, questCfg)
	questData.ExpiredTime = m.GetQuestExpiredTime(ctx, questCfg)
	m.insertExpriedQuestList(questData.QuestId, questData.ExpiredTime)
	m.insertResetQuestList(questData.QuestId, questData.ResetTime)

	m.insertQuestExsitIdx(questCfg, public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING)
	m.addDirtyQuestData(ctx, &questData)

	m.questDataLog(ctx, &questData, int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK),
		int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING))
}

// func (m *UserQuestManager) addQuestFromDeleteCacheWithProgress(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList,
// 	deleteCahe *private_protocol_pbdesc.QuestDeleteCache,
// ) {
// 	switch deleteCahe.Status {
// 	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE:
// 		m.quests.CompletedQuests[questCfg.GetId()] = &public_protocol_pbdesc.DUserQuestCompletedData{
// 			QuestId:   questCfg.GetId(),
// 			Timepoint: deleteCahe.QuestLastStatusChangeTime,
// 		}
// 		m.insertQuestExsitIdx(questCfg, public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE)
// 		m.addQuestEventComplete(ctx, questCfg.GetId())

// 	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE:
// 		m.quests.ReceivedQuests[questCfg.GetId()] = &public_protocol_pbdesc.DUserQuestReceivedData{
// 			QuestId:   questCfg.GetId(),
// 			Timepoint: deleteCahe.QuestLastStatusChangeTime,
// 		}
// 		m.insertQuestExsitIdx(questCfg, public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE)
// 		m.addQuestEventReceived(ctx, questCfg.GetId())

// 	default:
// 		ctx.LogError("unknown quest status in delete cache",
//
// 			"quest_id", questCfg.GetId(), "status", deleteCahe.Status)
// 		return
// 	}

// 	// 插入到各种索引里面
// 	m.insertQuestExpiredIdx(ctx, questCfg)
// 	m.insertQuestProgressResetIdx(ctx, questCfg)
// }

func (m *UserQuestManager) insertQuestExsitIdx(questCfg *public_protocol_config.Readonly_ExcelQuestList, status public_protocol_common.EnQuestStatus) {
	// 插入任务存在索引
	m.existQuestIDs[questCfg.GetId()] = status
}

func (m *UserQuestManager) CheckQuestProgressComplete(ctx cd.RpcContext, progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	progressData *public_protocol_pbdesc.DUserQuestProgressData,
) bool {
	questProgessTypeConfig := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestProgressTypeById(int32(progressCfg.GetProgressParamOneofCase()))
	if questProgessTypeConfig == nil {
		ctx.LogError("quest progress type config not found",
			"progress_type_id", progressCfg.GetProgressParamOneofCase())
		return false
	}
	result := false

	switch questProgessTypeConfig.GetValueCompareType() {
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_AUTO_COMPLETE:
		result = true
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_GREATER_OR_EQUAL:
		result = progressData.GetValue() >= progressCfg.GetValue()
		if result {
			progressData.Value = progressCfg.GetValue()
		}
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_LESS_OR_EQUAL:
		result = progressData.GetValue() <= progressCfg.GetValue()
		if result {
			progressData.Value = progressCfg.GetValue()
		}
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_STRICTLY_EQUAL:
		result = progressData.GetValue() == progressCfg.GetValue()
	default:
		ctx.LogError("unknown quest progress value compare type",
			"compare_type",
			questProgessTypeConfig.GetValueCompareType())
	}

	return result
}

func (m *UserQuestManager) CleanUpExpiredQuests(ctx cd.RpcContext, now time.Time) {
	expriedQuestIDs := []int32{}

	for m.questExpriedList.Len() > 0 {
		entry := m.questExpriedList.Top()
		if entry.Timepoint <= now.Unix() {
			// 任务已经结束
			expriedQuestIDs = append(expriedQuestIDs, entry.QuestID)
			m.questExpriedList.Pop()
		} else {
			break
		}
	}

	sz := len(expriedQuestIDs)
	if sz != 0 {
		// 过期任务
		m.ExpiredQuests(ctx, expriedQuestIDs)
	}
}

func (m *UserQuestManager) ExpiredQuests(ctx cd.RpcContext, questList []int32) {
	for _, questID := range questList {
		m.ExpiredQuest(ctx, questID)
	}
}

func (m *UserQuestManager) ExpiredQuest(ctx cd.RpcContext, questID int32) {
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {

		ctx.LogWarn("try to delete quest but quest config not found",
			"quest_id", questID)
		return
	}

	originStatus := int(0)

	if m.existQuestIDs[questID] == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED ||
		m.existQuestIDs[questID] == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE {
		ctx.LogWarn("quest ExpiredQuest repeated Expired or Received", "quest_id", questID)
		return
	}

	// 任务删除后先加入删除缓存里保存一段时间，防止误下架等事故
	deleteCache := &private_protocol_pbdesc.QuestDeleteCache{
		DeleteCache:     &public_protocol_pbdesc.DQuestData{},
		DeleteTimepoint: ctx.GetNow().Unix(),
	}

	deleteSucess := false
	var expiredQuestData *public_protocol_pbdesc.DQuestData
	// 从索引中移除
	m.removeUserQuestIndexes(ctx, questCfg)
	if questData, ok := m.quests.ProgressingQuests[questID]; ok {
		expiredQuestData = questData
		deleteCache.DeleteCache = questData.Clone()
		delete(m.quests.ProgressingQuests, questID)
		deleteSucess = true
		originStatus = int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING)
	}

	if !deleteSucess {
		// 尝试从已完成任务里删除
		if questData, ok := m.quests.CompletedQuests[questID]; ok {
			expiredQuestData = questData
			deleteCache.DeleteCache = questData.Clone()
			delete(m.quests.CompletedQuests, questID)
			// deleteCache.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE
			// deleteCache.QuestLastStatusChangeTime = complete.GetTimepoint()
			originStatus = int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE)
			deleteSucess = true
		}
	}

	if deleteSucess {
		ctx.LogDebug("quest expried",
			"quest_id", questID)

		m.existQuestIDs[questID] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED
		m.quests.ExpiredQuestsID = append(m.quests.ExpiredQuestsID, questID)

		if expiredQuestData != nil {
			m.questDataLog(ctx, expiredQuestData, originStatus,
				int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED))
		}
		m.quests.DeleteCache[questID] = deleteCache
		// 标记脏数据 - 任务已删除（通过添加 Received 事件表示任务状态变化）
		m.addDirtyQuestExpriedData(ctx, questID)
	} else {
		ctx.LogDebug("quest false delete",
			"quest_id", questID)
	}
}

func (m *UserQuestManager) FinishQuests(ctx cd.RpcContext, questIDs []int32, noProgress bool) {
	for _, questID := range questIDs {
		m.FinishQuest(ctx, questID, noProgress)
	}
}

func (m *UserQuestManager) FinishQuest(ctx cd.RpcContext, questID int32, noProgress bool) {
	// 走到这里的任务说明已经之前检查过完成条件，现在将任务从progressing状态转到finished状态
	// noProgress 没有任务条件直接完成的任务，
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {
		ctx.LogError("try to finish quest but quest config not found",
			"quest_id", questID)
		return
	}
	if !noProgress {
		// 删除索引
		m.removeUserQuestIndexes(ctx, questCfg)
	}

	// 插入完成任务队列
	questData := m.quests.ProgressingQuests[questID]
	if questData == nil {
		// 任务直接完成会导致这种情况需要处理
		questData = &public_protocol_pbdesc.DQuestData{
			QuestId:     questID,
			ResetTime:   m.GetQuestResetTime(ctx, questCfg),
			ExpiredTime: m.GetQuestExpiredTime(ctx, questCfg),
			CreatedTime: ctx.GetNow().Unix(),
		}
		ctx.LogDebug("quest finish with no progress",
			"quest_id", questID)
	}
	// 	ctx.LogError("try to finish quest but quest data not found in ProgressingQuests",
	// 		 "quest_id", questID)
	// 	return
	// }
	delete(m.quests.ProgressingQuests, questID)
	questData.CompletedTime = ctx.GetNow().Unix()
	questData.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE
	m.quests.CompletedQuests[questID] = questData

	// 任务状态已完成
	m.existQuestIDs[questID] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE

	m.questDataLog(ctx, questData, int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING),
		int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE))

	// 自动领取
	giveOutType := public_protocol_config.EnQuestRewardGiveOutType_EN_QUEST_REWARD_GIVE_OUT_TYPE_AUTO_INVENTORY
	if questCfg.GetRewards().GetGiveOutType() == giveOutType {
		_, err := m.ReceivedQuestReward(ctx, questID, true)
		if err.IsError() {
			ctx.LogError("auto receive quest reward failed",
				"quest_id", questID,
				"error", err.GetStandardError())
		}
		return
	}

	unlockMgr := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if unlockMgr != nil {
		unlockMgr.OnUserUnlockDataChange(ctx, public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestFinish,
			0, int64(questID))
	}

	// 标记脏数据 - 任务完成
	m.addDirtyQuestData(ctx, questData)
	// TODO 日志
	// TODO 触发任务完成条件
}

func (m *UserQuestManager) ReceivedQuestsReward(ctx cd.RpcContext, questIDs []int32) (rewards []*public_protocol_pbdesc.DuserQuestRewardData, result cd.RpcResult) {

	rewards = make([]*public_protocol_pbdesc.DuserQuestRewardData, 0)

	for _, qid := range questIDs {

		rewarditem, ok := m.ReceivedQuestReward(ctx, qid, false)
		if ok.IsOK() {
			rewards = append(rewards, &public_protocol_pbdesc.DuserQuestRewardData{
				QuestId:     qid,
				RewardItems: rewarditem,
			})
		}

	}
	// TODO  这里如何返回需要和客户端商议
	return rewards, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) ReceivedQuestReward(ctx cd.RpcContext, questID int32, autoReceived bool) (rewards []*public_protocol_common.DItemBasic, result cd.RpcResult) {
	// 任务是否存在
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {
		ctx.LogError("try to receive quest reward but quest config not found",
			"quest_id", questID)
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 检查任务状态
	if m.existQuestIDs[questID] != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE {
		ctx.LogError("try to receive quest reward but quest not completed",
			"quest_id", questID)
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	questData, ok := m.quests.CompletedQuests[questID]
	if !ok || questData == nil {
		// 不存在完成任务数据 如果走到这里说明逻辑有问题
		ctx.LogError("try to receive quest reward but completed quest data not found",
			"quest_id", questID)
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	delete(m.quests.CompletedQuests, questID)

	// 插入到已领取任务队列
	questData.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE
	questData.ReceivedTime = ctx.GetNow().Unix()
	m.quests.ReceivedQuests[questID] = questData

	// 任务状态已领取
	m.existQuestIDs[questID] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE

	questReward := questCfg.GetRewards()

	// 发放奖励
	rewards, result = m.grantQuestReward(ctx, questID, questReward)
	m.addDirtyQuestData(ctx, questData)

	m.questDataLog(ctx, questData, int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE),
		int(public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE))

	unlockMgr := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if unlockMgr != nil {
		unlockMgr.OnUserUnlockDataChange(ctx, public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestReceived,
			0, int64(questID))
	}

	if autoReceived {
		// 添加到dirty auto reward
		m.addDirtyAutoReward(ctx, questID, rewards)
		return nil, result
	}

	return rewards, result
}

// grantQuestReward 发放任务奖励的辅助函数.
func (m *UserQuestManager) grantQuestReward(ctx cd.RpcContext, questID int32,
	questReward *public_protocol_config.Readonly_DQuestReward,
) (rewards []*public_protocol_common.DItemBasic, result cd.RpcResult) {
	if questReward == nil || len(questReward.GetItems()) == 0 {
		ctx.LogDebug("quest has no reward items, skip reward granting",
			"quest_id", questID,
		)
		return nil, cd.CreateRpcResultOk()
	}

	rewardOffsets := questReward.GetItems()
	rewardItemInsts, result := m.GetOwner().GenerateMultipleItemInstancesFromCfgOffset(ctx, rewardOffsets, false)
	if result.IsError() {
		ctx.LogError("generate quest reward items failed",
			"quest_id", questID,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return nil, result
	}

	addGuards, result := m.GetOwner().CheckAddItem(ctx, rewardItemInsts)
	if result.IsError() {
		ctx.LogError("check add quest reward failed",
			"quest_id", questID,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return nil, result
	}

	itemFlowReason := &data.ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_QUEST),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_QUEST_REWARD),
		Parameter:   int64(questID),
	}

	result = m.GetOwner().AddItem(ctx, addGuards, itemFlowReason)
	if !result.IsOK() {
		ctx.LogError("add quest reward items failed",
			"quest_id", questID,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return nil, result
	}

	ctx.LogInfo("quest reward items granted successfully",
		"quest_id", questID,
		"item_count", len(rewardItemInsts),
	)

	rewardBasic := make([]*public_protocol_common.DItemBasic, 0, len(rewardItemInsts))
	for _, itemInst := range rewardItemInsts {
		rewardBasic = append(rewardBasic, itemInst.GetItemBasic())
	}

	// TODO 日志

	return rewardBasic, cd.CreateRpcResultOk()
}

// ===== 脏数据同步 =====

func (m *UserQuestManager) addDirtyQuestData(ctx cd.RpcContext, questData *public_protocol_pbdesc.DQuestData) {
	m.registerQuestDirtyHandle()

	if m.dirtyQuestEvent == nil {
		m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DQuestData)
	}

	if questData == nil {
		return
	}

	m.dirtyQuestEvent[questData.QuestId] = questData
}

func (m *UserQuestManager) addDirtyQuestExpriedData(_ cd.RpcContext, questID int32) {
	m.registerQuestDirtyHandle()
	if m.dirtyExpriedQuestIDs == nil {
		m.dirtyExpriedQuestIDs = []int32{}
	}
	m.dirtyExpriedQuestIDs = append(m.dirtyExpriedQuestIDs, questID)
}

func (m *UserQuestManager) addDirtyAutoReward(_ cd.RpcContext, questID int32, rewards []*public_protocol_common.DItemBasic) {
	m.registerQuestDirtyHandle()
	if m.dirtyAutoReward == nil {
		m.dirtyAutoReward = make(map[int32][]*public_protocol_common.DItemBasic)
	}
	m.dirtyAutoReward[questID] = rewards
}

func (m *UserQuestManager) registerQuestDirtyHandle() {
	if m == nil {
		return
	}
	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
			return m.dumpQuestDirtyData(ctx, dirty)
		},
		func(ctx cd.RpcContext) {
			m.clearQuestDirtyData(ctx)
		},
	)
}

// 导出脏任务数据.
func (m *UserQuestManager) dumpQuestDirtyData(ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
	if m == nil || (len(m.dirtyQuestEvent) == 0 && len(m.dirtyExpriedQuestIDs) == 0 && len(m.dirtyAutoReward) == 0) {
		return false
	}

	// 遍历所有脏任务事件，添加到脏数据消息中
	events := dirty.MutableNormalDirtyChangeMessage().MutableDirtyQuestEvents()
	for _, questEvent := range m.dirtyQuestEvent {
		events.DirtyQuests = append(events.DirtyQuests, questEvent)
	}
	for _, questID := range m.dirtyExpriedQuestIDs {
		events.DeletedQuestIds = append(events.DeletedQuestIds, questID)
	}

	autoRewardEvents := dirty.MutableNormalDirtyChangeMessage().MutableAutoRewards()

	for questID, autoReward := range m.dirtyAutoReward {
		autoRewardEvents.Rewards = append(autoRewardEvents.Rewards,
			&public_protocol_common.DItemAutoReward{
				RewardItems: autoReward,
				ReasonType: &public_protocol_common.DItemAutoReward_QuestId{
					QuestId: questID,
				},
			})
	}

	return true
}

// 清理脏任务数据标记.
func (m *UserQuestManager) clearQuestDirtyData(_ cd.RpcContext) {
	if m == nil {
		return
	}
	m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DQuestData)
	m.dirtyExpriedQuestIDs = []int32{}
	m.dirtyAutoReward = make(map[int32][]*public_protocol_common.DItemBasic)
}

func (m *UserQuestManager) GetQuestExpiredTime(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) int64 {
	endPoint := int64(0)
	if questCfg.GetAvailablePeriod() != nil {
		switch questCfg.GetAvailablePeriod().GetValueOneofCase() {
		case public_protocol_config.DQuestAvailablePeriodType_EnValueID_Timedesc:
			endPoint = ctx.GetNow().Unix() + questCfg.GetAvailablePeriod().GetTimedesc().GetSeconds()

		case public_protocol_config.DQuestAvailablePeriodType_EnValueID_SpecificPeriod:
			endPoint = questCfg.GetAvailablePeriod().GetSpecificPeriod().GetEnd().GetSeconds()
		}
	}
	return endPoint
}

func (m *UserQuestManager) GetQuestResetTime(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList) int64 {

	periodDays := questCfg.GetProgressResetPeriod().GetPeriodDays()
	if periodDays <= 0 {
		return 0
	}
	dayStartTimepoint := questCfg.GetProgressResetPeriod().GetStartDayResetTimepoint().GetSeconds()
	if periodDays == logic_quest.InitLoginDays && dayStartTimepoint == 0 {

		dayStartTimepoint = logic_time.GetTodayStartTimepoint(nil).Unix() // TODO 默认的一开开始时间逻辑
	}
	// logic_quest.GetDayStartTimepoint(ctx.GetNow().Unix())

	now := ctx.GetNow().Unix()
	resetTimepoint := dayStartTimepoint
	if now > dayStartTimepoint {
		// 计算下一个重置时间点
		periodSec := int64(periodDays) * logic_quest.DaySeconds
		diff := now - dayStartTimepoint
		resetTimepoint = now + (periodSec - (diff % periodSec))
	}
	return resetTimepoint
}

// func (m *UserQuestManager) deleteExpriedDeletequestCache(ctx cd.RpcContext) {
// 	now := ctx.GetNow().Unix()

// 	deleteCacheKeepSeconds := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetDeleteQuestCacheTime().GetSeconds()
// 	if deleteCacheKeepSeconds <= logic_quest.DeleteCacheKeepSeconds {
// 		deleteCacheKeepSeconds = logic_quest.DeleteCacheKeepSeconds
// 	}

// 	for questID, deleteCache := range m.quests.MutableDeleteCache() {
// 		if now-deleteCache.GetDeleteTimepoint() > logic_quest.DeleteCacheKeepSeconds {
// 			delete(m.quests.MutableDeleteCache(), questID)
// 			ctx.LogDebug("delete expired quest delete cache",
// 				"quest_id", questID,
// 			)
// 		}
// 	}
// }

func (m *UserQuestManager) deleteQuestByStatusInner(ctx cd.RpcContext, quest_id int32, status public_protocol_common.EnQuestStatus) {
	switch status {
	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING:
		delete(m.quests.ReceivedQuests, quest_id)
	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE:
		delete(m.quests.CompletedQuests, quest_id)
	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE:
		delete(m.quests.ProgressingQuests, quest_id)
	case public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED:
		// 删除指定的 quest_id 元素
		for i, id := range m.quests.ExpiredQuestsID {
			if id == quest_id {
				m.quests.ExpiredQuestsID = slices.Delete(m.quests.ExpiredQuestsID, i, i+1)
				break
			}
		}
	default:
		ctx.LogError("unknown quest status in delete quest by status",
			"quest_id", quest_id,
			"status", status)
		return
	}
}

func (m *UserQuestManager) resetQuest(ctx cd.RpcContext, questCfg *public_protocol_config.Readonly_ExcelQuestList, resetByActiva bool) {
	// 删除之前任务数据

	if m.existQuestIDs[questCfg.GetId()] == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED && !resetByActiva {
		// 过期状态只能通过激活重置
		return
	}
	m.deleteQuestByStatusInner(ctx, questCfg.GetId(), public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE)
	m.deleteQuestByStatusInner(ctx, questCfg.GetId(), public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE)
	m.deleteQuestByStatusInner(ctx, questCfg.GetId(), public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED)
	m.questResetList.Remove(questCfg.GetId())
	m.questExpriedList.Remove(questCfg.GetId())

	ctx.LogInfo("questDataLog reset quest",
		"quest_id", questCfg.GetId(),
	)

	// 重新添加任务
	m.AddQuest(ctx, questCfg)
}

func (m *UserQuestManager) cleanUpDeleteCache(ctx cd.RpcContext) {
	pedding_delete_quest_ids := []int32{}
	for questID, deleteCache := range m.quests.DeleteCache {
		if deleteCache.GetDeleteTimepoint()+logic_quest.DeleteCacheKeepSeconds < ctx.GetNow().Unix() {
			pedding_delete_quest_ids = append(pedding_delete_quest_ids, questID)
		}
	}

	for _, questID := range pedding_delete_quest_ids {
		delete(m.quests.DeleteCache, questID)
		ctx.LogDebug("delete expired quest delete cache",
			"quest_id", questID,
		)
	}
}

func (m *UserQuestManager) ActivateQuest(ctx cd.RpcContext, questID int32) cd.RpcResult {
	// 激活任务
	result, ok := m.existQuestIDs[questID]

	if ok && !(result == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED ||
		result == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK) {
		ctx.LogError("try to activate quest but quest already exists",
			"quest_id", questID,
			"status", result)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_QUEST_ALREADY_ACCEPTED)
	}

	// 删除之前可能遗留的数据
	// if(ok && result == public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_EXPIRED) {
	// 	m.questResetList.Remove(questID)
	// 	m.questExpriedList.Remove(questID)
	// }

	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {
		ctx.LogError("try to activate quest but quest config not found",
			"quest_id", questID)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	m.resetQuest(ctx, questCfg, true)
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) questDataLog(ctx cd.RpcContext, questData *public_protocol_pbdesc.DQuestData, Pre, now int) {
	ctx.LogDebug("questDataLog quest data",
		"quest_id", questData.QuestId,
		"status", questData.Status,
		"expiredTime", questData.ExpiredTime,
		"resetTime", questData.ResetTime,
		"createdTime", questData.CreatedTime,
		"completedTime", questData.CompletedTime,
		"PreStatus", Pre,
		"nowStatus", now,
	)
	for _, progress := range questData.Progress {
		ctx.LogDebug("   quest progress data",
			"quest_id", questData.QuestId,
			"progress_unique_id", progress.UniqueId,
			"progress_value", progress.Value,
			"progress_status", progress.Status,
			"progress_createdTime", progress.CreatedTime,
		)
	}
}

func (m *UserQuestManager) ClientQueryQuestUpdateStatus(ctx cd.RpcContext) {
	m.RefreshLimitSecond(ctx)
}

// 通用条件

func checkRuleQuestStatus(m logic_condition.UserConditionManager, _ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, _ *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("UserCharacterManager not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	nowStatus := mgr.QueryQuestStatus(int32(rule.GetQuestStatus().GetQuestId()))
	if nowStatus != rule.GetQuestStatus().GetStatus() {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_QUEST_STATUS_NOT_MATCH)
	}
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GMForceUnlockQuest(ctx cd.RpcContext, questID int32) cd.RpcResult {
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {
		ctx.LogError("try to force unlock quest but quest config not found",
			"quest_id", questID)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	m.resetQuest(ctx, questCfg, false)
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GMForceFinishQuest(ctx cd.RpcContext, questID int32) cd.RpcResult {
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questID)
	if questCfg == nil {
		ctx.LogError("try to force unlock quest but quest config not found",
			"quest_id", questID)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	status, ok := m.existQuestIDs[questID]
	if !ok || status != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING {
		ctx.LogError("try to force finish quest but quest not in processing status",
			"quest_id", questID,
			"status", status)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_QUEST_STATUS_NOT_MATCH)
	}

	questData := m.quests.ProgressingQuests[questID]
	if questData == nil {
		ctx.LogError("try to force finish quest but quest data not found in processing quests",
			"quest_id", questID)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 直接将任务完成

	questData.Progress = []*public_protocol_pbdesc.DUserQuestProgressData{}

	for _, progressCfg := range questCfg.GetProgress() {
		Progress := public_protocol_pbdesc.DUserQuestProgressData{
			UniqueId: progressCfg.GetUniqueId(),
			Value:    progressCfg.GetValue(),
			// Status:      public_protocol_common.EnQuestProgressStatus_EN_QUEST_PROGRESS_STATUS_COMPLETE,
			// CreatedTime: ctx.GetNow().Unix(),
			// UniqueCount: make(map[int64]bool),
		}
		questData.Progress = append(questData.Progress, &Progress)
	}

	m.FinishQuest(ctx, questID, true)
	return cd.CreateRpcResultOk()
}
