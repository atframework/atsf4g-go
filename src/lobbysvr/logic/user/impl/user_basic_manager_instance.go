package lobbysvr_logic_user_impl

import (
	"fmt"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	"google.golang.org/protobuf/proto"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component-config"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func init() {
	var _ logic_user.UserBasicManager = (*UserBasicManager)(nil)
	data.RegisterUserModuleManagerCreator[logic_user.UserBasicManager](func(_ctx cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserBasicManager(owner)
	})

	registerCondition()
}

type UserBasicManager struct {
	data.UserModuleManagerBase

	dirtyExp         bool
	dirtyUserInfo    bool
	dirtyUserOptions bool

	dirtyLevelUpRewards map[int32]map[int64]*public_protocol_common.DItemInstance
	userOptions         *private_protocol_pbdesc.UserOptionsData
}

func CreateUserBasicManager(owner *data.User) *UserBasicManager {
	ret := &UserBasicManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserBasicManager) InitFromDB(_ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	m.userOptions = _dbUser.GetOptionsData()
	if m.userOptions == nil {
		m.userOptions = &private_protocol_pbdesc.UserOptionsData{}
	}
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) DumpToDB(_ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	dbUser.OptionsData = m.userOptions
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) CreateInit(ctx cd.RpcContext, versionType uint32) {
	m.GetOwner().MutableUserData().UserLevel = 1
	m.GetOwner().MutableUserData().UserExp = 0
}

func (m *UserBasicManager) RefreshLimitSecond(_ctx cd.RpcContext) {
}

func (m *UserBasicManager) DumpUserInfo() *public_protocol_pbdesc.DUserInfo {
	loginInfo := m.GetOwner().GetUserLogin()
	return &public_protocol_pbdesc.DUserInfo{
		UserLevel: m.GetUserLevel(),
		UserStat: &public_protocol_pbdesc.DUserStat{
			RegisterTime:  loginInfo.GetBusinessRegisterTime(),
			LastLoginTime: loginInfo.GetBusinessLoginTime(),
		},
	}
}

func (m *UserBasicManager) DumpUserOptions() *public_protocol_pbdesc.DUserOptions {
	return m.userOptions.GetCustomOptions()
}

func (m *UserBasicManager) insertDirtyHandle() {
	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
			ret := false
			dirtyData := dirty.MutableNormalDirtyChangeMessage()
			if m.dirtyExp {
				dirtyData.MutableDirtyItems().AppendItem(&public_protocol_common.DItemInstance{
					ItemBasic: &public_protocol_common.DItemBasic{
						TypeId: int32(public_protocol_common.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP),
						Count:  m.GetUserExp(),
					},
				})
				ret = true
			}

			if m.dirtyUserInfo {
				dirtyData.UserInfo = m.DumpUserInfo()
				ret = true
			}

			if m.dirtyUserOptions {
				dirtyData.UserOptions = m.GetUserClientOptions()
				ret = true
			}

			var dirtyLevelUpRewards *public_protocol_common.DItemAutoReward = nil
			for _, rewardItems := range m.dirtyLevelUpRewards {
				if rewardItems == nil {
					continue
				}
				for _, itemInst := range rewardItems {
					if dirtyLevelUpRewards == nil {
						dirtyLevelUpRewards = &public_protocol_common.DItemAutoReward{}
						dirtyLevelUpRewards.ReasonType = &public_protocol_common.DItemAutoReward_UserLevelUp{
							UserLevelUp: m.GetUserLevel(),
						}
						dirtyData.MutableAutoRewards().AppendRewards(dirtyLevelUpRewards)
					}

					if itemInst.GetItemBasic() != nil {
						dirtyLevelUpRewards.AppendRewardItems(itemInst.GetItemBasic())
						ret = true
					}
				}
			}

			return ret
		},
		func(_ctx cd.RpcContext) {
			m.dirtyExp = false
			m.dirtyUserInfo = false
			m.dirtyUserOptions = false
			clear(m.dirtyLevelUpRewards)
		},
	)
}

func (m *UserBasicManager) CheckAddUserExp(_ctx cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}
	// 奖励经验永远为true
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) CheckSubUserExp(ctx cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if m.GetUserExp() < v {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_NOT_ENOUGH)
	}

	return cd.CreateRpcResultOk()
}

