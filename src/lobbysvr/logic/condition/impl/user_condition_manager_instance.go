package lobbysvr_logic_condition_impl

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component-config"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
)

func init() {
	var _ logic_condition.UserConditionManager = (*UserConditionManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_condition.UserConditionManager](func(_ctx cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserConditionManager(owner)
	})
}

type UserConditionManager struct {
	data.UserModuleManagerBase
}

func CreateUserConditionManager(owner *data.User) *UserConditionManager {
	ret := &UserConditionManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserConditionManager) CheckStaticRuleId(ctx cd.RpcContext, ruleId int32, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	ruleCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelConditionPoolByConditionId(ruleId)
	if ruleCfg == nil {
		// TODO: 错误码: 条件规则不存在
		return cd.CreateRpcResultError(fmt.Errorf("rule config not found for rule id %d", ruleId), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if len(ruleCfg.GetBasicLimit().GetRule()) > 0 {
		return m.CheckStaticRules(ctx, ruleCfg.GetBasicLimit().GetRule(), runtime)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRuleId(ctx cd.RpcContext, ruleId int32, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	ruleCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelConditionPoolByConditionId(ruleId)
	if ruleCfg == nil {
		// TODO: 错误码: 条件规则不存在
		return cd.CreateRpcResultError(fmt.Errorf("rule config not found for rule id %d", ruleId), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if len(ruleCfg.GetBasicLimit().GetRule()) > 0 {
		return m.CheckDynamicRules(ctx, ruleCfg.GetBasicLimit().GetRule(), runtime)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckRuleId(ctx cd.RpcContext, ruleId int32, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	result := m.CheckStaticRuleId(ctx, ruleId, runtime)
	if !result.IsOK() {
		return result
	}

	return m.CheckDynamicRuleId(ctx, ruleId, runtime)
}

func (m *UserConditionManager) CheckStaticRules(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		checkHandle := logic_condition.GetStaticRuleChecker(rule.GetRuleTypeReflectType())
		if checkHandle == nil {
			continue
		}

		mi := (logic_condition.UserConditionManager)(m)
		result := checkHandle(mi, ctx, rule, runtime.MakeCurrentRuntime(rule.GetRuleTypeReflectType()))
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRules(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		checkHandle := logic_condition.GetDynamicRuleChecker(rule.GetRuleTypeReflectType())
		if checkHandle == nil {
			continue
		}

		mi := (logic_condition.UserConditionManager)(m)
		result := checkHandle(mi, ctx, rule, runtime.MakeCurrentRuntime(rule.GetRuleTypeReflectType()))
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckRules(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	result := m.CheckStaticRules(ctx, rules, runtime)
	if !result.IsOK() {
		return result
	}

	return m.CheckDynamicRules(ctx, rules, runtime)
}

func (m *UserConditionManager) CheckDateTimeStaticLimit(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	// 容忍一定秒数，避免时间同步误差导致的问题
	toilateSeconds := int64(5)

	// 忽略低于1秒的精度
	nowSec := ctx.GetNow().Unix()
	for _, rule := range rules {
		endTime := rule.GetEndTime().GetSeconds()

		// 永远有效的时间段，直接通过，开始时间会在动态条件中检查
		if endTime <= 0 {
			return cd.CreateRpcResultOk()
		}

		// 任意一个时间段未结束即通过，开始时间会在动态条件中检查
		if endTime > 0 && endTime+toilateSeconds < nowSec {
			continue
		}

		return cd.CreateRpcResultOk()
	}

	// TODO: 错误码: 所有可用时段都已结束
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
}

func (m *UserConditionManager) CheckDateTimeDynamicLimit(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	// 容忍一定秒数，避免时间同步误差导致的问题
	toilateSeconds := int64(5)

	// 忽略低于1秒的精度
	nowSec := ctx.GetNow().Unix()
	for _, rule := range rules {
		startTime := rule.GetBeginTime().GetSeconds()
		endTime := rule.GetEndTime().GetSeconds()
		if startTime > nowSec+toilateSeconds {
			continue
		}

		if endTime > 0 && endTime+toilateSeconds < nowSec {
			continue
		}

		return cd.CreateRpcResultOk()
	}

	// TODO: 错误码: 未开始
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
}

func (m *UserConditionManager) CheckDateTimeLimit(ctx cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
	result := m.CheckDateTimeStaticLimit(ctx, rules)
	if !result.IsOK() {
		return result
	}

	return m.CheckDateTimeDynamicLimit(ctx, rules)
}

func (m *UserConditionManager) CheckBasicStaticLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if limit == nil {
		return cd.CreateRpcResultOk()
	}

	if len(limit.GetValidTime()) > 0 {
		result := m.CheckDateTimeStaticLimit(ctx, limit.GetValidTime())
		if !result.IsOK() {
			return result
		}
	}

	if len(limit.GetRule()) > 0 {
		result := m.CheckStaticRules(ctx, limit.GetRule(), runtime)
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckBasicDynamicLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if limit == nil {
		return cd.CreateRpcResultOk()
	}

	if len(limit.GetValidTime()) > 0 {
		result := m.CheckDateTimeDynamicLimit(ctx, limit.GetValidTime())
		if !result.IsOK() {
			return result
		}
	}

	if len(limit.GetRule()) > 0 {
		result := m.CheckDynamicRules(ctx, limit.GetRule(), runtime)
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckBasicLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	result := m.CheckBasicStaticLimit(ctx, limit, runtime)
	if !result.IsOK() {
		return result
	}

	return m.CheckBasicDynamicLimit(ctx, limit, runtime)
}

func (m *UserConditionManager) CheckCounterStaticLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	// TODO: 实现静态计数器检查
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterDynamicLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	// TODO: 实现动态计数器检查
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterLimit(ctx cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	result := m.CheckCounterStaticLimit(ctx, limit, storage)
	if !result.IsOK() {
		return result
	}

	return m.CheckCounterDynamicLimit(ctx, limit, storage)
}

func (m *UserConditionManager) AddCounter(ctx cd.RpcContext, storage *public_protocol_common.DConditionCounterStorage, offset int64) cd.RpcResult {
	// TODO: 实现计数器增加逻辑
	return cd.CreateRpcResultOk()
}
