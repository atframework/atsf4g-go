package lobbysvr_logic_item_impl

import (
	"fmt"
	"reflect"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func init() {
	var _ logic_condition.UserConditionManager = (*UserConditionManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_condition.UserConditionManager](func(_ctx *cd.RpcContext,
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

func (m *UserConditionManager) CheckStaticRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckRuleId(ctx *cd.RpcContext, ruleId int32) cd.RpcResult {
	result := m.CheckStaticRuleId(ctx, ruleId)
	if !result.IsOK() {
		return result
	}

	return m.CheckDynamicRuleId(ctx, ruleId)
}

func (m *UserConditionManager) CheckStaticRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		ruleType := reflect.TypeOf(rule.GetRuleType())
		checkHandle, exists := conditionRuleCheckers[ruleType]
		if !exists || checkHandle.StaticChecker == nil {
			continue
		}

		result := checkHandle.StaticChecker(m, ctx, rule)
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		ruleType := reflect.TypeOf(rule.GetRuleType())
		checkHandle, exists := conditionRuleCheckers[ruleType]
		if !exists || checkHandle.DynamicChecker == nil {
			continue
		}

		result := checkHandle.DynamicChecker(m, ctx, rule)
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckRules(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRule) cd.RpcResult {
	result := m.CheckStaticRules(ctx, rules)
	if !result.IsOK() {
		return result
	}

	return m.CheckDynamicRules(ctx, rules)
}

func (m *UserConditionManager) CheckDateTimeStaticLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
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

func (m *UserConditionManager) CheckDateTimeDynamicLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
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

func (m *UserConditionManager) CheckDateTimeLimit(ctx *cd.RpcContext, rules []*public_protocol_common.DConditionRuleRangeDatetime) cd.RpcResult {
	result := m.CheckDateTimeStaticLimit(ctx, rules)
	if !result.IsOK() {
		return result
	}

	return m.CheckDateTimeDynamicLimit(ctx, rules)
}

func (m *UserConditionManager) CheckBasicStaticLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult {
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
		result := m.CheckStaticRules(ctx, limit.GetRule())
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckBasicDynamicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult {
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
		result := m.CheckDynamicRules(ctx, limit.GetRule())
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckBasicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionBasicLimit) cd.RpcResult {
	result := m.CheckBasicStaticLimit(ctx, limit)
	if !result.IsOK() {
		return result
	}

	return m.CheckBasicDynamicLimit(ctx, limit)
}

func (m *UserConditionManager) CheckCounterStaticLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	// TODO: 实现静态计数器检查
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterDynamicLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	// TODO: 实现动态计数器检查
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterLimit(ctx *cd.RpcContext, limit *public_protocol_common.DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	result := m.CheckCounterStaticLimit(ctx, limit, storage)
	if !result.IsOK() {
		return result
	}

	return m.CheckCounterDynamicLimit(ctx, limit, storage)
}

func (m *UserConditionManager) AddCounter(ctx *cd.RpcContext, storage *public_protocol_common.DConditionCounterStorage, offset int64) cd.RpcResult {
	// TODO: 实现计数器增加逻辑
	return cd.CreateRpcResultOk()
}

type conditionRuleCheckHandle struct {
	StaticChecker  func(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult
	DynamicChecker func(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult
}

func buildRuleCheckers() map[reflect.Type]*conditionRuleCheckHandle {
	ret := map[reflect.Type]*conditionRuleCheckHandle{}

	ret[reflect.TypeOf(&public_protocol_common.DConditionRule_LoginChannel{})] = &conditionRuleCheckHandle{
		StaticChecker:  checkRuleUserLoginChannel,
		DynamicChecker: nil,
	}

	ret[reflect.TypeOf(&public_protocol_common.DConditionRule_SystemPlatform{})] = &conditionRuleCheckHandle{
		StaticChecker:  checkRuleUserSystemPlatform,
		DynamicChecker: nil,
	}

	ret[reflect.TypeOf(&public_protocol_common.DConditionRule_UserLevel{})] = &conditionRuleCheckHandle{
		StaticChecker:  checkRuleUserLevelStatic,
		DynamicChecker: checkRuleUserLevelDynamic,
	}

	ret[reflect.TypeOf(&public_protocol_common.DConditionRule_HasItem{})] = &conditionRuleCheckHandle{
		StaticChecker:  nil,
		DynamicChecker: checkRuleHasItem,
	}

	return ret
}

var conditionRuleCheckers = buildRuleCheckers()

func checkRuleUserLoginChannel(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult {
	loginChannel := uint64(m.GetOwner().GetLoginInfo().GetAccount().GetChannelId())

	if len(rule.GetLoginChannel().GetValues()) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, v := range rule.GetLoginChannel().GetValues() {
		if loginChannel == v {
			return cd.CreateRpcResultOk()
		}
	}

	// TODO: 错误码: 登入平台不满足要求
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
}

func checkRuleUserSystemPlatform(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult {
	loginChannel := uint64(m.GetOwner().GetClientInfo().GetSystemId())

	if len(rule.GetSystemPlatform().GetValues()) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, v := range rule.GetSystemPlatform().GetValues() {
		if loginChannel == v {
			return cd.CreateRpcResultOk()
		}
	}

	// TODO: 错误码: 登入平台不满足要求
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
}

func checkRuleUserLevelStatic(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult {
	if rule.GetUserLevel().GetLeft() <= 1 {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserBasicManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	userLevel := mgr.GetUserLevel()
	if int64(userLevel) < rule.GetUserLevel().GetLeft() {
		// TODO: 错误码: 最小等级不满足要求
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleUserLevelDynamic(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult {
	if rule.GetUserLevel().GetRight() <= 0 {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserBasicManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	userLevel := mgr.GetUserLevel()
	if int64(userLevel) > rule.GetUserLevel().GetRight() {
		// TODO: 错误码: 最大等级不满足要求
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleHasItem(m *UserConditionManager, ctx *cd.RpcContext, rule *public_protocol_common.DConditionRule) cd.RpcResult {
	if len(rule.GetHasItem().GetValues()) == 0 {
		return cd.CreateRpcResultOk()
	}

	values := rule.GetHasItem().GetValues()
	typeId := int32(values[0])
	if typeId == 0 {
		return cd.CreateRpcResultOk()
	}

	minCount := int64(0)
	maxCount := int64(0)

	if len(values) >= 2 {
		minCount = values[1]
	}
	if len(values) >= 3 {
		maxCount = values[2]
	}

	itemStats := m.GetOwner().GetItemTypeStatistics(typeId)
	if minCount > 0 && (itemStats == nil || itemStats.TotalCount < minCount) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(m.GetOwner().GetNotEnoughErrorCode(typeId)))
	}

	if maxCount < 0 && itemStats != nil && itemStats.TotalCount > 0 {
		// TODO: 错误码: 不允许拥有道具
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(m.GetOwner().GetNotEnoughErrorCode(typeId)))
	}

	if maxCount > 0 && (itemStats != nil && itemStats.TotalCount > maxCount) {
		// TODO: 错误码: 道具数量过多
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(m.GetOwner().GetNotEnoughErrorCode(typeId)))
	}

	return cd.CreateRpcResultOk()
}
