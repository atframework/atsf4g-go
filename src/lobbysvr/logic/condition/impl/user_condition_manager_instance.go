package lobbysvr_logic_condition_impl

import (
	"fmt"
	"math"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component-config"
	logical_time "github.com/atframework/atsf4g-go/component-logical_time"

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

	counterStorageIdAllocator int64
	dirtyCounterStorages      map[int64]*public_protocol_common.DConditionCounterStorage
	refreshCounterCache       map[int64]struct{}
}

func CreateUserConditionManager(owner *data.User) *UserConditionManager {
	ret := &UserConditionManager{
		UserModuleManagerBase:     *data.CreateUserModuleManagerBase(owner),
		counterStorageIdAllocator: 0,
		dirtyCounterStorages:      make(map[int64]*public_protocol_common.DConditionCounterStorage),
		refreshCounterCache:       make(map[int64]struct{}),
	}

	return ret
}

func (m *UserConditionManager) InitFromDB(_ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	m.counterStorageIdAllocator = dbUser.GetConditionData().GetStorageIdAllocator()
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) DumpToDB(_ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	dbUser.MutableConditionData().StorageIdAllocator = m.counterStorageIdAllocator
	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CreateInit(ctx cd.RpcContext, _versionType uint32) {
	m.counterStorageIdAllocator = ctx.GetSysNow().UnixMicro() - 1763568000000000
	if m.counterStorageIdAllocator <= 0 {
		m.counterStorageIdAllocator = 1
	}
}

func (m *UserConditionManager) LoginInit(ctx cd.RpcContext) {
	if m.counterStorageIdAllocator <= 0 {
		m.counterStorageIdAllocator = ctx.GetSysNow().UnixMicro() - 1763568000000000
	}
	if m.counterStorageIdAllocator <= 0 {
		m.counterStorageIdAllocator = 1
	}
}

func (m *UserConditionManager) RefreshLimitSecond(_ctx cd.RpcContext) {
	if m == nil {
		return
	}

	clear(m.refreshCounterCache)
}

func (m *UserConditionManager) AllocateConditionCounterStorageId() int64 {
	if m == nil {
		return 0
	}

	m.counterStorageIdAllocator++
	return m.counterStorageIdAllocator
}

func (m *UserConditionManager) AllocateCouterStorage(ctx cd.RpcContext, version int64) *public_protocol_common.DConditionCounterStorage {
	if m == nil {
		return nil
	}

	return &public_protocol_common.DConditionCounterStorage{
		CounterStorageId: m.AllocateConditionCounterStorageId(),
		CurrentVersion:   version,
	}
}

func (m *UserConditionManager) DumpConditionCounterData(to *public_protocol_pbdesc.DConditionCounterData) {
	if m == nil || to == nil {
		return
	}

	logic_condition.ForeachConditionCounterDelegate(m.GetOwner(), func(d logic_condition.UserConditionCounterDelegate) bool {
		if d == nil {
			return true
		}

		d.ForeachConditionCounter(func(storage *public_protocol_common.DConditionCounterStorage) bool {
			if storage == nil {
				return true
			}

			to.AppendCounterList(storage)
			return true
		})

		return true
	})
}

func (m *UserConditionManager) insertDirtyHandle() {
	if m == nil {
		return
	}

	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(_ctx cd.RpcContext, dirty *data.UserDirtyData) bool {
			ret := false
			if len(m.dirtyCounterStorages) <= 0 {
				return ret
			}

			dirtyData := dirty.MutableNormalDirtyChangeMessage()
			for _, counterStorage := range m.dirtyCounterStorages {
				dirtyData.MutableDirtyConditionCounter().AppendCounterList(counterStorage)
				ret = true
			}
			return ret
		},
		func(_ctx cd.RpcContext) {
			clear(m.dirtyCounterStorages)
		})
}

func (m *UserConditionManager) insertCounterDirty(storage *public_protocol_common.DConditionCounterStorage) {
	if m == nil || storage == nil {
		return
	}

	if storage.GetCounterStorageId() == 0 {
		return
	}

	m.dirtyCounterStorages[storage.GetCounterStorageId()] = storage
	m.insertDirtyHandle()
}

