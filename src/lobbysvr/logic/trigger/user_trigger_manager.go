package lobbysvr_logic_buff

import (
	"fmt"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserTriggerManager interface {
	data.UserModuleManagerImpl
	TriggerRule(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DTriggerRule)
}

func buildTriggerRules() map[lu.TypeID]TriggerFunc {
	ret := map[lu.TypeID]TriggerFunc{}
	return ret
}

var triggerRules = buildTriggerRules()

type TriggerFunc = func(ctx cd.RpcContext, user *data.User, rule *public_protocol_common.Readonly_DTriggerRule)

func RegisterTriggerRule(t lu.TypeID, trigger TriggerFunc) error {
	if trigger == nil {
		return fmt.Errorf("trigger function must be non-nil")
	}

	if _, exists := triggerRules[t]; exists {
		return fmt.Errorf("rule checker for type %v already exists", t)
	}

	triggerRules[t] = trigger
	return nil
}

func GetTriggerRuleFunc(t lu.TypeID) TriggerFunc {
	trigger, exists := triggerRules[t]
	if !exists {
		return nil
	}

	return trigger
}
