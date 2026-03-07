package lobbysvr_logic_module_unlock_internal

import (
	"time"

	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_module_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/module_unlock"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
)

func init() {
	var _ logic_module_unlock.UserModuleUnlockManager = (*UserModuleUnlockManager)(nil)
	data.RegisterUserModuleManagerCreator[logic_module_unlock.UserModuleUnlockManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return CreateUserModuleUnlockManager(owner)
	})
}

type UserModuleUnlockManager struct {
	data.UserModuleManagerBase

	idToModule map[int32]*public_protocol_pbdesc.DModuleUnlocked

	unlockResourceVersion uint64

	dirtyModuleUnlockEvent []*public_protocol_pbdesc.DModuleUnlocked

	// 模块解锁事件回调 map[moduleId]callback
	unlockCallbacks map[int32]logic_module_unlock.ModuleUnlockCallback
}

func CreateUserModuleUnlockManager(owner *data.User) *UserModuleUnlockManager {
	return &UserModuleUnlockManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		idToModule:            make(map[int32]*public_protocol_pbdesc.DModuleUnlocked),
		unlockCallbacks:       make(map[int32]logic_module_unlock.ModuleUnlockCallback),
	}
}

// db load & save

func (m *UserModuleUnlockManager) InitFromDB(ctx cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	if dbUser.GetModuleUnlockData() != nil {
		for _, mod := range dbUser.GetModuleUnlockData().GetUnlockModules() {
			if mod == nil {
				continue
			}
			m.idToModule[mod.GetModuleId()] = mod
		}
		m.unlockResourceVersion = dbUser.GetModuleUnlockData().GetUnlockResourceVersion()
	}
	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if functionUnlockManager != nil {
		functionUnlockManager.RegisterFunctionUnlockEvent(ctx, public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_MODULE, m)
	}
	return cd.CreateRpcResultOk()
}

func (m *UserModuleUnlockManager) DumpToDB(_ cd.RpcContext,
	dbUser *private_protocol_pbdesc.DatabaseTableUser,
) cd.RpcResult {
	if dbUser == nil {
		return cd.CreateRpcResultOk()
	}
	dbUser.ModuleUnlockData = &private_protocol_pbdesc.UserModuleUnlockData{
		UnlockModules:         make([]*public_protocol_pbdesc.DModuleUnlocked, 0, len(m.idToModule)),
		UnlockResourceVersion: m.unlockResourceVersion,
	}
	for _, mod := range m.idToModule {
		dbUser.ModuleUnlockData.UnlockModules = append(dbUser.ModuleUnlockData.UnlockModules, mod)
	}
	return cd.CreateRpcResultOk()
}

func (m *UserModuleUnlockManager) DumpModuleUnlockData(moduleUnlockData *public_protocol_pbdesc.DUserModuleUnlockData) {
	if moduleUnlockData == nil {
		return
	}
	moduleUnlockData.UnlockModules = make([]*public_protocol_pbdesc.DModuleUnlocked, 0, len(m.idToModule))
	for _, mod := range m.idToModule {
		moduleUnlockData.UnlockModules = append(moduleUnlockData.UnlockModules, mod)
	}
}

func (m *UserModuleUnlockManager) LoginInit(ctx cd.RpcContext) {
	// functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	// if functionUnlockManager != nil {
	// 	functionUnlockManager.RegisterFunctionUnlockEvent(ctx, public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_MODULE, m)
	// }
	// m.Rebuild(ctx)
}

func (m *UserModuleUnlockManager) Rebuild(ctx cd.RpcContext) {

	if m.unlockResourceVersion == config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeHashCodeVersion() {
		return
	}

	m.unlockResourceVersion = config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeHashCodeVersion()

	rows := config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeAllOfModuleId()
	if rows == nil {
		return
	}

	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())

	for _, row := range *rows {
		if row == nil {
			continue
		}

		if m.IsModuleUnlocked(row.GetModuleId()) {
			continue
		}

		unlocked := false
		if functionUnlockManager != nil {
			unlocked = functionUnlockManager.CheckFunctionUnlock(ctx, row.GetUnlockCondition())
		}
		if unlocked {
			m.moduleUnlockInner(ctx, row.GetModuleId())
		} else {
			moduleId := row.GetModuleId()
			mod := m.idToModule[moduleId]
			if mod == nil {
				mod = &public_protocol_pbdesc.DModuleUnlocked{
					ModuleId:        moduleId,
					UnlockTimepoint: 0,
				}
				m.idToModule[moduleId] = mod
			}
		}
	}
}

