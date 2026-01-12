package lobbysvr_data

import (
	"slices"

	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_log "github.com/atframework/atsf4g-go/component-protocol-private/log/protocol/log"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

func (u *User) GetItemManager(typeId int32) UserItemManagerImpl {
	if u.itemManagerList == nil {
		return nil
	}
	// Binary search
	index, found := slices.BinarySearchFunc(u.itemManagerList, typeId, func(a userItemManagerWrapper, b int32) int {
		if a.idRange.endTypeId <= b {
			return -1
		}
		if a.idRange.beginTypeId > b {
			return 1
		}
		return 0
	})

	if index < 0 || index >= len(u.itemManagerList) || !found {
		return nil
	}

	return u.itemManagerList[index].manager
}

func (u *User) AddItem(ctx cd.RpcContext, itemOffset []*ItemAddGuard, reason *ItemFlowReason) Result {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*ItemAddGuard
	})
	for _, offset := range itemOffset {
		if offset == nil {
			continue
		}
		typeId := offset.Item.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*ItemAddGuard
			}{
				data: make([]*ItemAddGuard, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	result := cd.CreateRpcResultOk()

	ossLog := &private_protocol_log.OperationSupportSystemLog{}
	for mgr, group := range splitByMgr {
		subResult := mgr.AddItem(ctx, group.data, reason)
		if subResult.IsError() {
			subResult.LogError(ctx, "user add item failed")
			result = subResult
		}
		for _, itemGuard := range group.data {
			for _, addedItem := range itemGuard.GetAddedItems() {
				ossLog.Detail = nil
				data := ossLog.MutableItemFlow()
				data.OperationType = private_protocol_log.OSSItemFlow_EN_OSS_ITEM_FLOW_OPERATION_TYPE_ADD
				data.OperationCount = addedItem.GetItemBasic().GetCount()
				data.ItemId = addedItem.GetItemBasic().GetTypeId()
				data.AfterCount = mgr.GetTypeStatistics(ctx, addedItem.GetItemBasic().GetTypeId()).GetTotalCount()
				data.MajorReason = public_protocol_common.EnItemFlowReasonMajorType(reason.GetMajorReason())
				data.MinorReason = public_protocol_common.EnItemFlowReasonMinorType(reason.GetMinorReason())
				data.Parameter = reason.GetParameter()
				data.Result = subResult.GetResponseCode()
				u.SendUserOssLog(ctx, ossLog)
			}
		}
	}

	return result
}

func (u *User) SubItem(ctx cd.RpcContext, itemOffset []*ItemSubGuard, reason *ItemFlowReason) Result {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*ItemSubGuard
	})
	for _, offset := range itemOffset {
		if offset == nil {
			continue
		}
		typeId := offset.Item.GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*ItemSubGuard
			}{
				data: make([]*ItemSubGuard, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	result := cd.CreateRpcResultOk()

	ossLog := &private_protocol_log.OperationSupportSystemLog{}
	for mgr, group := range splitByMgr {
		subResult := mgr.SubItem(ctx, group.data, reason)
		if subResult.IsError() {
			subResult.LogError(ctx, "user sub item failed")
			result = subResult
		}
		for _, itemGuard := range group.data {
			ossLog.Detail = nil
			data := ossLog.MutableItemFlow()
			data.OperationType = private_protocol_log.OSSItemFlow_EN_OSS_ITEM_FLOW_OPERATION_TYPE_SUB
			data.OperationCount = itemGuard.Item.GetCount()
			data.ItemId = itemGuard.Item.GetTypeId()
			data.AfterCount = mgr.GetTypeStatistics(ctx, itemGuard.Item.GetTypeId()).GetTotalCount()
			data.MajorReason = public_protocol_common.EnItemFlowReasonMajorType(reason.GetMajorReason())
			data.MinorReason = public_protocol_common.EnItemFlowReasonMinorType(reason.GetMinorReason())
			data.Parameter = reason.GetParameter()
			data.Result = subResult.GetResponseCode()
			u.SendUserOssLog(ctx, ossLog)
		}
	}

	return result
}

func (u *User) UseItem(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result) {
	typeId := itemBasic.GetTypeId()
	if itemBasic.GetCount() <= 0 {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}
	row := config.GetConfigManager().GetCurrentConfigGroup().GetExcelItemByItemId(typeId)
	if row == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}
	if row.GetUseAction().GetActionTypeOneofCase() == 0 {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_CANNOT_USE)
	}
	result := u.checkUseItem(ctx, itemBasic, row.GetUseAction(), useParam)
	if result.IsError() {
		return nil, result
	}

	guard, result := u.CheckSubItem(ctx, []*public_protocol_common.DItemBasic{itemBasic})
	if result.IsError() {
		return nil, result
	}

	u.SubItem(ctx, guard, reason)

	return u.useItemInner(ctx, itemBasic, row.GetUseAction(), useParam, reason)
}