func (m *UserConditionManager) CheckStaticRuleId(ctx cd.RpcContext, ruleId int32, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	ruleCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelConditionPoolByConditionId(ruleId)
	if ruleCfg == nil {
		// 错误码: 条件规则不存在
		return cd.CreateRpcResultError(fmt.Errorf("rule config not found for rule id %d", ruleId), public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_RULE_ID_NOT_FOUND)
	}

	if len(ruleCfg.GetBasicLimit().GetRule()) > 0 {
		return m.CheckStaticRules(ctx, ruleCfg.GetBasicLimit().GetRule(), runtime)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRuleId(ctx cd.RpcContext, ruleId int32, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	ruleCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelConditionPoolByConditionId(ruleId)
	if ruleCfg == nil {
		// 错误码: 条件规则不存在
		return cd.CreateRpcResultError(fmt.Errorf("rule config not found for rule id %d", ruleId), public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_RULE_ID_NOT_FOUND)
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

func (m *UserConditionManager) CheckStaticRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		checkHandle := logic_condition.GetStaticRuleChecker(rule.GetRuleTypeTypeID())
		if checkHandle == nil {
			continue
		}

		mi := (logic_condition.UserConditionManager)(m)
		result := checkHandle(mi, ctx, rule, runtime.MakeCurrentRuntime(rule.GetRuleTypeTypeID()))
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckDynamicRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	if len(rules) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, rule := range rules {
		checkHandle := logic_condition.GetDynamicRuleChecker(rule.GetRuleTypeTypeID())
		if checkHandle == nil {
			continue
		}

		mi := (logic_condition.UserConditionManager)(m)
		result := checkHandle(mi, ctx, rule, runtime.MakeCurrentRuntime(rule.GetRuleTypeTypeID()))
		if !result.IsOK() {
			return result
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckRules(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	result := m.CheckStaticRules(ctx, rules, runtime)
	if !result.IsOK() {
		return result
	}

	return m.CheckDynamicRules(ctx, rules, runtime)
}

func (m *UserConditionManager) CheckDateTimeStaticLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult {
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

	// 错误码: 所有可用时段都已结束
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_DATETIME_ALREADY_END)
}

func (m *UserConditionManager) CheckDateTimeDynamicLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult {
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

	// 错误码: 未开始
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_DATETIME_NOT_START)
}

func (m *UserConditionManager) CheckDateTimeLimit(ctx cd.RpcContext, rules []*public_protocol_common.Readonly_DConditionRuleRangeDatetime) cd.RpcResult {
	result := m.CheckDateTimeStaticLimit(ctx, rules)
	if !result.IsOK() {
		return result
	}

	return m.CheckDateTimeDynamicLimit(ctx, rules)
}

func (m *UserConditionManager) CheckBasicStaticLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
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

func (m *UserConditionManager) CheckBasicDynamicLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
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

func (m *UserConditionManager) CheckBasicLimit(ctx cd.RpcContext, limit *public_protocol_common.Readonly_DConditionBasicLimit, runtime *logic_condition.RuleCheckerRuntime) cd.RpcResult {
	result := m.CheckBasicStaticLimit(ctx, limit, runtime)
	if !result.IsOK() {
		return result
	}

	return m.CheckBasicDynamicLimit(ctx, limit, runtime)
}

func (m *UserConditionManager) RefreshCounter(_ctx cd.RpcContext, now time.Time,
	limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) {
	if m == nil || storage == nil {
		return
	}

	if _, exists := m.refreshCounterCache[storage.GetCounterStorageId()]; exists {
		return
	}
	m.refreshCounterCache[storage.GetCounterStorageId()] = struct{}{}

	nowSec := now.Unix()
	versionedCounter := storage.MutableVersionCounter()

	isDirty := false
	dayStartOffset := time.Duration(limit.GetDayStartOffset()) * time.Second
	if limit.GetCounterVersion() != 0 && limit.GetCounterVersion() != storage.GetCurrentVersion() {
		storage.CurrentVersion = limit.GetCounterVersion()

		versionedCounter.DailyCounter = 0
		versionedCounter.WeeklyCounter = 0
		versionedCounter.MonthlyCounter = 0
		versionedCounter.SumCounter = 0
		versionedCounter.CustomCounter = 0

		updateTime := logical_time.GetTodayStartTimepoint(&dayStartOffset).Unix()
		versionedCounter.MutableDailyNextCheckpoint().Seconds = updateTime
		versionedCounter.MutableWeeklyNextCheckpoint().Seconds = logical_time.GetCurrentWeekStartTimepoint(&dayStartOffset).Unix()
		// TODO: 月度重置时间点计算
		if limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0 {
			customDurationSec := limit.GetCustomDuration().GetSeconds()
			cycles := (nowSec - limit.GetCustomStartTime().GetSeconds()) / customDurationSec
			versionedCounter.MutableCustomNextCheckpoint().Seconds = cycles*customDurationSec + limit.GetCustomStartTime().GetSeconds()
		} else {
			versionedCounter.CustomNextCheckpoint = nil
		}

		isDirty = true
	} else {
		if limit.GetDaily() > 0 && nowSec >= versionedCounter.GetDailyNextCheckpoint().GetSeconds() {
			versionedCounter.DailyCounter = 0
			versionedCounter.MutableDailyNextCheckpoint().Seconds = logical_time.GetNextDayStartTimepoint(&dayStartOffset).Unix()
			isDirty = true
		}

		if limit.GetWeekly() > 0 && nowSec >= versionedCounter.GetWeeklyNextCheckpoint().GetSeconds() {
			versionedCounter.WeeklyCounter = 0
			versionedCounter.MutableWeeklyNextCheckpoint().Seconds = logical_time.GetNextWeekStartTimepoint(&dayStartOffset).Unix()
			isDirty = true
		}

		// TODO: 月度重置时间点计算

		if limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0 && nowSec >= versionedCounter.GetCustomNextCheckpoint().GetSeconds() {
			versionedCounter.CustomCounter = 0

			customDurationSec := limit.GetCustomDuration().GetSeconds()
			cycles := (nowSec - limit.GetCustomStartTime().GetSeconds()) / customDurationSec
			versionedCounter.MutableCustomNextCheckpoint().Seconds = (cycles+1)*customDurationSec + limit.GetCustomStartTime().GetSeconds()
			isDirty = true
		} else {
			versionedCounter.CustomNextCheckpoint = nil
		}
	}

	// 动态计数器变更,只要有动态时间限制就视为有计数验证规则
	if limit.GetDynamicDuration().GetSeconds() > 0 {
		dynamicDurationSec := limit.GetDynamicDuration().GetSeconds()
		dynamicLastCheckpointSec := versionedCounter.GetDynamicLastCheckpoint().GetSeconds()
		if versionedCounter.GetDynamicLeftCounter() < limit.GetDynamicLimit() && nowSec >= dynamicLastCheckpointSec+dynamicDurationSec {
			maxAddCircle := (nowSec - dynamicLastCheckpointSec) / dynamicDurationSec

			if versionedCounter.GetDynamicLeftCounter()+maxAddCircle >= limit.GetDynamicLimit() {
				// 加满
				versionedCounter.MutableDynamicLastCheckpoint().Seconds = nowSec
				versionedCounter.DynamicLeftCounter = limit.GetDynamicLimit()
			} else {
				// 未加满
				versionedCounter.DynamicLeftCounter += maxAddCircle
				versionedCounter.MutableDynamicLastCheckpoint().Seconds = dynamicLastCheckpointSec + maxAddCircle*dynamicDurationSec
			}

			isDirty = true
		}
	}

	if isDirty {
		m.insertCounterDirty(storage)
	}
}

func (m *UserConditionManager) HasCounterLimit(limit *public_protocol_common.DConditionCounterLimit) bool {
	if limit == nil {
		return false
	}

	return limit.GetSum() > 0 ||
		limit.GetDaily() > 0 ||
		limit.GetWeekly() > 0 ||
		limit.GetMonthly() > 0 ||
		(limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0) ||
		(limit.GetDynamicDuration().GetSeconds() > 0)
}

func (m *UserConditionManager) HasCounterLimitCfg(limit *public_protocol_common.Readonly_DConditionCounterLimit) bool {
	if limit == nil {
		return false
	}

	return limit.GetSum() > 0 ||
		limit.GetDaily() > 0 ||
		limit.GetWeekly() > 0 ||
		limit.GetMonthly() > 0 ||
		(limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0) ||
		(limit.GetDynamicDuration().GetSeconds() > 0)
}

func (m *UserConditionManager) CalculateCounterLimitCfgMaxLeftTimes(limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) int64 {
	if limit == nil || storage == nil {
		return 0
	}

	minTimes := int64(math.MaxInt64)
	if limit.GetSum() > 0 {
		if left := limit.GetSum() - storage.GetVersionCounter().GetSumCounter(); left < minTimes {
			minTimes = left
		}
	}

	if limit.GetDaily() > 0 {
		if left := limit.GetDaily() - storage.GetVersionCounter().GetDailyCounter(); left < minTimes {
			minTimes = left
		}
	}

	if limit.GetWeekly() > 0 {
		if left := limit.GetWeekly() - storage.GetVersionCounter().GetWeeklyCounter(); left < minTimes {
			minTimes = left
		}
	}

	if limit.GetMonthly() > 0 {
		if left := limit.GetMonthly() - storage.GetVersionCounter().GetMonthlyCounter(); left < minTimes {
			minTimes = left
		}
	}

	if limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0 {
		if left := limit.GetCustomLimit() - storage.GetVersionCounter().GetCustomCounter(); left < minTimes {
			minTimes = left
		}
	}

	if limit.GetDynamicDuration().GetSeconds() > 0 {
		if left := storage.GetVersionCounter().GetDynamicLeftCounter(); left < minTimes {
			minTimes = left
		}
	}

	if minTimes == math.MaxInt64 || minTimes < 0 {
		return 0
	}

	return minTimes
}

func (m *UserConditionManager) CheckCounterStaticLimit(ctx cd.RpcContext, now time.Time, offset int64,
	limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	if offset < 0 {
		offset = 0
	}

	m.RefreshCounter(ctx, now, limit, storage)

	if limit == nil {
		return cd.CreateRpcResultOk()
	}

	if storage == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 静态计数器检查
	if limit.GetSum() > 0 && storage.GetVersionCounter().GetSumCounter()+offset > limit.GetSum() {
		// 错误码: 计数器总量超限
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_SUM_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterDynamicLimit(ctx cd.RpcContext, now time.Time, offset int64,
	limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	if offset < 0 {
		offset = 0
	}

	m.RefreshCounter(ctx, now, limit, storage)

	if limit == nil {
		return cd.CreateRpcResultOk()
	}

	if storage == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 动态计数器检查
	if limit.GetDaily() > 0 && storage.GetVersionCounter().GetDailyCounter()+offset > limit.GetDaily() {
		// 错误码: 计数器日限超限
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_DAILY_LIMIT)
	}

	if limit.GetWeekly() > 0 && storage.GetVersionCounter().GetWeeklyCounter()+offset > limit.GetWeekly() {
		// 错误码: 计数器周限超限
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_WEEKLY_LIMIT)
	}

	if limit.GetMonthly() > 0 && storage.GetVersionCounter().GetMonthlyCounter()+offset > limit.GetMonthly() {
		// 错误码: 计数器月限超限
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_MONTHLY_LIMIT)
	}

	if limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0 &&
		storage.GetVersionCounter().GetCustomCounter()+offset > limit.GetCustomLimit() {
		// 错误码: 计数器自定义限超限
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_CUSTOM_LIMIT)
	}

	if limit.GetDynamicDuration().GetSeconds() > 0 {
		if storage.GetVersionCounter().GetDynamicLeftCounter() < offset {
			// 错误码: 计数器动态限超限
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_DYNAMIC_LIMIT)
		}
	}

	return cd.CreateRpcResultOk()
}

func (m *UserConditionManager) CheckCounterLimit(ctx cd.RpcContext, now time.Time, offset int64,
	limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	result := m.CheckCounterStaticLimit(ctx, now, offset, limit, storage)
	if !result.IsOK() {
		return result
	}

	return m.CheckCounterDynamicLimit(ctx, now, offset, limit, storage)
}

func (m *UserConditionManager) AddCounter(ctx cd.RpcContext, now time.Time, offset int64,
	limit *public_protocol_common.Readonly_DConditionCounterLimit,
	storage *public_protocol_common.DConditionCounterStorage,
) cd.RpcResult {
	if storage == nil || offset <= 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	m.RefreshCounter(ctx, now, limit, storage)

	// TODO: 实现计数器增加逻辑
	storage.MutableVersionCounter().DailyCounter += offset
	storage.MutableVersionCounter().WeeklyCounter += offset
	storage.MutableVersionCounter().MonthlyCounter += offset
	storage.MutableVersionCounter().SumCounter += offset

	if limit.GetCustomLimit() > 0 && limit.GetCustomDuration().GetSeconds() > 0 {
		storage.MutableVersionCounter().CustomCounter += offset
	}

	if limit.GetDynamicDuration().GetSeconds() > 0 {
		if storage.MutableVersionCounter().DynamicLeftCounter >= offset {
			storage.MutableVersionCounter().DynamicLeftCounter -= offset
		} else {
			storage.MutableVersionCounter().DynamicLeftCounter = 0
		}
	}

	m.insertCounterDirty(storage)
	return cd.CreateRpcResultOk()
}