func (m *UserModuleUnlockManager) ChecekModuleUnlockCondition(ctx cd.RpcContext, cfg *public_protocol_config.Readonly_ExcelModuleUnlockType) bool {
	functionUnlockManager := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if functionUnlockManager == nil {
		ctx.LogError("functionUnlockManager is nil")
		return false
	}
	return functionUnlockManager.CheckFunctionUnlock(ctx, cfg.GetUnlockCondition())
}

func (m *UserModuleUnlockManager) NotifyFunctionUnlock(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, unlockIDs []int32) {
	if len(unlockIDs) == 0 {
		return
	}
	if functionID != public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_MODULE {
		return
	}

	group := config.GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return
	}
	for _, moduleId := range unlockIDs {
		if m.IsModuleUnlocked(moduleId) {
			// 已经解锁，跳过
			continue
		}
		row := group.GetExcelModuleUnlockTypeByModuleId(moduleId)
		if row == nil || m.ChecekModuleUnlockCondition(ctx, row) == false {
			continue
		}

		m.moduleUnlockInner(ctx, moduleId)
	}
}

func (m *UserModuleUnlockManager) IsModuleUnlocked(moduleId int32) bool {
	mod := m.idToModule[moduleId]
	if mod == nil {
		return false
	}
	return mod.GetUnlockTimepoint() > 0
}

func (m *UserModuleUnlockManager) UnlockModuleByQuest(moduleId int32) int32 {
	mod := m.idToModule[moduleId]
	if mod == nil {
		mod = &public_protocol_pbdesc.DModuleUnlocked{}
		m.idToModule[moduleId] = mod
	}
	if mod.UnlockTimepoint == 0 {
		mod.ModuleId = moduleId
		mod.UnlockTimepoint = time.Now().Unix()
	}
	return 0
}

func (m *UserModuleUnlockManager) GMUnlockAllModules(ctx cd.RpcContext) {
	rows := config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeAllOfModuleId()
	if rows == nil {
		return
	}
	m.idToModule = make(map[int32]*public_protocol_pbdesc.DModuleUnlocked)

	for _, row := range *rows {
		if row == nil {
			continue
		}
		m.moduleUnlockInner(ctx, row.GetModuleId())
	}
}
func (m *UserModuleUnlockManager) GMUnlockModule(ctx cd.RpcContext, moduleId int32) {

	row := config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeByModuleId(moduleId)
	if row == nil {
		return
	}
	m.moduleUnlockInner(ctx, moduleId)

}
func (m *UserModuleUnlockManager) GMQueryModuleStatus(moduleId int32) bool {
	mod, ok := m.idToModule[moduleId]
	if !ok || mod == nil {
		return false
	}
	return mod.GetUnlockTimepoint() > 0
}

// RegisterModuleUnlockCallback 注册指定moduleId的解锁事件回调
func (m *UserModuleUnlockManager) RegisterModuleUnlockCallback(moduleId int32, callback logic_module_unlock.ModuleUnlockCallback) {
	if moduleId <= 0 || callback == nil {
		return
	}
	m.unlockCallbacks[moduleId] = callback
}

func (m *UserModuleUnlockManager) moduleUnlockInner(ctx cd.RpcContext, moduleId int32) {
	mod := m.idToModule[moduleId]
	if mod == nil {
		mod = &public_protocol_pbdesc.DModuleUnlocked{}
		m.idToModule[moduleId] = mod
	}
	mod.ModuleId = moduleId
	mod.UnlockTimepoint = ctx.GetNow().Unix()
	m.addModuleUnlockDirty(ctx, mod)

	// 触发解锁
	unlockMgr := data.UserGetModuleManager[logic_unlock.UserUnlockManager](m.GetOwner())
	if unlockMgr != nil {
		unlockMgr.OnUserUnlockDataChange(ctx, public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_ModuleUnlocked, int64(moduleId))
	}

	// 触发该moduleId的解锁回调
	if callback, exists := m.unlockCallbacks[moduleId]; exists && callback != nil {
		callback(ctx)
	}
}

