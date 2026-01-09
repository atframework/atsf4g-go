package lobbysvr_logic_mall_impl

import (
	"fmt"
	"reflect"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	config "github.com/atframework/atsf4g-go/component-config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_mall "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mall"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

var userManagerReflectType reflect.Type

func init() {
	var _ logic_mall.UserMallManager = (*UserMallManager)(nil)
	userManagerReflectType = lu.GetStaticReflectType[UserMallManager]()

	var _ logic_condition.UserConditionCounterDelegate = (*UserMallManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_mall.UserMallManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return CreateUserMallManager(owner)
	})

	// 注册condition counter delegate
	logic_condition.RegisterConditionCounterDelegate[logic_mall.UserMallManager](func(u *data.User) logic_condition.UserConditionCounterDelegate {
		mgr := data.UserGetModuleManager[logic_mall.UserMallManager](u)
		if mgr == nil {
			return nil
		}

		finalMgr, ok := mgr.(*UserMallManager)
		if !ok || finalMgr == nil {
			return nil
		}

		return finalMgr
	})
	registerCondition()
}

func (m *UserMallManager) GetReflectType() reflect.Type {
	return userManagerReflectType
}

type UserMallManager struct {
	owner *data.User
	data.UserModuleManagerBase

	productData map[int32]*private_protocol_pbdesc.UserMallProductData
	// Dirty 数据
	dirtyMallProduct map[int32]struct{}
}

func CreateUserMallManager(owner *data.User) *UserMallManager {
	return &UserMallManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),

		productData:      make(map[int32]*private_protocol_pbdesc.UserMallProductData),
		dirtyMallProduct: make(map[int32]struct{}),
	}
}

func (m *UserMallManager) InitFromDB(ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	mallData := _dbUser.GetMallData()
	for _, data := range mallData.GetMallData() {
		m.productData[data.GetProductData().GetProductId()] = data
	}

	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserMallManager) DumpToDB(ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	_dbUser.MutableMallData().MallData = make([]*private_protocol_pbdesc.UserMallProductData, 0, len(m.productData))
	for _, data := range m.productData {
		_dbUser.MutableMallData().MallData = append(_dbUser.MutableMallData().MallData, &private_protocol_pbdesc.UserMallProductData{
			ProductData:             data.GetProductData(),
			TotalCounterStorageData: data.GetTotalCounterStorageData(),
		})
	}
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserMallManager) CreateInit(ctx cd.RpcContext, _versionType uint32) {
	// Nothing
}

func (m *UserMallManager) LoginInit(ctx cd.RpcContext) {
}

func (m *UserMallManager) GetCounterSizeCapacity() int32 {
	if m == nil {
		return 0
	}

	return int32(len(m.productData))
}

