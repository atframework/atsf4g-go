package lobbysvr_logic_mall_impl

import (
	"fmt"
	"math/rand/v2"

	config "github.com/atframework/atsf4g-go/component/config"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	logical_time "github.com/atframework/atsf4g-go/component/logical_time"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_config "github.com/atframework/atsf4g-go/component/protocol/public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
)

type randomProductUnit struct {
	productId int32
	weight    int32
}

func (m *UserMallManager) refreshMallRandomSheet(ctx cd.RpcContext, mallSheetId int32) cd.RpcResult {
	ctx.LogDebug("user_mall_instance: Refreshing mall random sheet", "mallSheetId", mallSheetId)
	sheetCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelMallSheetByMallSheetId(mallSheetId)
	if sheetCfg == nil {
		return cd.CreateRpcResultError(fmt.Errorf("invalid mall sheet id %d", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_FOUND)
	}

	if sheetCfg.GetRandomCfg().GetIsRandom() == false {
		return cd.CreateRpcResultError(fmt.Errorf("mall sheet id %d is not random mall sheet", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_RANDOM)
	}

	randomCount := sheetCfg.GetRandomCfg().GetMarketLocationCount()
	if randomCount <= 0 {
		return cd.CreateRpcResultError(fmt.Errorf("invalid random count %d for mall sheet id %d", randomCount, mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_RANDOM_COUNT_INVALID)
	}

	productCfgs := config.GetConfigManager().GetCurrentConfigGroup().GetExcelMallProductByMallSheetId(mallSheetId)
	if len(productCfgs) == 0 {
		return cd.CreateRpcResultError(fmt.Errorf("no product config for mall sheet id %d", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_FOUND)
	}

	randomProductList := make([]*randomProductUnit, 0, len(productCfgs))
	for _, productCfg := range productCfgs {
		m.randomProductMap[productCfg.GetProductId()] = false
		if m.CheckProductRandomCond(ctx, productCfg) {
			randomProductList = append(randomProductList, &randomProductUnit{
				productId: productCfg.GetProductId(),
				weight:    productCfg.GetRandomCfg().GetRandomWeight(),
			})
		} else {
			ctx.LogDebug("user_mall_instance: product skip", "productId", productCfg.GetProductId(), "mallSheetId", mallSheetId)
		}
	}

	// 过滤有效元素（weight > 0），计算总权重
	valid := make([]*randomProductUnit, 0, len(randomProductList))
	var sumWeight int32
	for _, p := range randomProductList {
		if p.weight > 0 {
			valid = append(valid, p)
			sumWeight += p.weight
		}
	}

	if len(valid) == 0 {
		return cd.CreateRpcResultError(fmt.Errorf("no valid product for mall sheet id %d", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_FOUND)
	}

	selectedProducts := make([]*randomProductUnit, 0, randomCount)

	// 有效元素数量不足 randomCount 时全取
	if int(randomCount) >= len(valid) {
		selectedProducts = append(selectedProducts, valid...)
	} else {
		// 每轮按权重随机选一个，选中后从候选集移除
		for i := 0; i < int(randomCount) && len(valid) > 0; i++ {
			selectWeight := rand.Int32N(sumWeight)
			idx := 0
			for ; idx < len(valid); idx++ {
				if selectWeight < valid[idx].weight {
					break
				}
				selectWeight -= valid[idx].weight
			}
			if idx >= len(valid) {
				idx = len(valid) - 1
			}
			selected := valid[idx]
			selectedProducts = append(selectedProducts, selected)
			sumWeight -= selected.weight
			// 末尾交换删除，O(1)
			valid[idx] = valid[len(valid)-1]
			valid = valid[:len(valid)-1]
		}
	}

	mallRandomSheet := m.randomSheetData[mallSheetId]
	if mallRandomSheet == nil {
		mallRandomSheet = &private_protocol_pbdesc.UserMallRandomSheetData{
			MallSheetId: mallSheetId,
		}
		m.randomSheetData[mallSheetId] = mallRandomSheet
	}
	_, nextRefreshTime := m.GetNextRefreshTimepoint(ctx, mallSheetId)
	mallRandomSheet.NextRefreshTimepoint = nextRefreshTime
	mallRandomSheet.Product = make([]*public_protocol_pbdesc.DUserMallRandomProductData, 0, len(selectedProducts))
	for _, p := range selectedProducts {
		mallRandomSheet.Product = append(mallRandomSheet.Product, &public_protocol_pbdesc.DUserMallRandomProductData{
			ProductId: p.productId,
		})
		ctx.LogDebug("user_mall_instance: Selected product", "productId", p.productId, "mallSheetId", mallSheetId)
		m.randomProductMap[p.productId] = true
	}

	if mallRandomSheet.RefreshCountResetTimepoint == 0 || mallRandomSheet.RefreshCountResetTimepoint <= ctx.GetNow().Unix() {
		ctx.LogDebug("user_mall_instance: refresh RefreshCount")
		mallRandomSheet.RefreshCount = 0
		mallRandomSheet.RefreshCountResetTimepoint = logical_time.GetNextDayStartTimepoint(nil).Unix()
	}

	ctx.LogDebug("user_mall_instance: Refreshing mall finish", "mallSheetId", mallSheetId, "nextRefreshTime", nextRefreshTime, "selectedProductCount", len(selectedProducts))

	m.dirtyRandomSheet = append(m.dirtyRandomSheet, mallSheetId)
	m.insertDirtyHandle()
	return cd.CreateRpcResultOk()
}

func (m *UserMallManager) GetNextRefreshTimepoint(ctx cd.RpcContext, mallSheetId int32) (cd.RpcResult, int64) {
	sheetCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelMallSheetByMallSheetId(mallSheetId)
	if sheetCfg == nil {
		return cd.CreateRpcResultError(fmt.Errorf("invalid mall sheet id %d", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_FOUND), 0
	}

	if sheetCfg.GetRandomCfg().GetIsRandom() == false {
		return cd.CreateRpcResultError(fmt.Errorf("mall sheet id %d is not random mall sheet", mallSheetId), public_protocol_pbdesc.EnErrorCode_EN_ERR_MALL_SHEET_NOT_RANDOM), 0
	}

	startTime := sheetCfg.GetRandomCfg().GetStartTimepoint().GetSeconds()
	now := ctx.GetNow().Unix()

	if now < startTime {
		return cd.CreateRpcResultOk(), startTime
	}

	period := (now-startTime)/int64(sheetCfg.GetRandomCfg().GetDuration()) + 1

	nextFreshTime := startTime + period*int64(sheetCfg.GetRandomCfg().GetDuration())

	return cd.CreateRpcResultOk(), nextFreshTime
}

func (m *UserMallManager) CheckProductRandomCond(ctx cd.RpcContext, productCfg *public_protocol_config.Readonly_ExcelMallProduct) bool {

	conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())

	if conditionMgr == nil {
		return false
	}

	// 商品可见条件
	if logic_condition.HasLimitData(productCfg.GetVisibleCondition()) {
		rpcResult := conditionMgr.CheckBasicLimit(ctx, productCfg.GetVisibleCondition(), nil)
		if !rpcResult.IsOK() {
			ctx.LogDebug("Product not visible",
				"product_id", productCfg.GetProductId(),
				"mall_sheet_id", productCfg.GetMallSheetId(),
			)
			return false
		}
	}

	// 商品解锁条件
	if logic_condition.HasLimitData(productCfg.GetUnlockCondition()) {
		rpcResult := conditionMgr.CheckBasicLimit(ctx, productCfg.GetUnlockCondition(), nil)
		if !rpcResult.IsOK() {
			ctx.LogDebug("Product not unlocked",
				"product_id", productCfg.GetProductId(),
				"mall_sheet_id", productCfg.GetMallSheetId(),
			)
			return false
		}
	}

	// 过滤不满足条件的商品配置
	productData, ok := m.productData[productCfg.GetProductId()]
	if !ok {
		// 不存在 说明从未购买过
		return true
	}

	conditionMgr.RefreshCounter(ctx, ctx.GetNow(), productCfg.GetTotalCounterCondition(), productData.MutableTotalCounterStorageData())
	conditionMgr.ResetCustomLimitCounter(ctx, ctx.GetNow(), productData.MutableTotalCounterStorageData())

	// 是否可购买
	checkResult := conditionMgr.CheckCounterLimit(ctx, ctx.GetNow(), 1,
		productCfg.GetTotalCounterCondition(), productData.MutableTotalCounterStorageData())
	if checkResult.IsError() {
		ctx.LogDebug("Product random condition not match",
			"product_id", productCfg.GetProductId(),
			"mall_sheet_id", productCfg.GetMallSheetId(),
			"reason", "total counter limit not match",
		)
		return false
	}

	return true
}
