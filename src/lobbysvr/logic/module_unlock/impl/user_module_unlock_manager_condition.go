package lobbysvr_logic_module_unlock_internal

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_module_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/module_unlock"
)

func registerCondition() {
	logic_condition.AddRuleChecker(public_protocol_common.GetTypeIDDConditionRule_ModuleUnlock(), nil, ChecekModuleUnlockCondition)
}

func ChecekModuleUnlockCondition(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule.GetModuleUnlock() <= 0 {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserBasicManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if !mgr.IsModuleUnlocked(rule.GetModuleUnlock()) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MODULE_NOT_UNLOCKED)
	}

	return cd.CreateRpcResultOk()
}