func (m *UserMallManager) ForeachConditionCounter(f func(storage *public_protocol_common.DConditionCounterStorage) bool) {
	if m == nil || f == nil {
		return
	}

	for _, data := range m.productData {
		if data == nil || data.ProductData == nil {
			continue
		}

		if data.GetTotalCounterStorageData() != nil {
			if !f(data.GetTotalCounterStorageData()) {
				return
			}
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////

func (m *UserMallManager) MallPurchase(ctx cd.RpcContext, productId int32, purchasePriority int32,
	expectCostItems []*public_protocol_common.DItemBasic, rspBody *service_protocol.SCMallPurchaseRsp) int32 {
	// 先判断商城是否解锁
	productRow := config.GetConfigManager().GetCurrentConfigGroup().GetExcelMallProductByProductIdPurchasePriority(productId, purchasePriority)
	if productRow == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_PRODUCT_NOT_FOUND)
	}

	if !productRow.GetIsOnSale() {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_PRODUCT_NOT_ON_SELL)
	}

	mallRow := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().
		GetMallByMallSheet(productRow.GetMallSheetId())

	if mallRow == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_PRODUCT_NOT_ON_SELL)
	}

	// 商城解锁条件
	if logic_condition.HasLimitData(mallRow.GetUnlockCondition()) {
		conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
		if conditionMgr == nil {
			return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}

		rpcResult := conditionMgr.CheckBasicLimit(ctx, mallRow.GetUnlockCondition(), logic_condition.CreateRuleCheckerRuntime())
		if !rpcResult.IsOK() {
			return rpcResult.GetResponseCode()
		}
	}

	// 商品解锁条件
	{
		if logic_condition.HasLimitData(productRow.GetUnlockCondition()) {
			conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
			if conditionMgr == nil {
				return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			}

			rpcResult := conditionMgr.CheckBasicLimit(ctx, productRow.GetUnlockCondition(), logic_condition.CreateRuleCheckerRuntime())
			if !rpcResult.IsOK() {
				return rpcResult.GetResponseCode()
			}
		}

		if logic_condition.HasLimitData(productRow.GetPurchaseCondition()) {
			conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
			if conditionMgr == nil {
				return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			}

			rpcResult := conditionMgr.CheckBasicLimit(ctx, productRow.GetPurchaseCondition(), logic_condition.CreateRuleCheckerRuntime())
			if !rpcResult.IsOK() {
				return rpcResult.GetResponseCode()
			}
		}
	}

	conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
	if conditionMgr == nil {
		ctx.LogError("UserConditionManager not found",
			"product_id", productRow.GetProductId(),
		)
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 检查次数
	productData, ok := m.productData[productRow.GetProductId()]
	if !ok || productData == nil {
		// 不存在则创建
		totalCounterStorage := conditionMgr.AllocateCouterStorage(ctx, productRow.GetTotalCounterCondition().GetCounterVersion())
		if totalCounterStorage == nil {
			ctx.LogError("AllocateCouterStorage failed",
				"product_id", productRow.GetProductId(),
			)
			return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_PRODUCT_INIT_FAILED)
		}
		productData = &private_protocol_pbdesc.UserMallProductData{
			ProductData: &public_protocol_pbdesc.DMallProductData{
				ProductId:             productRow.GetProductId(),
				TotalCounterStorageId: totalCounterStorage.GetCounterStorageId(),
			},
			TotalCounterStorageData: totalCounterStorage,
		}
		m.productData[productRow.GetProductId()] = productData
		m.dirtyMallProduct[productRow.GetProductId()] = struct{}{}
		m.insertDirtyHandle()
	}
	checkResult := conditionMgr.CheckCounterLimit(ctx, ctx.GetNow(), int64(1),
		productRow.GetTotalCounterCondition(), productData.MutableTotalCounterStorageData())
	if checkResult.IsError() {
		return checkResult.GetResponseCode()
	}

	// 检查消耗
	var subGuard []*data.ItemSubGuard
	if len(productRow.GetPurchaseCost()) != 0 {
		result := m.GetOwner().CheckCostItemCfg(ctx, expectCostItems, productRow.GetPurchaseCost())
		if result.IsError() {
			ctx.LogError("check cost item failed", "error", result.GetResponseCode())
			return result.GetResponseCode()
		}

		subGuard, result = m.GetOwner().CheckSubItem(ctx, expectCostItems)
		if result.IsError() {
			ctx.LogError("check sub item failed", "error", result.GetResponseCode())
			return result.GetResponseCode()
		}

	}

	// 检查发放
	rewardInstances, result := m.GetOwner().GenerateMultipleItemInstancesFromCfgOffset(ctx, productRow.GetProductItems(), true)
	if result.IsError() {
		result.LogError(ctx, "generate reward item instances failed")
		return result.GetResponseCode()
	}

	addGuard, result := m.GetOwner().CheckAddItem(ctx, rewardInstances)
	if result.IsError() {
		result.LogError(ctx, "add golden pot upgrade reward items failed")
		return result.GetResponseCode()
	}

	// 加次数
	conditionMgr.AddCounter(ctx, ctx.GetNow(), 1,
		productRow.GetTotalCounterCondition(),
		productData.MutableTotalCounterStorageData())

	// 扣除消耗
	if len(subGuard) != 0 {
		m.GetOwner().SubItem(ctx, subGuard, &data.ItemFlowReason{
			MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_MALL),
			MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_MALL_PURCHASE_COST),
			Parameter:   int64(productRow.GetProductId()),
		})
	}

	// 发放奖励
	m.GetOwner().AddItem(ctx, addGuard, &data.ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_MALL),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_MALL_PURCHASE_REWARD),
		Parameter:   int64(productRow.GetProductId()),
	})

	return 0
}

func (m *UserMallManager) FetchData() *public_protocol_pbdesc.DUserMallData {
	ret := &public_protocol_pbdesc.DUserMallData{}
	ret.ProductData = make([]*public_protocol_pbdesc.DMallProductData, 0, len(m.productData))
	for _, data := range m.productData {
		ret.ProductData = append(ret.ProductData, data.GetProductData())
	}
	return ret
}

func (m *UserMallManager) insertDirtyHandle() {
	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(ctx cd.RpcContext, dirty *data.UserDirtyData) (ret bool) {
			dirtyData := dirty.MutableNormalDirtyChangeMessage()
			ret = false
			for id := range m.dirtyMallProduct {
				v, ok := m.productData[id]
				if ok && v != nil {
					dirtyData.MutableDirtyMall().AppendProductData(v.GetProductData())
					ret = true
				}
			}
			return ret
		},
		func(ctx cd.RpcContext) {
			clear(m.dirtyMallProduct)
		},
	)
}

