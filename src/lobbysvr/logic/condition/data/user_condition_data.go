package lobbysvr_logic_condition_data

import (
	"reflect"
)

type RuleCheckerParameterPair struct {
	key   reflect.Type
	value interface{}
}

type RuleCheckerRuntime struct {
	ruleParameter        map[reflect.Type]interface{}
	currentRuleParameter interface{}
}

func (r *RuleCheckerRuntime) GetRuleParameter(t reflect.Type) interface{} {
	if r == nil {
		return nil
	}

	return r.ruleParameter[t]
}

func (r *RuleCheckerRuntime) MakeCurrentRuntime(t reflect.Type) *RuleCheckerRuntime {
	if r == nil {
		return nil
	}

	return &RuleCheckerRuntime{
		ruleParameter:        r.ruleParameter,
		currentRuleParameter: r.GetRuleParameter(t),
	}
}

func CreateRuntimeParameterPair[T interface{}](v T) RuleCheckerParameterPair {
	return RuleCheckerParameterPair{
		key:   reflect.TypeOf((*T)(nil)).Elem(),
		value: v,
	}
}

func CreateRuleCheckerRuntime(params ...RuleCheckerParameterPair) *RuleCheckerRuntime {
	if len(params) == 0 {
		return nil
	}

	m := make(map[reflect.Type]interface{}, len(params))
	for _, p := range params {
		m[p.key] = p.value
	}

	ret := &RuleCheckerRuntime{
		ruleParameter:        m,
		currentRuleParameter: nil,
	}

	return ret
}

func GetRuleRuntimeParameter[T interface{}](r *RuleCheckerRuntime) T {
	if r == nil {
		var empty T
		return empty
	}

	v := r.GetRuleParameter(reflect.TypeOf((*T)(nil)).Elem())
	if v == nil {
		var empty T
		return empty
	}

	ret, ok := v.(T)
	if !ok {
		var empty T
		return empty
	}

	return ret
}

func GetCurrentRuntime[T interface{}](r *RuleCheckerRuntime) T {
	if r == nil {
		var empty T
		return empty
	}

	ret, ok := r.currentRuleParameter.(T)
	if !ok {
		var empty T
		return empty
	}

	return ret
}

func MakeCurrentRuntime[T interface{}](r *RuleCheckerRuntime) *RuleCheckerRuntime {
	if r == nil {
		return nil
	}

	current := GetRuleRuntimeParameter[T](r)
	return &RuleCheckerRuntime{
		ruleParameter:        r.ruleParameter,
		currentRuleParameter: current,
	}
}
