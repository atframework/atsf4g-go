package lobbysvr_logic_quest_internal

import (
	"reflect"
	"slices"
	"sort"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"

	config "github.com/atframework/atsf4g-go/component-config"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_quest_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest/data"
)

func init() {
	var _ logic_quest.UserQuestManager = (*UserQuestManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_quest.UserQuestManager](func(_ctx *cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserQuestManager(owner)
	})

	data.RegisterUserItemManagerCreator([]data.UserItemTypeIdRange{
		data.MakeUserItemTypeIdRange(
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_QUEST_BEGIN),
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_QUEST_END)),
	}, func(ctx *cd.RpcContext, owner *data.User) data.UserItemManagerImpl {
		mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](owner)
		if mgr == nil {
			ctx.LogError("can not find user quest manager", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		convert, ok := mgr.(data.UserItemManagerImpl)
		if !ok || convert == nil {
			ctx.LogError("user quest manager does not implement UserItemManagerImpl", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		return convert
	})

	registerCondition()
}

type EventQueueItem struct {
	eventType public_protocol_common.EnQuestTriggerType
	params    logic_quest.TriggerParams
}

type UserQuestManager struct {
	data.UserModuleManagerBase
	data.UserItemManagerBase

	quests private_protocol_pbdesc.UserQuestData
	// progressDealQueue []*
	// UnlockDealQueue []*

	dirtyQuestEvent map[int32]*public_protocol_pbdesc.DUserQuestEvent

	waitToAddQuestIds           []int32
	waitToCheckProgressQuestIds []int32
	waitToFinishQuestIds        []int32
	waitToDeleteQuestIds        []int32

	eventQueue      []EventQueueItem
	eventQueueGuard bool
}

func CreateUserQuestManager(owner *data.User) *UserQuestManager {
	mgr := &UserQuestManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
	}
	return mgr
}

func (m *UserQuestManager) GetOwner() *data.User {
	return m.UserItemManagerBase.GetOwner()
}

func registerCondition() {
	// Register logic conditions
}

// 道具相关api实现

func (m *UserQuestManager) AddItem(ctx *cd.RpcContext, itemOffset []data.ItemAddGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) CheckAddItem(ctx *cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]data.ItemAddGuard, data.Result) {
	return m.CreateItemAddGuard(itemOffset)
}

func (m *UserQuestManager) SubItem(ctx *cd.RpcContext, itemOffset []data.ItemSubGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) CheckSubItem(ctx *cd.RpcContext, itemOffset []*public_protocol_common.DItemBasic) ([]data.ItemSubGuard, data.Result) {
	return m.CreateItemSubGuard(itemOffset)
}

func (m *UserQuestManager) ForeachItem(fn func(item *public_protocol_common.DItemInstance) bool) {
}

