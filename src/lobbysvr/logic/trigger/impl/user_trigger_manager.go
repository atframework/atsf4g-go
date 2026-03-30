package lobbysvr_logic_trigger_impl

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_trigger "github.com/atframework/atsf4g-go/service-lobbysvr/logic/trigger"
)

type UserTriggerManager struct {
	data.UserModuleManagerBase
}

func init() {
	var _ logic_trigger.UserTriggerManager = (*UserTriggerManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_trigger.UserTriggerManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return createUserTriggerManager(owner)
	})
}

func createUserTriggerManager(owner *data.User) *UserTriggerManager {
	ret := &UserTriggerManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserTriggerManager) TriggerRule(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DTriggerRule) {
	for _, rule := range rules {
		triggerFunc := logic_trigger.GetTriggerRuleFunc(rule.GetRuleTypeTypeID())
		if triggerFunc == nil {
			continue
		}
		triggerFunc(ctx, m.GetOwner(), rule)
	}
}
