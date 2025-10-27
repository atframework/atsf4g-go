package lobbysvr_logic_item

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// 所有Static接口指用户不重新登入，不可能变化的条件检查。比如平台限制，等级上限等。
// 所有Dynamic接口指用户在不重新登入情况下，可能变化的条件检查。比如等级下限等，解锁关卡等。
type UserConditionManager interface {
	data.UserModuleManagerImpl

	CheckStaticRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult
	CheckDynamicRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult
	CheckRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult

	CheckStaticRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult
	CheckDynamicRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult
	CheckRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult

	CheckDateTimeStaticLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult
	CheckDateTimeDynamicLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult
	CheckDateTimeLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult

	CheckBasicStaticLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult
	CheckBasicDynamicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult
	CheckBasicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult

	CheckCounterStaticLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit, storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult
	CheckCounterDynamicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit, storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult
	CheckCounterLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit, storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult

	AddCounter(ctx *cd.RpcContext, storage *public_protocol_common.DConditionCounterStorage, offset int64) cd.RpcResult
}

func HasLimitData(limit *public_protocol_common.DConditionBasicLimit) bool {
	if limit == nil {
		return false
	}

	return len(limit.GetRule()) > 0 || len(limit.GetValidTime()) > 0
}