func (m *UserQuestManager) GenerateItemInstanceFromBasic(ctx *cd.RpcContext, itemOffset *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GenerateItemInstanceFromOffset(ctx *cd.RpcContext, itemOffset *public_protocol_common.DItemOffset) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GetTypeStatistics(typeId int32) *data.ItemTypeStatistics {
	return nil
}

func (m *UserQuestManager) GetItemFromBasic(itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultOk()
}

func (m *UserQuestManager) GetNotEnoughErrorCode(typeId int32) int32 {
	return 0
}

func (m *UserQuestManager) CheckTypeIdValid(typeId int32) bool {
	return true
}

// db load & save

func (m *UserQuestManager) InitFromDB(ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {

	m.quests = *dbUser.GetQuestData().Clone()
	return cd.CreateRpcResultOk()
}
func (m *UserQuestManager) DumpToDB(_ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	if dbUser == nil {
		return cd.CreateRpcResultOk()
	}
	dbUser.QuestData = m.quests.Clone()
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) LoginInit(_ctx *cd.RpcContext) {
	m.OnResourceVersionChanged(_ctx)
}

func (m *UserQuestManager) RefreshLimitSecond(_ctx *cd.RpcContext) {
	// TODO implement Jijunliang
}

func (m *UserQuestManager) QueryQuestStatus(questID int32) public_protocol_common.EnQuestStatus {
	questStatus, ok := m.quests.ExistQuestIds[questID]
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

// 任务触发
func (m *UserQuestManager) QuestTriggerEvent(_ctx *cd.RpcContext, triggerType public_protocol_common.EnQuestTriggerType, param logic_quest.TriggerParams) {
	m.eventQueue = append(m.eventQueue, EventQueueItem{
		eventType: triggerType,
		params:    param,
	})
	if m.eventQueueGuard {
		return
	}
	m.eventQueueGuard = true
	defer func() {
		m.eventQueueGuard = false
	}()

	// 清理可能无效的任务
	now := _ctx.GetNow()
	m.CleanUpExpiredQuests(_ctx, now)
	for len(m.eventQueue) > 0 {
		eventItem := m.eventQueue[0]
		m.eventQueue = m.eventQueue[1:]
		// 处理事件
		m.TriggerEventInner(_ctx, eventItem)
	}
}

func (m *UserQuestManager) TriggerEventInner(_ctx *cd.RpcContext, eventItem EventQueueItem) {
	triggerCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestTriggerEventType.GetById(int32(eventItem.eventType))
	if triggerCfg == nil {
		_ctx.LogError("quest trigger config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "trigger_type", eventItem.eventType)
		return
	}
	// 新任务解锁
	for _, unlockType := range triggerCfg.GetUnlockConditionTypes() {
		m.TryUnlockQuestByType(_ctx, unlockType, eventItem.params)
	}
	// 任务重置(周期任务)
	m.StartCheckPeriodQuestRest(_ctx, int32(eventItem.eventType), eventItem.params)

	//任务进度更新

	for _, pregressType := range triggerCfg.GetProgressTypes() {
		m.UpdateQuestProgressByType(_ctx, eventItem.eventType, pregressType, eventItem.params)
	}
}

func (m *UserQuestManager) UpdateQuestProgressByType(_ctx *cd.RpcContext, triggerType public_protocol_common.EnQuestTriggerType,
	pregressType public_protocol_common.EnQuestProgressType, params logic_quest.TriggerParams) {
	// 根据触发类型得到所有可能需要更新进度
	questProgressList := m.GetQuestProgressListByType(pregressType)

	pendingFinishQuestIds := []int32{}
	for _, questProgress := range questProgressList.GetQuestProgressList() {
		_ctx.LogDebug("updating quest progress", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
			"progress_type", pregressType, "quest_id", questProgress.GetQuestId())

		if params.HasSpecifyQuestID && questProgress.GetQuestId() != params.SpecifyQuestID {
			continue
		}
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestList.GetById(questProgress.GetQuestId())

		if questCfg == nil {
			_ctx.LogError("quest config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
				"quest_id", questProgress.GetQuestId())
			continue
		}

		// 检查任务是否有效
		if m.CheckQuestInvalid(questCfg) {
			continue
		}

		// 检查通用条件
		if !m.CheckQuestCommonCondition(_ctx, questCfg) {
			continue
		}

		// 更新任务进度
		// m.UpdateQuestProgressInner(_ctx, questCfg, questProgress, triggerType, pregressType, params)
		for _, progressCfg := range questCfg.GetProgress() {
			if progressCfg.GetTypeId() != pregressType {
				continue
			}
			origin_value := questProgress.Value
			m.AddquestProgressInner(_ctx, questCfg, questProgress, triggerType, pregressType, params, progressCfg)
			_ctx.LogDebug("quest progress updated", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
				"quest_id", questProgress.GetQuestId(), "progress_type", pregressType,
				"origin_value", origin_value, "new_value", questProgress.Value)

			if origin_value != questProgress.Value && m.CheckQuestComplete(_ctx, progressCfg, questProgress) {
				// 任务已经完成
				pendingFinishQuestIds = append(pendingFinishQuestIds, questProgress.GetQuestId())
			}
		}
	}
	m.FinishQuests(_ctx, pendingFinishQuestIds, false)
}

func (m *UserQuestManager) AddquestProgressInner(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList,
	questProgress *public_protocol_pbdesc.DUserQuestData, triggerType public_protocol_common.EnQuestTriggerType,
	pregressType public_protocol_common.EnQuestProgressType, params logic_quest.TriggerParams,
	progressCfg *public_protocol_config.DQuestConditionProgress) {

	progressTypeCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestProgressType.GetById(int32(pregressType))
	if progressTypeCfg == nil {
		_ctx.LogError("quest progress type config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
			"progress_type", pregressType)
		return
	}

	for _, conditionId := range progressCfg.GetConditionIds() {
		conditionCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestProgressCondition.GetById(int32(conditionId))
		if conditionCfg == nil {
			_ctx.LogError("quest progress condition config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
				"progress_type", pregressType)
			return
		}

		if !slices.Contains(progressTypeCfg.GetSupporProgressConditionTypes(), conditionId) {
			_ctx.LogError("quest progress condition type not supported by progress type", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
				"progress_type", pregressType, "condition_type", conditionId)
			return
		}

		// 是否满足过滤器
		progressFilter := logic_quest_data.GetQuestProgressFilterUpdateHandler(reflect.TypeOf(conditionCfg.Value.Limit))
		if progressFilter != nil {
			ok := progressFilter(_ctx, params, progressCfg, questProgress)
			if !ok.IsOK() {
				return
			}
		}
	}

	progressHander := logic_quest_data.GetQuestProgressHandler(pregressType)
	if progressHander.UpdateHandler == nil {
		_ctx.LogError("quest progress handler UpdateHandler not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
			"progress_type", pregressType)
		return
	}
	progressHander.UpdateHandler(_ctx, params, progressCfg, questProgress)

	// 日志
	// 脏数据同步
}

func (m *UserQuestManager) CheckQuestCommonCondition(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) bool {
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
		_ctx.LogError("failed to get UserConditionManager", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId())
		return false
	}

	// 检查所有条件（包括静态和动态）
	rpcResult := conditionMgr.CheckBasicLimit(_ctx, commonCondition, logic_condition.CreateRuleCheckerRuntime())
	return rpcResult.IsOK()
}

func (m *UserQuestManager) GetQuestProgressListByType(progressType public_protocol_common.EnQuestProgressType) *public_protocol_pbdesc.DQuestProgressDataList {
	return m.quests.MutableUserQuestList().MutableProcessingQuests()[int32(progressType)]
}

func (m *UserQuestManager) TryUnlockQuestByType(_ctx *cd.RpcContext, unlockType int32, params logic_quest.TriggerParams) {
	// 按照解锁的类型找到所有可能进行解锁的任务
	TryUnlockQuestIds := config.GetBoundUnlockQuestIds(config.GetConfigManager().GetCurrentConfigGroup(), unlockType, params.X, params.Y)

	for _, questId := range TryUnlockQuestIds {
		// 正常来说这里的任务玩家不可能已经解锁
		// 但策划可以随便改表
		questStatus := m.quests.ExistQuestIds[questId]
		if questStatus != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK {
			continue
		}
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestList.GetById(questId)
		if m.CheckQuestIsUnlock(_ctx, questCfg) {
			m.AddQuest(_ctx, questCfg)
		}
	}
}

func (m *UserQuestManager) StartCheckPeriodQuestRest(_ctx *cd.RpcContext, unlockType int32, params logic_quest.TriggerParams) {
	// 只能由时间触发
	if unlockType != int32(public_protocol_common.EnQuestTriggerType_EN_QUEST_TRIGGER_TYPE_TASK_TIME_TICK) {
		return
	}

	// 不考虑回退时间导致的一系列问题，以后有必要再说
	// now := _ctx.GetNow()
	reset_index := m.quests.MutableProgressResetData().MutableResetEntrys()
	if !m.quests.GetProgressResetData().GetIsChanged() {
		// 排序
		sort.SliceStable(reset_index, func(i, j int) bool {
			return int32(reset_index[i].ResetTimepoint) < int32(reset_index[j].ResetTimepoint)
		})
		m.quests.MutableProgressResetData().IsChanged = true
		m.quests.MutableProgressResetData().ResetEntrys = reset_index
	}

	m.CheckPeriodQuestRestDeal(_ctx, params.X)

}

func (m *UserQuestManager) CheckPeriodQuestRestDeal(_ctx *cd.RpcContext, now int64) {
	reset_index := m.quests.MutableProgressResetData().MutableResetEntrys()
	for _, resetEntry := range reset_index {
		questCfg := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestList.GetById(resetEntry.QuestId)
		if questCfg == nil {
			_ctx.LogError("quest config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", resetEntry.QuestId)
			continue
		}
		if questCfg.GetProgressResetPeriod().GetPeriodDays() == 0 {
			_ctx.LogError("quest reset period days is zero", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", resetEntry.QuestId)
			continue
		}

		if m.CheckQuestInvalid(questCfg) {
			continue
		}

		if resetEntry.ResetTimepoint < now {
			continue
		}
		// 开始重置
		m.PeriodQuestRestDeal(_ctx, questCfg, resetEntry, now)
	}
}

func (m *UserQuestManager) PeriodQuestRestDeal(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList, resetEntry *private_protocol_pbdesc.UserQuestProgressResetEntry, now int64) {

	// 重置下次刷新时间
	period_sec := int64(questCfg.GetProgressResetPeriod().GetPeriodDays()) * logic_quest.DaySeconds
	if now < resetEntry.GetResetTimepoint() {
		// 必定是resetEntry.GetResetTimepoint()大于数个周期的情况
		if (resetEntry.GetResetTimepoint()-now)%period_sec == 0 {
			resetEntry.ResetTimepoint = now + period_sec
		} else {
			resetEntry.ResetTimepoint = resetEntry.GetResetTimepoint() -
				(resetEntry.GetResetTimepoint()-now)/period_sec*period_sec
		}
	} else if now == resetEntry.GetResetTimepoint() {
		// 加上一个周期数
		resetEntry.ResetTimepoint = resetEntry.GetResetTimepoint() + period_sec
	} else {
		// 要把周期数都补上
		resetEntry.ResetTimepoint = resetEntry.GetResetTimepoint() +
			(now-resetEntry.GetResetTimepoint()+period_sec)/period_sec*period_sec
	}
	// 重置任务

	// oldStatus := m.quests.ExistQuestIds[questCfg.GetId()]
	receivedQuestsList := m.quests.GetUserQuestList().GetReceivedQuests()
	completeQuestList := m.quests.GetUserQuestList().GetCompletedQuests()
	receivedQuest := receivedQuestsList[questCfg.GetId()]
	if receivedQuest != nil {
		delete(receivedQuestsList, questCfg.GetId())
	}
	completeQuest := completeQuestList[questCfg.GetId()]
	if completeQuest != nil {
		delete(completeQuestList, questCfg.GetId())
	}

	if m.QuestHasNoProgress(questCfg) {
		// 无进度直接完成
		m.FinishQuest(_ctx, questCfg.GetId(), true)
		return
	}

	for _, progressCfg := range questCfg.GetProgress() {
		if progressCfg.GetTypeId() == 0 {
			_ctx.LogError("progress type id is zero", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId())
			return
		}
		// 重置进度
		progressMp := m.quests.MutableUserQuestList().MutableProcessingQuests()[int32(progressCfg.GetTypeId())]
		questprogressValue := progressMp.GetQuestProgressList()[questCfg.GetId()]

		questprogressValue.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING
		questprogressValue.Value = 0
		questprogressValue.CreatedTime = time.Now().Unix()
		questprogressValue.QuestId = questCfg.GetId()
		questprogressValue.UniqueCount = make(map[int64]bool)
	}

	// 标记脏数据 - 任务进度已重置
	m.addQuestEventProgressUpdate(_ctx, questCfg.GetId(), &public_protocol_pbdesc.DUserQuestData{
		QuestId: questCfg.GetId(),
		Status:  public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING,
	}, public_protocol_common.EnQuestProgressType(questCfg.GetProgress()[0].GetTypeId()))

}

func (m *UserQuestManager) QuestHasNoProgress(questCfg *public_protocol_config.ExcelQuestList) bool {
	return len(questCfg.GetProgress()) == 0
}

func (m *UserQuestManager) CheckQuestInvalid(QuestCfg *public_protocol_config.ExcelQuestList) bool {
	// 判断任务是合法 CleanUpQuestIsInvalid 整合一下可以
	// quest_config->available_period().value_case()
	return true
}

// 资源版本变化时，检查任务解锁和完成状态
func (m *UserQuestManager) OnResourceVersionChanged(_ctx *cd.RpcContext) {
	// 登录时资源变化需要重新判断未解锁的任务的解锁&&进行中任务的完成状态
	now := _ctx.GetNow()
	// TODO(建个索引)
	AllQuest := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().QuestSequence
	for _, questCfg := range AllQuest {
		if questCfg == nil {
			continue
		}

		m.CleanUpQuestIsInvalid(_ctx, questCfg) // 任务已经非法
		m.QueryQuestIsFinish(questCfg.GetId())  //  任务是否已经完成

		if questCfg.GetOn() == false {
			continue
		}

		questStatus := m.quests.ExistQuestIds[questCfg.GetId()]
		if questStatus != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK {
			continue
		}
		m.quests.ExistQuestIds[questCfg.GetId()] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_LOCK

		// 时间条件
		if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
			if now.Unix() < questCfg.GetAvailablePeriod().GetSpecificPeriod().Start.GetSeconds() || now.Unix() > questCfg.GetAvailablePeriod().GetSpecificPeriod().End.GetSeconds() {
				_ctx.LogDebug("quest is not in available period", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId())
				continue
			}
		}
		// 检查所有解锁条件
		isUnlock := m.CheckQuestIsUnlock(_ctx, questCfg)

		if isUnlock {
			m.AddQuest(_ctx, questCfg)
		}

		// TODO 版本号管理，现在还没版本号不记录
		// m.quests.UserQuestList.ExcelVersion = config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestList

	}
}

func (m *UserQuestManager) CheckQuestIsUnlock(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) bool {
	if len(questCfg.GetUnlockConditions()) == 0 {
		// 无解锁条件，默认解锁
		return true
	}

	if len(questCfg.GetUnlockConditions()) == 1 && questCfg.GetUnlockConditions()[0] == nil {
		return true

	}

	questUnlockHandler := logic_quest_data.GetQuestUnlockHandle()
	for _, cond := range questCfg.GetUnlockConditions() {
		if questUnlockHandler[reflect.TypeOf(cond.GetUnlockType())] == nil {
			_ctx.LogError("quest unlock handler not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId(), "condition", cond)
			return false
		}

		rpcResult := (*questUnlockHandler[reflect.TypeOf(cond.GetUnlockType())])(_ctx, cond, m.GetOwner())
		if !rpcResult.IsOK() {
			_ctx.LogDebug("quest unlock condition not met", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId(), "condition", cond, "error", rpcResult.Error)
			return false
		}
	}
	return true
}

// 检查任务是否已经下架或者失效
func (m *UserQuestManager) CleanUpQuestIsInvalid(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) {
	// 清理过去和下架的任务
	if questCfg.GetOn() == false {
		m.DeleteQuestForce(_ctx, questCfg)
		_ctx.LogInfo("quest is off, delete quest", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId())
	}

	// 删除过期任务
	if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
		if _ctx.GetNow().Unix() > questCfg.GetAvailablePeriod().GetSpecificPeriod().End.GetSeconds() {
			m.DeleteQuestForce(_ctx, questCfg)
			_ctx.LogInfo("quest is expired, delete quest", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId())
		}
	}
}

// 检查任务进度是否已经满足完成条件
func (m *UserQuestManager) CheckQuestProgressValue(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) {

}

// 强制删除任务
func (m *UserQuestManager) DeleteQuestForce(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) {

}

func (m *UserQuestManager) AddQuest(_ctx *cd.RpcContext, questCfg *public_protocol_config.ExcelQuestList) {
	questId := questCfg.GetId()
	if m.QuestHasNoProgress(questCfg) {
		// 任务默认解锁就完成
		m.FinishQuest(_ctx, questId, true)
		return
	}

	// 判断下是否解锁就完成了
	peddingAddProgress := []*public_protocol_pbdesc.DUserQuestData{}
	peddingAddFinishQuestIds := []int32{}
	for _, progressCfg := range questCfg.GetProgress() {
		if progressCfg.GetTypeId() == 0 {
			_ctx.LogError("progress type id is zero", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId())
			return
		}
		// 获得初始化赋值
		Progress := public_protocol_pbdesc.DUserQuestData{}
		Progress.Status = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING
		questProgressHandler := logic_quest_data.GetQuestProgressHandler(progressCfg.GetTypeId())

		if questProgressHandler.InitHandler != nil {
			rpcResult := questProgressHandler.InitHandler(_ctx, progressCfg, &Progress, m.GetOwner())
			if !rpcResult.IsOK() {
				_ctx.LogError("init quest progress value failed", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questCfg.GetId(), "progress_type", progressCfg.GetTypeId(), "error", rpcResult.Error)
				return
			}
		}
		Progress.CreatedTime = time.Now().Unix()
		Progress.QuestId = questId
		Progress.UniqueCount = make(map[int64]bool)
		if m.CheckQuestComplete(_ctx, progressCfg, &Progress) {
			// 任务解锁就完成
			peddingAddFinishQuestIds = append(peddingAddFinishQuestIds, questId)
			return
		} else {
			// 加入任务进度待添加列表
			peddingAddProgress = append(peddingAddProgress, &Progress)
		}
	}

	// 插入到任务进度里面
	for _, progressData := range peddingAddProgress {
		processingQuestsList := m.quests.MutableUserQuestList().MutableProcessingQuests()[int32(questCfg.GetProgress()[0].GetTypeId())]
		processingQuestsList.GetQuestProgressList()[questId] = progressData
	}

	// 插入到各种索引里面
	m.insertQuestExpriedIdx(questCfg)
	m.insertQuestProgressResetIdx(questCfg)
	m.insertQuestExsitIdx(questCfg, public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_PROCESSING)

	if len(peddingAddFinishQuestIds) > 0 {
		m.FinishQuests(_ctx, peddingAddFinishQuestIds, true)
	}

	// 标记脏数据 - 任务解锁
	for _, progressData := range peddingAddProgress {
		m.addQuestEventUnlock(_ctx, questId, progressData,
			public_protocol_common.EnQuestProgressType(questCfg.GetProgress()[0].GetTypeId()))
	}

}

func (m *UserQuestManager) insertQuestExpriedIdx(questCfg *public_protocol_config.ExcelQuestList) {
	// 插入任务过期索引
	if questCfg.GetAvailablePeriod() != nil && questCfg.GetAvailablePeriod().GetSpecificPeriod() != nil {
		specificEndTimepointData := m.quests.MutableSpecificEndTimepointData()
		entry := &private_protocol_pbdesc.UserQuestSpecificEndTimepointEntry{
			QuestId:      questCfg.GetId(),
			EndTimepoint: questCfg.GetAvailablePeriod().GetSpecificPeriod().End.GetSeconds(),
		}
		specificEndTimepointData.EndtimeEntrys = append(specificEndTimepointData.EndtimeEntrys, entry)
		specificEndTimepointData.IsChanged = true
	}
}

func (m *UserQuestManager) insertQuestProgressResetIdx(questCfg *public_protocol_config.ExcelQuestList) {
	// 插入任务进度重置索引 Check
	if questCfg.GetProgressResetPeriod() != nil && questCfg.GetProgressResetPeriod().GetPeriodDays() > 0 {
		progressResetData := m.quests.MutableProgressResetData()
		now := time.Now().Unix()
		entry := &private_protocol_pbdesc.UserQuestProgressResetEntry{
			QuestId:        questCfg.GetId(),
			ResetTimepoint: now + int64(questCfg.GetProgressResetPeriod().GetPeriodDays())*logic_quest.DaySeconds,
		}
		progressResetData.ResetEntrys = append(progressResetData.ResetEntrys, entry)
		progressResetData.IsChanged = true
	}
}

func (m *UserQuestManager) insertQuestExsitIdx(questCfg *public_protocol_config.ExcelQuestList, status public_protocol_common.EnQuestStatus) {
	// 插入任务存在索引
	m.quests.ExistQuestIds[questCfg.GetId()] = status
}

func (m *UserQuestManager) CheckQuestComplete(_ctx *cd.RpcContext, progressCfg *public_protocol_config.DQuestConditionProgress, progressData *public_protocol_pbdesc.DUserQuestData) bool {
	questProgessTypeConfig := config.GetConfigManager().GetCurrentConfigGroup().ExcelQuestProgressType.GetById(int32(progressCfg.GetTypeId()))
	if questProgessTypeConfig == nil {
		_ctx.LogError("quest progress type config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "progress_type_id", progressCfg.GetTypeId())
		return false
	}
	result := false

	switch questProgessTypeConfig.ValueCompareType {
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_AUTO_COMPLETE:
		result = true
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_GREATER_OR_EQUAL:
		result = progressData.GetValue() >= progressCfg.GetValue()
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_LESS_OR_EQUAL:
		result = progressData.GetValue() <= progressCfg.GetValue()
	case public_protocol_common.EnQuestProgressValueCompareType_EN_QUEST_PROGRESS_VALUE_COMPARE_TYPE_STRICTLY_EQUAL:
		result = progressData.GetValue() == progressCfg.GetValue()
	default:
		_ctx.LogError("unknown quest progress value compare type", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "compare_type", questProgessTypeConfig.ValueCompareType)
	}

	return result
}

func (m *UserQuestManager) CleanUpExpiredQuests(_ctx *cd.RpcContext, now time.Time) {
	expriedQuestIds := []int32{}

	specificEndTimepointData := m.quests.MutableSpecificEndTimepointData()
	entries := specificEndTimepointData.MutableEndtimeEntrys()
	if !m.quests.GetSpecificEndTimepointData().GetIsChanged() {
		// 对EndtimeEntrys进行排序
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].EndTimepoint < entries[j].EndTimepoint
		})
		// 将排序后的切片写回（MutableEndtimeEntrys 已返回底层切片，直接排序即可）
		specificEndTimepointData.IsChanged = true
	}

	for _, entry := range specificEndTimepointData.GetEndtimeEntrys() {
		if entry.EndTimepoint <= now.Unix() {
			// 任务已经结束
			expriedQuestIds = append(expriedQuestIds, entry.QuestId)
		} else {
			break
		}
	}

	sz := len(expriedQuestIds)
	if sz != 0 {
		// 删除过期任务
		m.DeleteExpiredQuests(_ctx, expriedQuestIds)

		// 如果全部删除，直接置空
		if sz >= len(entries) {
			specificEndTimepointData.EndtimeEntrys = []*private_protocol_pbdesc.UserQuestSpecificEndTimepointEntry{}
			return
		}
		specificEndTimepointData.EndtimeEntrys = specificEndTimepointData.EndtimeEntrys[sz:]
	}

}

func (m *UserQuestManager) DeleteExpiredQuests(_ctx *cd.RpcContext, questList []int32) {
	for _, questId := range questList {
		m.DeleteExpiredQuest(_ctx, questId)
	}
}

func (m *UserQuestManager) DeleteExpiredQuest(_ctx *cd.RpcContext, questId int32) {
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questId)
	if questCfg == nil {
		_ctx.LogWarn("try to delete quest but quest config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId)
		return
	}
	deleteSucess := false
	for _, cond := range questCfg.GetProgress() {
		processingQuestsList := m.quests.UserQuestList.ProcessingQuests[int32(cond.GetTypeId())]
		if processingQuestsList != nil {
			questData := processingQuestsList.GetQuestProgressList()[questId]
			if questData != nil {
				delete(processingQuestsList.GetQuestProgressList(), questId)
				deleteSucess = true
			}
		}
	}
	if deleteSucess {
		_ctx.LogDebug("quest deleted", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId)

		// 标记脏数据 - 任务已删除（通过添加 Received 事件表示任务状态变化）
		m.addQuestEventReceived(_ctx, questId)
	}
}

func (m *UserQuestManager) FinishQuests(_ctx *cd.RpcContext, questIds []int32, noProgress bool) {
	for _, questId := range questIds {
		m.FinishQuest(_ctx, questId, noProgress)
	}
}

func (m *UserQuestManager) FinishQuest(_ctx *cd.RpcContext, questId int32, noProgress bool) {
	// 走到这里的任务说明已经之前检查过完成条件，现在将任务从progressing状态转到finished状态
	// noProgress 没有任务条件直接完成的任务，
	if !noProgress {
		// 删除任务进度
		m.deleteQuestProgress(_ctx, questId)
	}

	// 插入完成任务队列
	completeQuestsList := m.quests.MutableUserQuestList().MutableCompletedQuests()
	completeQuestsList[questId] = &public_protocol_pbdesc.DUserQuestCompletedData{
		QuestId:   questId,
		Timepoint: time.Now().Unix(),
	}

	// 任务状态已完成
	m.quests.ExistQuestIds[questId] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE

	// 从存在任务列表中删除
	delete(m.quests.ExistQuestIds, questId)

	// 自动领取
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questId)
	if questCfg != nil && questCfg.GetRewards().GetGiveOutType() == public_protocol_config.EnQuestRewardGiveOutType_EN_QUEST_REWARD_GIVE_OUT_TYPE_AUTO_INVENTORY {
		err := m.ReceivedQuestReward(_ctx, questId, true)
		if err.IsError() {
			_ctx.LogError("auto receive quest reward failed", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId, "error", err.GetStandardError())
		}
	}

	// 标记脏数据 - 任务完成
	m.addQuestEventComplete(_ctx, questId)
	// TODO 日志
	// TODO 触发任务完成条件
}