func (u *User) checkUseItem(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useAction *public_protocol_common.Readonly_DItemUseAction, useParam *public_protocol_common.DItemUseParam) Result {
	handlers, exists := useItemHandlerRegistry[useAction.GetActionTypeOneofCase()]
	if !exists || handlers == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_CANNOT_USE)
	}
	if handlers.CheckUseItemHandler != nil {
		return handlers.CheckUseItemHandler(ctx, u, useAction, itemBasic, useParam)
	}
	return cd.CreateRpcResultOk()
}

func (u *User) useItemInner(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useAction *public_protocol_common.Readonly_DItemUseAction, useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result) {
	handlers, exists := useItemHandlerRegistry[useAction.GetActionTypeOneofCase()]
	if !exists || handlers == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_CANNOT_USE)
	}
	if handlers.UseItemHandler != nil {
		return handlers.UseItemHandler(ctx, u, useAction, itemBasic, useParam, reason)
	}
	return nil, cd.CreateRpcResultOk()
}

func (u *User) GenerateItemInstanceFromCfgOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.Readonly_DItemOffset) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemOffset.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromCfgOffset(ctx, itemOffset)
}

func (u *User) GenerateItemInstanceFromOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.DItemOffset) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemOffset.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromOffset(ctx, itemOffset)
}

func (u *User) GenerateMultipleItemInstancesFromCfgOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.Readonly_DItemOffset, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemOffset))
	for _, offset := range itemOffset {
		itemInst, result := u.GenerateItemInstanceFromCfgOffset(ctx, offset)
		if result.IsError() {
			if ignoreError {
				ctx.LogError("generate item instance from item offset failed",
					"error", result.Error, "resoponse_code", result.ResponseCode,

					"item_type_id", offset.GetTypeId(), "item_count", offset.GetCount(),
				)
				continue
			}
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) GenerateMultipleItemInstancesFromCfgOffsetRatio(ctx cd.RpcContext, itemOffset []*public_protocol_common.Readonly_DItemOffset, ignoreError bool, ratio int64) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemOffset))
	for _, offset := range itemOffset {
		if ratio == 1 {
			itemInst, result := u.GenerateItemInstanceFromCfgOffset(ctx, offset)
			if result.IsError() {
				if ignoreError {
					ctx.LogError("generate item instance from item offset failed",
						"error", result.Error, "resoponse_code", result.ResponseCode,

						"item_type_id", offset.GetTypeId(), "item_count", offset.GetCount(),
					)
					continue
				}
				return nil, result
			}
			ret = append(ret, itemInst)
		} else {
			scaledOffset := offset.ToMessage()
			scaledOffset.Count = scaledOffset.Count * ratio
			itemInst, result := u.GenerateItemInstanceFromOffset(ctx, scaledOffset)
			if result.IsError() {
				if ignoreError {
					ctx.LogError("generate item instance from item offset failed",
						"error", result.Error, "resoponse_code", result.ResponseCode,

						"item_type_id", offset.GetTypeId(), "item_count", scaledOffset.GetCount(),
					)
					continue
				}
				return nil, result
			}
			ret = append(ret, itemInst)
		}
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) GenerateMultipleItemInstancesFromOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemOffset, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemOffset))
	for _, offset := range itemOffset {
		itemInst, result := u.GenerateItemInstanceFromOffset(ctx, offset)
		if result.IsError() {
			if ignoreError {
				ctx.LogError("generate item instance from item offset failed",
					"error", result.Error, "resoponse_code", result.ResponseCode,

					"item_type_id", offset.GetTypeId(), "item_count", offset.GetCount(),
				)
				continue
			}
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) GenerateItemInstanceFromBasic(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemBasic.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromBasic(ctx, itemBasic)
}

func (u *User) GenerateMultipleItemInstancesFromBasic(ctx cd.RpcContext, itemBasic []*public_protocol_common.DItemBasic, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemBasic))
	for _, basic := range itemBasic {
		itemInst, result := u.GenerateItemInstanceFromBasic(ctx, basic)
		if result.IsError() {
			ctx.LogError("generate item instance from item basic failed",
				"error", result.Error, "resoponse_code", result.ResponseCode,

				"item_type_id", basic.GetTypeId(), "item_count", basic.GetCount(),
			)
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) unpackMergeItemOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]*public_protocol_common.DItemInstance, Result) {
	if len(itemOffset) == 0 {
		return nil, cd.CreateRpcResultOk()
	}

	mergeItemInstan := make(map[int32]map[int64]*public_protocol_common.DItemInstance)
	itemOffsetSize := 0
	for _, offset := range itemOffset {
		// 解包合并ItemOffset
		typeId := offset.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}
		items, result := mgr.UnpackItem(ctx, offset)
		if result.IsError() {
			return nil, result
		}

		for _, item := range items {
			if _, exists := mergeItemInstan[item.GetItemBasic().GetTypeId()]; !exists {
				mergeItemInstan[item.GetItemBasic().GetTypeId()] = make(map[int64]*public_protocol_common.DItemInstance)
			}
			v := mergeItemInstan[item.GetItemBasic().GetTypeId()]

			existItem, exists := v[item.GetItemBasic().GetGuid()]
			if exists {
				existItem.GetItemBasic().Count += item.GetItemBasic().GetCount()
			} else {
				v[item.GetItemBasic().GetGuid()] = item.Clone()
				itemOffsetSize++
			}
		}
	}

	// 输出
	ret := make([]*public_protocol_common.DItemInstance, 0, itemOffsetSize)
	for _, guidMap := range mergeItemInstan {
		for _, item := range guidMap {
			ret = append(ret, item)
		}
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) CheckAddItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]*ItemAddGuard, Result) {
	unpackMergeItemOffset, result := u.unpackMergeItemOffset(ctx, itemOffset)
	if result.IsError() {
		return nil, result
	}

	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*public_protocol_common.DItemInstance
	})
	for _, offset := range unpackMergeItemOffset {
		typeId := offset.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*public_protocol_common.DItemInstance
			}{
				data: make([]*public_protocol_common.DItemInstance, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	ret := make([]*ItemAddGuard, 0, len(unpackMergeItemOffset))
	for mgr, group := range splitByMgr {
		subRet, subResult := mgr.CheckAddItem(ctx, group.data)
		if subResult.IsError() {
			return nil, subResult
		}
		ret = append(ret, subRet...)
	}
	return ret, cd.CreateRpcResultOk()
}

func (u *User) CheckSubItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemBasic) ([]*ItemSubGuard, Result) {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*public_protocol_common.DItemBasic
	})
	for _, offset := range itemOffset {
		typeId := offset.GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user sub item failed, item manager not found", "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*public_protocol_common.DItemBasic
			}{
				data: make([]*public_protocol_common.DItemBasic, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	ret := make([]*ItemSubGuard, 0, len(itemOffset))
	for mgr, group := range splitByMgr {
		subRet, subResult := mgr.CheckSubItem(ctx, group.data)
		if subResult.IsError() {
			return nil, subResult
		}
		ret = append(ret, subRet...)
	}
	return ret, cd.CreateRpcResultOk()
}

func (u *User) GetItemTypeStatistics(ctx cd.RpcContext, typeId int32) *ItemTypeStatistics {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil
	}

	return mgr.GetTypeStatistics(ctx, typeId)
}