func (m *UserMallManager) GetProductCounter(productId int32) *public_protocol_common.DConditionCounterStorage {
	productData, ok := m.productData[productId]
	if ok && productData != nil {
		return productData.GetTotalCounterStorageData()
	}
	return nil
}

func registerCondition() {
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_MallProductPurchaseCountAll(),
		nil, checkRuleMallProductPurchaseCountAll)
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_MallProductPurchaseCountDaily(),
		nil, checkRuleMallProductPurchaseCountDaily)
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_MallProductPurchaseCountWeekly(),
		nil, checkRuleLotteryPoolGroupSumCountWeekly)
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_MallProductPurchaseCountMonthly(),
		nil, checkRuleMallProductPurchaseCountMonthly)
	logic_condition.AddRuleChecker(public_protocol_common.GetReflectTypeDConditionRule_MallProductPurchaseCountCustom(),
		nil, checkRuleMallProductPurchaseCountCustom)
}

func checkConditionCountLimit(rule *public_protocol_common.Readonly_DConditionRuleMallProductPurchaseCount, count int64) bool {
	if count < rule.GetMinValue() {
		return false
	}

	if rule.GetMaxValue() < 0 && count > 0 {
		return false
	}

	if rule.GetMaxValue() > 0 && count > rule.GetMaxValue() {
		return false
	}

	return true
}

func checkRuleMallProductPurchaseCountAll(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule == nil {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_mall.UserMallManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	counter := mgr.GetProductCounter(rule.GetMallProductPurchaseCountAll().GetProductId())
	count := counter.VersionCounter.GetSumCounter()

	if !checkConditionCountLimit(rule.GetMallProductPurchaseCountAll(), count) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_SUM_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleMallProductPurchaseCountDaily(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule == nil {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_mall.UserMallManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	counter := mgr.GetProductCounter(rule.GetMallProductPurchaseCountDaily().GetProductId())
	count := int64(0)
	if counter != nil && ctx.GetNow().Before(counter.VersionCounter.GetDailyNextCheckpoint().AsTime()) {
		count = counter.VersionCounter.GetDailyCounter()
	}

	if !checkConditionCountLimit(rule.GetMallProductPurchaseCountDaily(), count) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_DAILY_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleLotteryPoolGroupSumCountWeekly(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule == nil {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_mall.UserMallManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	counter := mgr.GetProductCounter(rule.GetMallProductPurchaseCountWeekly().GetProductId())
	count := int64(0)
	if counter != nil && ctx.GetNow().Before(counter.VersionCounter.GetWeeklyNextCheckpoint().AsTime()) {
		count = counter.VersionCounter.GetWeeklyCounter()
	}

	if !checkConditionCountLimit(rule.GetMallProductPurchaseCountWeekly(), count) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_WEEKLY_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleMallProductPurchaseCountMonthly(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule == nil {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_mall.UserMallManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	counter := mgr.GetProductCounter(rule.GetMallProductPurchaseCountMonthly().GetProductId())
	count := int64(0)
	if counter != nil && ctx.GetNow().Before(counter.VersionCounter.GetMonthlyNextCheckpoint().AsTime()) {
		count = counter.VersionCounter.GetMonthlyCounter()
	}

	if !checkConditionCountLimit(rule.GetMallProductPurchaseCountMonthly(), count) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_MONTHLY_LIMIT)
	}

	return cd.CreateRpcResultOk()
}

func checkRuleMallProductPurchaseCountCustom(m logic_condition.UserConditionManager, ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DConditionRule, runtime *logic_condition.RuleCheckerRuntime,
) cd.RpcResult {
	if rule == nil {
		return cd.CreateRpcResultOk()
	}

	mgr := data.UserGetModuleManager[logic_mall.UserMallManager](m.GetOwner())
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	counter := mgr.GetProductCounter(rule.GetMallProductPurchaseCountCustom().GetProductId())
	count := int64(0)
	if counter != nil && ctx.GetNow().Before(counter.VersionCounter.GetCustomNextCheckpoint().AsTime()) {
		count = counter.VersionCounter.GetCustomCounter()
	}

	if !checkConditionCountLimit(rule.GetMallProductPurchaseCountCustom(), count) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_CONDITION_CUSTOM_LIMIT)
	}

	return cd.CreateRpcResultOk()
}