func (m *UserModuleUnlockManager) addModuleUnlockDirty(_ctx cd.RpcContext, dirtyModule *public_protocol_pbdesc.DModuleUnlocked) {
	m.registerModuleUnlockDirtyHandle()
	m.dirtyModuleUnlockEvent = append(m.dirtyModuleUnlockEvent, dirtyModule)
}

func (m *UserModuleUnlockManager) dumpModuleUnlockDirtyData(ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
	if m == nil || len(m.dirtyModuleUnlockEvent) == 0 {
		return false
	}

	dirtyMsg := dirty.MutableNormalDirtyChangeMessage()
	for _, drityModule := range m.dirtyModuleUnlockEvent {
		// 将脏模块数据添加到消息中
		dirtyMsg.MutableDirtyModuleUnlock().DirtyModules = append(dirtyMsg.MutableDirtyModuleUnlock().DirtyModules, drityModule.Clone())
		ctx.LogDebug("module unlock event to be synced",
			"module_id", drityModule.GetModuleId(),
		)
	}
	return true
}

// 清理脏任务数据标记.
func (m *UserModuleUnlockManager) clearModuleUnlockDirtyData(_ cd.RpcContext) {
	if m == nil {
		return
	}
	m.dirtyModuleUnlockEvent = []*public_protocol_pbdesc.DModuleUnlocked{}
}

// 注册任务脏数据推送 handle（确保只注册一次）.
func (m *UserModuleUnlockManager) registerModuleUnlockDirtyHandle() {
	if m == nil {
		return
	}

	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		// 导出函数：将脏任务事件数据转换为 protobuf 并发送
		func(ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
			return m.dumpModuleUnlockDirtyData(ctx, dirty)
		},
		// 清理函数：导出后清理脏事件列表
		func(ctx cd.RpcContext) {
			m.clearModuleUnlockDirtyData(ctx)
		},
	)
}

func (m *UserModuleUnlockManager) ReceviedReward(ctx cd.RpcContext, moduleId int32) (result cd.RpcResult, rewards []*public_protocol_common.DItemBasic) {
	mod := m.idToModule[moduleId]
	if mod == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MODULE_NOT_FOUND), nil
	}

	if mod.GetUnlockTimepoint() <= 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MODULE_NOT_UNLOCKED), nil
	}

	if mod.GetIsReceived() == true {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MODULE_REWARD_ALREADY_RECEIVED), nil
	}
	rewardCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelModuleUnlockTypeByModuleId(moduleId)
	if rewardCfg == nil || len(rewardCfg.GetRewards()) == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MODULE_REWARD_NOT_FOUND), nil
	}

	rewardItemInsts, result := m.GetOwner().GenerateMultipleItemInstancesFromCfgOffset(ctx, rewardCfg.GetRewards(), false)
	if result.IsError() {
		ctx.LogError("generate module reward items failed",
			"module_id", moduleId,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return result, nil
	}

	addGuards, result := m.GetOwner().CheckAddItem(ctx, rewardItemInsts)
	if result.IsError() {
		ctx.LogError("check add module reward failed",
			"module_id", moduleId,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return result, nil
	}

	itemFlowReason := &data.ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_MODULE_UNLOCK),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_MODULE_REWARD),
		Parameter:   int64(moduleId),
	}

	mod.IsReceived = true
	m.idToModule[moduleId] = mod
	m.addModuleUnlockDirty(ctx, mod)

	result = m.GetOwner().AddItem(ctx, addGuards, itemFlowReason)
	if !result.IsOK() {
		ctx.LogError("add module reward items failed",
			"module_id", moduleId,
			"error", result.GetStandardError(),
			"response_code", result.GetResponseCode(),
		)
		return result, nil
	}

	rewardBasic := make([]*public_protocol_common.DItemBasic, 0, len(addGuards))
	for _, addGuard := range addGuards {
		rewardBasic = append(rewardBasic, addGuard.Item.GetItemBasic())
	}

	return cd.CreateRpcResultOk(), rewardBasic
}
