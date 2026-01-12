package lobbysvr_logic_condition

import (
	"fmt"
	"reflect"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	logic_condition_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition/data"
)

type CheckConditionFunc = func(m UserConditionManager, ctx cd.RpcContext, rule *public_protocol_common.Readonly_DConditionRule, runtime *RuleCheckerRuntime) cd.RpcResult

// 所有Static接口指用户不重新登入，不可能变化的条件检查。比如平台限制，等级上限等。
// 所有Dynamic接口指用户在不重新登入情况下，可能变化的条件检查。比如等级下限等，解锁关卡等。
type UserConditionManager interface {
	data.UserModuleManagerImpl

	AllocateConditionCounterStorageId() int64
	AllocateCouterStorage(ctx cd.RpcContext, version int64) *public_protocol_common.DConditionCounterStorage

	DumpConditionCounterData(to *public_protocol_pbdesc.DConditionCounterData)

	CheckStaticRuleId(ctx cd.RpcContext, ruleId int32, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckDynamicRuleId(ctx cd.RpcContext, ruleId int32, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckRuleId(ctx cd.RpcContext, ruleId int32, runtime *RuleCheckerRuntime) cd.RpcResult

	CheckStaticRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckDynamicRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *RuleCheckerRuntime) cd.RpcResult

	CheckDateTimeStaticLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult
	CheckDateTimeDynamicLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult
	CheckDateTimeLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult

	CheckBasicStaticLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckBasicDynamicLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *RuleCheckerRuntime) cd.RpcResult
	CheckBasicLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *RuleCheckerRuntime) cd.RpcResult

	RefreshCounter(ctx cd.RpcContext, now time.Time,
		limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage,
	)

	HasCounterLimit(limit *public_protocol_common.DConditionCounterLimit) bool
	HasCounterLimitCfg(limit *public_protocol_common.Readonly_DConditionCounterLimit) bool

	CalculateCounterLimitCfgMaxLeftTimes(limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage) int64

	CheckCounterStaticLimit(ctx cd.RpcContext, now time.Time, offset int64,
		limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult
	CheckCounterDynamicLimit(ctx cd.RpcContext, now time.Time, offset int64,
		limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult
	CheckCounterLimit(ctx cd.RpcContext, now time.Time, offset int64,
		limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult

	AddCounter(ctx cd.RpcContext, now time.Time, offset int64,
		limit *public_protocol_common.Readonly_DConditionCounterLimit,
		storage *public_protocol_common.DConditionCounterStorage) cd.RpcResult
}

type UserConditionCounterDelegate interface {
	GetCounterSizeCapacity() int32
	ForeachConditionCounter(f func(storage *public_protocol_common.DConditionCounterStorage) bool)
}

var userConditionCounterDelegates = map[reflect.Type]func(u *data.User) UserConditionCounterDelegate{}

func RegisterConditionCounterDelegate[FinalType interface{}](fn func(u *data.User) UserConditionCounterDelegate) {
	t := reflect.TypeOf((*FinalType)(nil)).Elem()
	userConditionCounterDelegates[t] = fn
}

func ForeachConditionCounterDelegate(u *data.User, fn func(d UserConditionCounterDelegate) bool) {
	if u == nil {
		return
	}

	for _, getDelegate := range userConditionCounterDelegates {
		delegate := getDelegate(u)
		if delegate != nil {
			if !fn(delegate) {
				return
			}
		}
	}
}

func HasLimitData(limit *public_protocol_common.Readonly_DConditionBasicLimit) bool {
	if limit == nil {
		return false
	}

	return len(limit.GetRule()) > 0 || len(limit.GetValidTime()) > 0
}

type (
	RuleCheckerParameterPair = logic_condition_data.RuleCheckerParameterPair
	RuleCheckerRuntime       = logic_condition_data.RuleCheckerRuntime
)

func CreateRuleCheckerRuntime(params ...RuleCheckerParameterPair) *RuleCheckerRuntime {
	return logic_condition_data.CreateRuleCheckerRuntime(params...)
}

func CreateEmptyRuleCheckerRuntime() *RuleCheckerRuntime {
	return nil
}

type conditionRuleCheckHandle struct {
	StaticChecker  CheckConditionFunc
	DynamicChecker CheckConditionFunc
}

func buildRuleCheckers() map[reflect.Type]*conditionRuleCheckHandle {
	ret := map[reflect.Type]*conditionRuleCheckHandle{}
	return ret
}

var conditionRuleCheckers = buildRuleCheckers()

// 注册规则检查器，此函数请在init函数中调用
// t 必须是 DConditionRule.RuleType 中的具体类型
func AddRuleChecker(t reflect.Type, staticChecker CheckConditionFunc, dynamicChecker CheckConditionFunc) error {
	if staticChecker == nil && dynamicChecker == nil {
		return fmt.Errorf("at least one of staticChecker or dynamicChecker must be non-nil")
	}

	if _, exists := conditionRuleCheckers[t]; exists {
		return fmt.Errorf("rule checker for type %v already exists", t)
	}

	conditionRuleCheckers[t] = &conditionRuleCheckHandle{
		StaticChecker:  staticChecker,
		DynamicChecker: dynamicChecker,
	}

	return nil
}

func GetStaticRuleChecker(t reflect.Type) CheckConditionFunc {
	checker, exists := conditionRuleCheckers[t]
	if !exists {
		return nil
	}

	return checker.StaticChecker
}

func GetDynamicRuleChecker(t reflect.Type) CheckConditionFunc {
	checker, exists := conditionRuleCheckers[t]
	if !exists {
		return nil
	}

	return checker.DynamicChecker
}