func (u *User) GetItemFromBasic(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, Result) {
	if itemBasic == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	typeId := itemBasic.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GetItemFromBasic(ctx, itemBasic)
}

func (u *User) GetNotEnoughErrorCode(typeId int32) int32 {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GetNotEnoughErrorCode(typeId)
}

func (u *User) CheckTypeIdValid(typeId int32) bool {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return false
	}

	return mgr.CheckTypeIdValid(typeId)
}

// 检查期望消耗是否满足配置要求.
func (u *User) CheckCostItemCfg(ctx cd.RpcContext,
	realCost []*public_protocol_common.DItemBasic,
	expectCost []*public_protocol_common.Readonly_DItemOffset,
) Result {
	return u.CheckCostRatioItemCfg(ctx, realCost, expectCost, 1)
}

func (u *User) CheckCostRatioItemCfg(ctx cd.RpcContext,
	realCost []*public_protocol_common.DItemBasic,
	expectCost []*public_protocol_common.Readonly_DItemOffset,
	ratio int64) Result {
	if len(expectCost) == 0 {
		return cd.CreateRpcResultOk()
	}

	if ratio <= 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	countByTypeId := make(map[int32]int64)
	for _, cost := range realCost {
		typeId := cost.GetTypeId()
		if typeId == 0 || cost.GetCount() <= 0 {
			continue
		}

		countByTypeId[typeId] += cost.GetCount()
	}

	for _, expect := range expectCost {
		typeId := expect.GetTypeId()
		expectCount := expect.GetCount() * ratio
		actualCount, exists := countByTypeId[typeId]
		if !exists || actualCount < expectCount {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(u.GetNotEnoughErrorCode(typeId)))
		}
	}

	return cd.CreateRpcResultOk()
}

