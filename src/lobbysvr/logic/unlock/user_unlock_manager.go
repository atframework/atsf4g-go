package lobbysvr_logic_unlock

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// UserUnlockListener 接收功能解锁事件的模块需实现
type UserUnlockListener interface {
	Rebuild(ctx cd.RpcContext)
	// 可能会重复触发同一个unlockID的解锁，要去重
	NotifyFunctionUnlock(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, unlockIDs []int32)
}

// UserUnlockManager 功能解锁管理接口
// 嵌入 data.UserModuleManagerImpl 以兼容用户模块生命周期管理
type UserUnlockManager interface {
	data.UserModuleManagerImpl
	RegisterFunctionUnlockEvent(ctx cd.RpcContext, functionID public_protocol_common.EnUnlockFunctionID, listener UserUnlockListener)
	OnUserUnlockDataChange(ctx cd.RpcContext, condType public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID, oldValue, newValue int64)
	CheckFunctionUnlock(ctx cd.RpcContext, conditions []*public_protocol_common.Readonly_DFunctionUnlockCondition) bool
}