// m.dirtyLevelUpRewards
func (m *UserBasicManager) mergeLevelUpRewardDirtyData(item *public_protocol_common.DItemInstance) {
	if m == nil || item == nil {
		return
	}

	if m.dirtyLevelUpRewards == nil {
		m.dirtyLevelUpRewards = make(map[int32]map[int64]*public_protocol_common.DItemInstance, 1)
	}

	typeId := item.GetItemBasic().GetTypeId()
	guid := item.GetItemBasic().GetGuid()
	count := item.GetItemBasic().GetCount()

	typeMap, ok := m.dirtyLevelUpRewards[typeId]
	if !ok {
		typeMap = make(map[int64]*public_protocol_common.DItemInstance, 1)
		m.dirtyLevelUpRewards[typeId] = typeMap
	}

	existingItem, ok := typeMap[guid]
	if !ok || existingItem == nil {
		typeMap[guid] = item.Clone()
		return
	} else {
		existingItem.MutableItemBasic().Count += count
	}
}

func (m *UserBasicManager) addLevelUpReward(ctx cd.RpcContext, cfg *public_protocol_config.Readonly_ExcelUserLevel) {
	if cfg == nil {
		return
	}

	if len(cfg.GetLevelUpReward()) == 0 {
		return
	}

	rewardInstances, _ := m.GetOwner().GenerateMultipleItemInstancesFromCfgOffset(ctx, cfg.GetLevelUpReward(), true)
	if len(rewardInstances) == 0 {
		return
	}

	batchAddGuard, result := m.GetOwner().CheckAddItem(ctx, rewardInstances)
	if result.IsOK() {
		m.GetOwner().AddItem(ctx, batchAddGuard, &data.ItemFlowReason{
			MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_USER),
			MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_USER_LEVEL_UP_REWARD),
			Parameter:   int64(cfg.GetLevel()),
		})

		for _, itemInst := range rewardInstances {
			m.mergeLevelUpRewardDirtyData(itemInst)
		}
		return
	}

	result.LogError(ctx, "batch check add item failed")

	for _, itemInst := range rewardInstances {
		singleAddGuard, result := m.GetOwner().CheckAddItem(ctx, []*public_protocol_common.DItemInstance{itemInst})
		if result.IsOK() {
			m.GetOwner().AddItem(ctx, singleAddGuard, &data.ItemFlowReason{
				MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_USER),
				MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_USER_LEVEL_UP_REWARD),
				Parameter:   int64(cfg.GetLevel()),
			})

			m.mergeLevelUpRewardDirtyData(itemInst)
			continue
		}

		result.LogError(ctx, "single check add item failed",
			"type_id", itemInst.GetItemBasic().GetTypeId(), "item_count", itemInst.GetItemBasic().GetCount())
	}
}