func (u *User) CheckCostItem(ctx cd.RpcContext,
	realCost []*public_protocol_common.DItemBasic,
	expectCost []*public_protocol_common.DItemOffset,
) Result {
	if len(expectCost) == 0 {
		return cd.CreateRpcResultOk()
	}

	countByTypeId := make(map[int32]int64)
	for _, cost := range realCost {
		typeId := cost.GetTypeId()
		if typeId == 0 || cost.GetCount() <= 0 {
			continue
		}

		countByTypeId[typeId] += cost.GetCount()
	}

	for _, expect := range expectCost {
		typeId := expect.GetTypeId()
		expectCount := expect.GetCount()
		actualCount, exists := countByTypeId[typeId]
		if !exists || actualCount < expectCount {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(u.GetNotEnoughErrorCode(typeId)))
		}
	}

	return cd.CreateRpcResultOk()
}

// 检查期望消耗是否满足配置要求.
func (u *User) MergeCostItem(expectCost ...[]*public_protocol_common.Readonly_DItemOffset) []*public_protocol_common.Readonly_DItemOffset {
	if len(expectCost) == 0 {
		return nil
	}

	if len(expectCost) == 1 {
		return expectCost[0]
	}

	countByTypeId := make(map[int32]int64)
	for _, costList := range expectCost {
		for _, cost := range costList {
			typeId := cost.GetTypeId()
			if countByTypeId[typeId] <= 0 {
				countByTypeId[typeId] = 0
			}

			countByTypeId[typeId] += cost.GetCount()
		}
	}

	ret := make([]*public_protocol_common.Readonly_DItemOffset, 0, len(countByTypeId))
	for typeId, count := range countByTypeId {
		o := &public_protocol_common.DItemOffset{
			TypeId: typeId,
			Count:  count,
		}
		ret = append(ret, o.ToReadonly())
	}

	return ret
}

type CheckUseItemHandle = func(ctx cd.RpcContext, user *User, useAction *public_protocol_common.Readonly_DItemUseAction, itemBasic *public_protocol_common.DItemBasic, useParam *public_protocol_common.DItemUseParam) Result
type UseItemHandle = func(ctx cd.RpcContext, user *User, useAction *public_protocol_common.Readonly_DItemUseAction, itemBasic *public_protocol_common.DItemBasic, useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result)

type UseItemHandlers struct {
	CheckUseItemHandler CheckUseItemHandle
	UseItemHandler      UseItemHandle
}

var useItemHandlerRegistry = make(map[public_protocol_common.DItemUseAction_EnActionTypeID]*UseItemHandlers)

func RegisterUseItemHandler(useAction public_protocol_common.DItemUseAction_EnActionTypeID,
	checkUseItemHandler CheckUseItemHandle,
	useItemHandler UseItemHandle,
) {
	if useAction <= 0 {
		return
	}

	if checkUseItemHandler == nil && useItemHandler == nil {
		return
	}

	handlers := &UseItemHandlers{
		CheckUseItemHandler: checkUseItemHandler,
		UseItemHandler:      useItemHandler,
	}

	useItemHandlerRegistry[useAction] = handlers
}

func init() {
	RegisterUseItemHandler(
		public_protocol_common.DItemUseAction_EnActionTypeID_RandomPool,
		func(ctx cd.RpcContext, user *User, useAction *public_protocol_common.Readonly_DItemUseAction,
			itemBasic *public_protocol_common.DItemBasic, useParam *public_protocol_common.DItemUseParam) Result {
			ret, _ := config.RandomWithPool(useAction.GetRandomPool(), itemBasic.GetCount(), useParam.GetRandomPoolIndex())
			if ret != 0 {
				return cd.CreateRpcResultError(nil, ret)
			}
			return cd.CreateRpcResultOk()
		},
		func(ctx cd.RpcContext, user *User, useAction *public_protocol_common.Readonly_DItemUseAction, itemBasic *public_protocol_common.DItemBasic,
			useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result) {
			poolId := useAction.GetRandomPool()
			poolCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelRandomPoolByPoolId(poolId)
			if poolCfg == nil {
				return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_FOUND)
			}

			ret, items := config.RandomWithPool(poolId, itemBasic.GetCount(), useParam.GetRandomPoolIndex())
			if ret != 0 {
				return nil, cd.CreateRpcResultError(nil, ret)
			}

			itemInsts, result := user.GenerateMultipleItemInstancesFromOffset(ctx, items, true)
			if result.IsError() {
				return nil, result
			}
			guard, result := user.CheckAddItem(ctx, itemInsts)
			if result.IsError() {
				return nil, result
			}
			user.AddItem(ctx, guard, reason)
			gainItem := []*public_protocol_common.DItemInstance{}
			for _, item := range guard {
				gainItem = append(gainItem, item.GetAddedItems()...)
			}
			return gainItem, cd.CreateRpcResultOk()
		})
}
