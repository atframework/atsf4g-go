package lobbysvr_logic_condition_data

import (
	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

type RuleCheckerParameterPair struct {
	key   lu.TypeID
	value interface{}
}

type RuleCheckerRuntime struct {
	ruleParameter        map[lu.TypeID]interface{}
	currentRuleParameter interface{}
}

func (r *RuleCheckerRuntime) GetRuleParameter(t lu.TypeID) interface{} {
	if r == nil {
		return nil
	}

	return r.ruleParameter[t]
}

func (r *RuleCheckerRuntime) MakeCurrentRuntime(t lu.TypeID) *RuleCheckerRuntime {
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
		key:   lu.GetTypeIDOf[T](),
		value: v,
	}
}

func CreateRuleCheckerRuntime(params ...RuleCheckerParameterPair) *RuleCheckerRuntime {
	if len(params) == 0 {
		return nil
	}

	m := make(map[lu.TypeID]interface{}, len(params))
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

	v := r.GetRuleParameter(lu.GetTypeIDOf[T]())
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
