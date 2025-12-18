package lobbysvr_logic_unlock_internal

import (
	"fmt"
	"reflect"

	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

var userManagerReflectType reflect.Type

// 确保实现接口
func init() {
	var _ logic_unlock.UserUnlockManager = (*UserUnlockManager)(nil)
	userManagerReflectType = lu.GetStaticReflectType[UserUnlockManager]()
	data.RegisterUserModuleManagerCreator[logic_unlock.UserUnlockManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return CreateUserUnlockManager(owner)
	})
}

type UserUnlockManager struct {
	data.UserModuleManagerBase

	modules   map[logic_unlock.UserUnlockListener]struct{}
	functions map[public_protocol_common.EnUnlockFunctionID]logic_unlock.UserUnlockListener

	lastCheckUnlockTime int64
}

func (m *UserUnlockManager) GetReflectType() reflect.Type {
	return userManagerReflectType
}

func CreateUserUnlockManager(owner *data.User) *UserUnlockManager {
	ret := &UserUnlockManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		modules:               make(map[logic_unlock.UserUnlockListener]struct{}),
		functions:             make(map[public_protocol_common.EnUnlockFunctionID]logic_unlock.UserUnlockListener),
	}
	return ret
}

func (m *UserUnlockManager) InitFromDB(_ cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	if dbUser.GetUnlockData() != nil {
		m.lastCheckUnlockTime = dbUser.GetUnlockData().GetLastCheckUnlockTimepoint()
	}
	return cd.CreateRpcResultOk()
}

func (m *UserUnlockManager) DumpToDB(_ cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	if dbUser == nil {
		return cd.CreateRpcResultOk()
	}
	dbUser.UnlockData = &private_protocol_pbdesc.UserUnlockData{
		LastCheckUnlockTimepoint: m.lastCheckUnlockTime,
	}
	return cd.CreateRpcResultOk()
}

func (m *UserUnlockManager) RegisterFunctionUnlockEvent(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, listener logic_unlock.UserUnlockListener) {
	if m == nil || listener == nil {
		return
	}
	m.functions[functionID] = listener
	m.modules[listener] = struct{}{}
}

// LoginInit 重建所有模块索引
func (m *UserUnlockManager) LoginInit(ctx cd.RpcContext) {
	m.RefreshLimitSecond(ctx)
}

// RefreshLimitSecond 按秒刷新（与 C++ 行为保持）
func (m *UserUnlockManager) RefreshLimitSecond(ctx cd.RpcContext) {
	for mod := range m.modules {
		if mod != nil {
			mod.Rebuild(ctx)
		}
	}
	// 时间解锁
	if ctx.GetNow().Unix() != m.lastCheckUnlockTime {
		m.OnUserUnlockDataChange(ctx, public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockTimepoint,
			m.lastCheckUnlockTime, ctx.GetNow().Unix())
		m.lastCheckUnlockTime = ctx.GetNow().Unix()
	}
}

// OnUserUnlockDataChange 根据条件变化触发解锁判定
func (m *UserUnlockManager) OnUserUnlockDataChange(ctx cd.RpcContext, condType public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID, oldValue, newValue int64) {
	if m == nil {
		return
	}
	// 取索引区间数据
	valueFunctionIndex := config.GetUnlockData(condType, oldValue, newValue)
	if len(valueFunctionIndex) == 0 {
		return
	}
	for _, value := range valueFunctionIndex {
		for _, functions := range value.Functions {
			listener, ok := m.functions[functions.FunctionID]
			if !ok || listener == nil {
				// 未注册模块，直接跳过
				continue
			}
			var unlockIDs []int32
			for _, unit := range functions.UnlockIDs {
				if unit == nil {
					continue
				}
				if m.CheckFunctionUnlock(ctx, unit.UnlockConditions) {
					unlockIDs = append(unlockIDs, unit.ID)
				}
			}
			if len(unlockIDs) > 0 {
				listener.NotifyFunctionUnlock(ctx, functions.FunctionID, unlockIDs)
			}
		}
	}
}

// CheckFunctionUnlock 判断条件是否满足
func (m *UserUnlockManager) CheckFunctionUnlock(ctx cd.RpcContext, conditions []*public_protocol_common.Readonly_DFunctionUnlockCondition) bool {
	if m == nil {
		return false
	}
	result := true
	for _, cond := range conditions {
		if cond == nil {
			continue
		}
		switch cond.GetConditionTypeOneofCase() {
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockTimepoint:
			result = m.CheckTimeUnlockConditnion(ctx, cond)
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_PlayerLevel:
			result = m.CheckPlayerLevelConditnion(ctx, cond)
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockByOtherSystem:
			result = false
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestFinish:
			result = m.CheckQuestConditnion(ctx, int32(cond.GetQuestFinish()), public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_COMPLETE)
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestReceived:
			result = m.CheckQuestConditnion(ctx, int32(cond.GetQuestReceived()), public_protocol_common.EnQuestStatus_EN_QUEST_STATUS_RECEIVE)
		case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_Activate:
			result = false
		default:
			ctx.LogError("unknown function unlock condition type: %d", cond.GetConditionTypeOneofCase())
			return false
		}
		if !result {
			return false
		}
	}
	return true
}

// DebugForceTrigger 便于 GM 或测试强制触发（可选）
func (m *UserUnlockManager) DebugForceTrigger(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, ids []int32) error {
	if m == nil {
		return fmt.Errorf("manager is nil")
	}
	listener, ok := m.functions[functionID]
	if !ok || listener == nil {
		return fmt.Errorf("listener not found for functionID %d", functionID)
	}
	listener.NotifyFunctionUnlock(ctx, functionID, ids)
	return nil
}

func (m *UserUnlockManager) CheckTimeUnlockConditnion(ctx cd.RpcContext, cond *public_protocol_common.Readonly_DFunctionUnlockCondition) bool {
	if cond.GetUnlockTimepoint().GetStartTimepoint().GetSeconds() > ctx.GetNow().Unix() {
		return false
	}
	return true
}

func (m *UserUnlockManager) CheckPlayerLevelConditnion(ctx cd.RpcContext, cond *public_protocol_common.Readonly_DFunctionUnlockCondition) bool {
	userBasicMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
	if userBasicMgr == nil {
		return false
	}
	return userBasicMgr.GetUserLevel() >= uint32(cond.GetPlayerLevel())
}

func (m *UserUnlockManager) CheckQuestConditnion(ctx cd.RpcContext, questID int32, status public_protocol_common.EnQuestStatus) bool {
	userQuestMgr := data.UserGetModuleManager[logic_quest.UserQuestManager](m.GetOwner())
	if userQuestMgr == nil {
		return false
	}

	return userQuestMgr.QueryQuestStatus(questID) == status
}