func (m *UserQuestManager) deleteQuestProgress(_ctx *cd.RpcContext, questId int32) {
	for _, cond := range config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questId).GetProgress() {
		processingQuestsList := m.quests.UserQuestList.ProcessingQuests[int32(cond.GetTypeId())]
		if processingQuestsList != nil {
			questData := processingQuestsList.GetQuestProgressList()[questId]
			if questData != nil {
				delete(processingQuestsList.GetQuestProgressList(), questId)
			}
		}
	}
}

func (m *UserQuestManager) ReceivedQuestSReward(_ctx *cd.RpcContext, questId []int32, autoReceived bool) cd.RpcResult {
	for _, qid := range questId {
		m.FinishQuest(_ctx, qid, false)
	}
	// TODO  这里如何返回需要在考虑下
	return cd.CreateRpcResultOk()
}

func (m *UserQuestManager) ReceivedQuestReward(_ctx *cd.RpcContext, questId int32, autoReceived bool) cd.RpcResult {
	// 任务是否存在
	questCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelQuestListById(questId)
	if questCfg == nil {
		_ctx.LogError("try to receive quest reward but quest config not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 检查任务状态
	if m.quests.ExistQuestIds[questId] != public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE {
		_ctx.LogError("try to receive quest reward but quest not completed", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	_, ok := m.quests.GetUserQuestList().GetCompletedQuests()[questId]
	if !ok {
		// 不存在完成任务数据 如果走到这里说明逻辑有问题
		_ctx.LogError("try to receive quest reward but completed quest data not found", "zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(), "quest_id", questId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	delete(m.quests.GetUserQuestList().GetCompletedQuests(), questId)

	// 插入到已领取任务队列
	receivedQuestsList := m.quests.MutableUserQuestList().MutableReceivedQuests()
	receivedQuestsList[questId] = &public_protocol_pbdesc.DUserQuestReceivedData{
		QuestId:   questId,
		Timepoint: time.Now().Unix(),
	}

	// 任务状态已领取
	m.quests.ExistQuestIds[questId] = public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE

	questReward := questCfg.GetRewards()

	if questReward != nil && len(questReward.GetItems()) > 0 {
		// 说明是任务无奖励，可能是虚拟的触发任务
		_ctx.LogDebug("quest has no reward items, skip reward granting",
			"zone_id", m.GetOwner().GetZoneId(),
			"user_id", m.GetOwner().GetUserId(),
			"quest_id", questId,
		)
		return cd.CreateRpcResultOk()
	}

	// 发放奖励
	// 步骤 1: 生成奖励道具实例

	if questReward != nil && len(questReward.GetItems()) > 0 {
		rewardOffsets := questReward.GetItems()
		rewardItemInsts, result := m.GetOwner().GenerateMultipleItemInstancesFromOffset(_ctx, rewardOffsets)
		if result.IsError() {
			_ctx.LogError("generate quest reward items failed",
				"zone_id", m.GetOwner().GetZoneId(),
				"user_id", m.GetOwner().GetUserId(),
				"quest_id", questId,
				"error", result.GetStandardError(),
				"response_code", result.GetResponseCode(),
			)
			return result
		}

		// 步骤 2: 检查添加
		addGuards, result := m.GetOwner().CheckAddItem(_ctx, rewardItemInsts)
		if result.IsError() {
			_ctx.LogError("check add quest reward failed",
				"zone_id", m.GetOwner().GetZoneId(),
				"user_id", m.GetOwner().GetUserId(),
				"quest_id", questId,
				"error", result.GetStandardError(),
				"response_code", result.GetResponseCode(),
			)
			return result
		}

		// 步骤 3: 添加道具到背包
		itemFlowReason := &data.ItemFlowReason{
			// TODO: 道具流水原因
			// MajorReason: 1001, // QUEST_REWARD 任务奖励
			// MinorReason: 0,
			// Parameter:   int64(questId),
		}

		result = m.GetOwner().AddItem(_ctx, addGuards, itemFlowReason)
		if !result.IsOK() {
			_ctx.LogError("add quest reward items failed",
				"zone_id", m.GetOwner().GetZoneId(),
				"user_id", m.GetOwner().GetUserId(),
				"quest_id", questId,
				"error", result.GetStandardError(),
				"response_code", result.GetResponseCode(),
			)
			return result
		}

		_ctx.LogInfo("quest reward items granted successfully",
			"zone_id", m.GetOwner().GetZoneId(),
			"user_id", m.GetOwner().GetUserId(),
			"quest_id", questId,
			"item_count", len(rewardItemInsts),
		)
	}

	// 标记脏数据 - 添加任务事件到脏数据列表，用于推送给客户端
	// 事件类型：received - 任务奖励已领取
	m.addQuestEventReceived(_ctx, questId)

	// TODO 日志
	// TODO 触发任务完成条件

	return cd.CreateRpcResultOk()
}

// ===== 脏数据同步 =====

// addQuestEventUnlock - 添加"任务解锁"事件到脏数据
func (m *UserQuestManager) addQuestEventUnlock(_ctx *cd.RpcContext, questId int32, questData *public_protocol_pbdesc.DUserQuestData, progressType public_protocol_common.EnQuestProgressType) {
	m.registerQuestDirtyHandle()

	// 初始化脏事件 map
	if m.dirtyQuestEvent == nil {
		m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DUserQuestEvent)
	}

	// 创建 Unlock 类型事件
	questEvent := &public_protocol_pbdesc.DUserQuestEvent{
		EventId: _ctx.GetNow().UnixNano(),
		Event: &public_protocol_pbdesc.DUserQuestEvent_Unlock{
			Unlock: &public_protocol_pbdesc.DUserQuestProgressEvent{
				Data: questData,
				Type: progressType,
			},
		},
	}

	// 替换或添加到 map（自动覆盖旧事件）
	m.dirtyQuestEvent[questId] = questEvent
}

// addQuestEventProgressUpdate - 添加"进度更新"事件到脏数据
func (m *UserQuestManager) addQuestEventProgressUpdate(_ctx *cd.RpcContext, questId int32, questData *public_protocol_pbdesc.DUserQuestData, progressType public_protocol_common.EnQuestProgressType) {
	m.registerQuestDirtyHandle()

	// 初始化脏事件 map
	if m.dirtyQuestEvent == nil {
		m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DUserQuestEvent)
	}

	// 创建 ProgressUpdate 类型事件
	questEvent := &public_protocol_pbdesc.DUserQuestEvent{
		EventId: _ctx.GetNow().UnixNano(),
		Event: &public_protocol_pbdesc.DUserQuestEvent_ProgressUpdate{
			ProgressUpdate: &public_protocol_pbdesc.DUserQuestProgressEvent{
				Data: questData,
				Type: progressType,
			},
		},
	}

	// 替换或添加到 map（自动覆盖旧事件）
	m.dirtyQuestEvent[questId] = questEvent
}

// addQuestEventComplete - 添加"任务完成"事件到脏数据
func (m *UserQuestManager) addQuestEventComplete(_ctx *cd.RpcContext, questId int32) {
	m.registerQuestDirtyHandle()

	// 初始化脏事件 map
	if m.dirtyQuestEvent == nil {
		m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DUserQuestEvent)
	}

	// 从完成任务列表获取完成数据
	completedQuest := m.quests.GetUserQuestList().GetCompletedQuests()[questId]
	if completedQuest == nil {
		completedQuest = &public_protocol_pbdesc.DUserQuestCompletedData{
			QuestId:   questId,
			Timepoint: time.Now().Unix(),
		}
	}

	// 创建 Complete 类型事件
	questEvent := &public_protocol_pbdesc.DUserQuestEvent{
		EventId: _ctx.GetNow().UnixNano(),
		Event: &public_protocol_pbdesc.DUserQuestEvent_Complete{
			Complete: completedQuest,
		},
	}

	// 替换或添加到 map（自动覆盖旧事件）
	m.dirtyQuestEvent[questId] = questEvent
}

// addQuestEventReceived - 添加"奖励已领取"事件到脏数据
func (m *UserQuestManager) addQuestEventReceived(_ctx *cd.RpcContext, questId int32) {
	m.registerQuestDirtyHandle()
	// 初始化脏事件 map
	if m.dirtyQuestEvent == nil {
		m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DUserQuestEvent)
	}

	// 从已领取任务列表获取数据
	receivedQuest := m.quests.GetUserQuestList().GetReceivedQuests()[questId]
	if receivedQuest == nil {
		receivedQuest = &public_protocol_pbdesc.DUserQuestReceivedData{
			QuestId:   questId,
			Timepoint: time.Now().Unix(),
		}
	}

	// 创建 Received 类型事件
	questEvent := &public_protocol_pbdesc.DUserQuestEvent{
		EventId: _ctx.GetNow().UnixNano(),
		Event: &public_protocol_pbdesc.DUserQuestEvent_Received{
			Received: receivedQuest,
		},
	}

	// 替换或添加到 map（自动覆盖旧事件）
	m.dirtyQuestEvent[questId] = questEvent
}

// 注册任务脏数据推送 handle（确保只注册一次）
func (m *UserQuestManager) registerQuestDirtyHandle() {
	if m == nil {
		return
	}

	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		// 导出函数：将脏任务事件数据转换为 protobuf 并发送
		func(ctx *cd.RpcContext, dirty *data.UserDirtyData) bool {
			return m.dumpQuestDirtyData(ctx, dirty)
		},
		// 清理函数：导出后清理脏事件列表
		func(ctx *cd.RpcContext) {
			m.clearQuestDirtyData(ctx)
		},
	)
}

// 导出脏任务数据
func (m *UserQuestManager) dumpQuestDirtyData(ctx *cd.RpcContext, dirty *data.UserDirtyData) bool {
	if m == nil || len(m.dirtyQuestEvent) == 0 {
		return false
	}

	// 遍历所有脏任务事件，添加到脏数据消息中
	for questId, questEvent := range m.dirtyQuestEvent {
		// TODO: 当 proto 中 SCUserDirtyChgSync 添加任务事件字段后，将事件添加到相应的字段
		dirty.MutableNormalDirtyChangeMessage().MutableDirtyQuestEvents().Events = append(dirty.MutableNormalDirtyChangeMessage().MutableDirtyQuestEvents().Events, questEvent)
		ctx.LogDebug("quest event to be synced",
			"zone_id", m.GetOwner().GetZoneId(),
			"user_id", m.GetOwner().GetUserId(),
			"quest_id", questId,
			"event_id", questEvent.GetEventId(),
		)
	}
	return true
}

// 清理脏任务数据标记
func (m *UserQuestManager) clearQuestDirtyData(ctx *cd.RpcContext) {
	if m == nil {
		return
	}

	// 清空脏事件 map，为下一次变更做准备
	m.dirtyQuestEvent = make(map[int32]*public_protocol_pbdesc.DUserQuestEvent)
}