func (m *UserBasicManager) AddUserExp(ctx cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if v == 0 {
		return cd.CreateRpcResultOk()
	}

	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	userExpConfigIndex := configGroup.GetCustomIndex().GetUserExpLevelConfigIndex()
	if userExpConfigIndex == nil {
		return cd.CreateRpcResultError(fmt.Errorf("Can not find UserExpLevelConfigIndex"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	oldLevel := m.GetUserLevel()
	oldExp := m.GetUserExp()
	newExp := oldExp
	hasDirty := false
	if oldExp < userExpConfigIndex.MaxExp {
		if oldExp+v > userExpConfigIndex.MaxExp {
			newExp = userExpConfigIndex.MaxExp
		} else {
			newExp = oldExp + v
		}

		hasDirty = true
		m.dirtyExp = true
		m.GetOwner().MutableUserData().UserExp = newExp
	}

	checkLevelUp := m.GetUserLevel()
	for checkLevelUp < userExpConfigIndex.MaxLevel {
		checkLevelUp++

		levelConfig := configGroup.GetExcelUserLevelByLevel(int32(checkLevelUp))
		if levelConfig == nil {
			ctx.LogWarn("user level configure is missing", "level", checkLevelUp)
			continue
		}

		if newExp < levelConfig.GetExp() {
			break
		}

		// 升级
		m.GetOwner().MutableUserData().UserLevel = uint32(checkLevelUp)
		hasDirty = true
		m.dirtyUserInfo = true

		// 升级奖励
		m.addLevelUpReward(ctx, levelConfig)
	}

	// 触发任务 玩家升级
	if oldLevel != m.GetUserLevel() {
		questMgr := data.UserGetModuleManager[logic_quest.UserQuestManager](m.GetOwner())
		if questMgr != nil {
			questMgr.QuestTriggerEvent(ctx, private_protocol_pbdesc.QuestTriggerParams_EnParamID_PlayerLevel,
				&private_protocol_pbdesc.QuestTriggerParams{
					Param: &private_protocol_pbdesc.QuestTriggerParams_PlayerLevel{
						PlayerLevel: &private_protocol_pbdesc.QuestTargetParamsLevel{
							PreLevel: int64(oldLevel),
							CurLevel: int64(m.GetUserLevel()),
						},
					},
				})
		}

		unlockMgr := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
		if unlockMgr != nil {
			unlockMgr.OnUserUnlockDataChange(ctx, public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_PlayerLevel,
				int64(oldLevel), int64(m.GetUserLevel()))
		}
	}

	if hasDirty {
		m.insertDirtyHandle()

		ctx.LogDebug("user level upgrade", "new_level", m.GetUserLevel(), "new_exp", newExp,
			"old_level", oldLevel, "old_exp", oldExp)
	}

	// TODO: OSS经营分析统计
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) SubUserExp(_ctx cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if v == 0 {
		return cd.CreateRpcResultOk()
	}

	if m.GetUserExp() < v {
		m.GetOwner().MutableUserData().UserExp = 0
	} else {
		m.GetOwner().MutableUserData().UserExp -= v
	}

	m.insertDirtyHandle()

	// TODO: OSS经营分析统计
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) GetUserExp() int64 {
	return m.GetOwner().GetUserData().GetUserExp()
}

func (m *UserBasicManager) GetUserLevel() uint32 {
	return m.GetOwner().GetUserData().GetUserLevel()
}

func (m *UserBasicManager) GmResetUserExp(ctx cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	m.GetOwner().MutableUserData().UserExp = 0
	m.GetOwner().MutableUserData().UserLevel = 1

	return m.AddUserExp(ctx, v)
}

func (m *UserBasicManager) ForeachItem(fn func(item *public_protocol_common.DItemInstance) bool) bool {
	if fn == nil {
		return true
	}

	inst := &public_protocol_common.DItemInstance{
		ItemBasic: &public_protocol_common.DItemBasic{
			TypeId: int32(public_protocol_common.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP),
			Count:  m.GetUserExp(),
		},
	}

	return fn(inst)
}

func (m *UserBasicManager) GetAttributesCacheVersion() int64 {
	return 0
}

func (m *UserBasicManager) GetUserClientOptions() *public_protocol_pbdesc.DUserOptions {
	if m == nil {
		return nil
	}

	return m.userOptions.GetCustomOptions()
}

func (m *UserBasicManager) UpdateUserClientOptions(ctx cd.RpcContext, opts *public_protocol_pbdesc.DUserOptions) data.Result {
	if opts == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	sizeLimit := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUserOptionsMaxSize()
	if proto.Size(opts) > int(sizeLimit) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_CLIENT_OPTIONS_SIZE_LIMIT)
	}

	m.userOptions.CustomOptions = opts.Clone()

	m.dirtyUserOptions = true
	m.insertDirtyHandle()
	return cd.CreateRpcResultOk()
}

func registerCondition() {
	logic_condition.AddRuleChecker(public_protocol_common.GetTypeIDDConditionRule_LoginChannel(), checkRuleUserLoginChannel, nil)
	logic_condition.AddRuleChecker(public_protocol_common.GetTypeIDDConditionRule_SystemPlatform(), checkRuleUserSystemPlatform, nil)
	logic_condition.AddRuleChecker(public_protocol_common.GetTypeIDDConditionRule_UserLevel(), checkRuleUserLevelStatic, checkRuleUserLevelDynamic)
}

func checkRuleUserLoginChannel(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	loginChannel := uint64(m.GetOwner().GetAccountInfo().GetChannelId())

	if len(rule.GetLoginChannel().GetValues()) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, v := range rule.GetLoginChannel().GetValues() {
		if loginChannel == v {
			return cd.CreateRpcResultOk()
		}
	}

	// 错误码: 登入平台不满足要求
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_INVALID_PLATFORM)
}

func checkRuleUserSystemPlatform(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	loginChannel := uint64(m.GetOwner().GetClientInfo().GetSystemId())

	if len(rule.GetSystemPlatform().GetValues()) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, v := range rule.GetSystemPlatform().GetValues() {
		if loginChannel == v {
			return cd.CreateRpcResultOk()
		}
	}

	// 错误码: 登入平台不满足要求
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_INVALID_CHANNEL)
}

func checkRuleUserLevelDynamic(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule.GetUserLevel().GetLeft() <= 1 {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserBasicManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	userLevel := mgr.GetUserLevel()
	if int64(userLevel) < rule.GetUserLevel().GetLeft() {
		// 错误码: 最小等级不满足要求
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_MIN_LEVEL_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleUserLevelStatic(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule.GetUserLevel().GetRight() <= 0 {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserBasicManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	userLevel := mgr.GetUserLevel()
	if int64(userLevel) > rule.GetUserLevel().GetRight() {
		// 错误码: 最大等级不满足要求
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_MAX_LEVEL_LIMIT)
	}

	return cd.CreateRpcResultOk()
}
