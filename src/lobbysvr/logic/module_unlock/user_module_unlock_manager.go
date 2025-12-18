package lobbysvr_logic_module_unlock

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
)

type UserModuleUnlockManager interface {
	data.UserModuleManagerImpl
	logic_unlock.UserUnlockListener

	IsModuleUnlocked(moduleId int32) bool
	UnlockModuleByQuest(moduleId int32) int32
	DumpModuleUnlockData(moduleUnlockData *public_protocol_pbdesc.DUserModuleUnlockData)

	GMUnlockAllModules(ctx cd.RpcContext)
	GMUnlockModule(ctx cd.RpcContext, moduleId int32)
	GMQueryModuleStatus(moduleId int32) bool

	// 注册指定moduleId的解锁事件回调
	RegisterModuleUnlockCallback(moduleId int32, callback ModuleUnlockCallback)
}

// ModuleUnlockCallback 模块解锁事件回调
type ModuleUnlockCallback func(ctx cd.RpcContext)
