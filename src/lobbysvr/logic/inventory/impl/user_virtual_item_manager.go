package lobbysvr_logic_inventory_impl

import (
	"fmt"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	pp_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

type UserVirtualItemManager struct {
	owner *UserInventoryManager

	cachedVirtualItemInstance   map[int32]*ppc.DItemInstance
	cachedVirtualItemStatistics map[int32]*data.ItemTypeStatistics
}

func createVirtualItemManager(owner *UserInventoryManager) *UserVirtualItemManager {
	return &UserVirtualItemManager{
		owner:                       owner,
		cachedVirtualItemInstance:   make(map[int32]*ppc.DItemInstance),
		cachedVirtualItemStatistics: make(map[int32]*data.ItemTypeStatistics),
	}
}

func (m *UserVirtualItemManager) mutableVirtualItemInstance(typeID int32) *ppc.DItemInstance {
	if m == nil {
		return &ppc.DItemInstance{
			ItemBasic: &ppc.DItemBasic{
				TypeId: typeID,
				Count:  0,
				Guid:   0,
			},
		}
	}

	ret, ok := m.cachedVirtualItemInstance[typeID]
	if !ok || ret == nil {
		ret = &ppc.DItemInstance{
			ItemBasic: &ppc.DItemBasic{
				TypeId: typeID,
				Count:  0,
				Guid:   0,
			},
		}
		m.cachedVirtualItemInstance[typeID] = ret
	}

	return ret
}

func (m *UserVirtualItemManager) mutableVirtualItemStatistics(typeID int32) *data.ItemTypeStatistics {
	if m == nil {
		return &data.ItemTypeStatistics{
			TotalCount: 0,
		}
	}

	ret, ok := m.cachedVirtualItemStatistics[typeID]
	if !ok || ret == nil {
		ret = &data.ItemTypeStatistics{
			TotalCount: 0,
		}
		m.cachedVirtualItemStatistics[typeID] = ret
	}

	return ret
}

func (m *UserVirtualItemManager) RefreshLimitSecond(_ctx cd.RpcContext) {
}

func (m *UserVirtualItemManager) InitFromDB(_ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser, _itemOffset *ppc.DItemInstance) (bool, cd.RpcResult) {
	return false, cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserVirtualItemManager) DumpToDB(_ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) (bool, cd.RpcResult) {
	return false, cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserVirtualItemManager) AddItem(ctx cd.RpcContext, itemOffset *data.ItemAddGuard, reason *data.ItemFlowReason) (process bool, result data.Result) {
	if itemOffset == nil || itemOffset.Item == nil {
		return true, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	itemTypeId := itemOffset.Item.GetItemBasic().GetTypeId()
	if itemTypeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_READ_ONLY_BEGIN) {
		return true, cd.CreateRpcResultError(fmt.Errorf("item %d is readonly", itemTypeId), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	questMgr := data.UserGetModuleManager[logic_quest.UserQuestManager](m.owner.GetOwner())
	if questMgr == nil {
		ctx.LogError("can not find user quest manager")
	}

	switch itemTypeId {
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return true, redirMgr.AddUserExp(ctx, itemOffset.Item.GetItemBasic().GetCount())
	default:
		break
	}

	return false, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) SubItem(ctx cd.RpcContext, itemOffset *data.ItemSubGuard, reason *data.ItemFlowReason) (process bool, result data.Result) {
	if itemOffset == nil || itemOffset.Item == nil {
		return true, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	itemTypeId := itemOffset.Item.GetTypeId()
	if itemTypeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_READ_ONLY_BEGIN) {
		return true, cd.CreateRpcResultError(fmt.Errorf("item %d is readonly", itemTypeId), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	switch itemTypeId {
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return true, redirMgr.SubUserExp(ctx, itemOffset.Item.GetCount())
	default:
		break
	}

	return false, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) CheckAddItem(ctx cd.RpcContext, itemOffset *ppc.DItemInstance) data.Result {
	if itemOffset == nil {
		return cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if itemOffset.GetItemBasic().GetCount() < 0 {
		return cd.CreateRpcResultError(nil, pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if itemOffset.GetItemBasic().GetGuid() != 0 {
		return cd.CreateRpcResultError(fmt.Errorf("virtual item can not have guid"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	itemTypeId := itemOffset.GetItemBasic().GetTypeId()
	if itemTypeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_READ_ONLY_BEGIN) {
		return cd.CreateRpcResultError(fmt.Errorf("item %d is readonly", itemTypeId), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY)
	}

	switch itemTypeId {
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_COIN):
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_CASH):
		return cd.CreateRpcResultOk()
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return redirMgr.CheckAddUserExp(ctx, itemOffset.GetItemBasic().GetCount())
	default:
		return cd.CreateRpcResultError(nil, pp_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) CheckSubItem(ctx cd.RpcContext, itemOffset *ppc.DItemBasic) (process bool, result data.Result) {
	if itemOffset == nil {
		return true, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if itemOffset.GetCount() < 0 {
		return true, cd.CreateRpcResultError(nil, pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if itemOffset.GetGuid() != 0 {
		return true, cd.CreateRpcResultError(fmt.Errorf("virtual item can not have guid"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	itemTypeId := itemOffset.GetTypeId()
	if itemTypeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_READ_ONLY_BEGIN) {
		return true, cd.CreateRpcResultError(fmt.Errorf("item %d is readonly", itemTypeId), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY)
	}

	switch itemTypeId {
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return true, redirMgr.CheckSubUserExp(ctx, itemOffset.GetCount())
	default:
		break
	}

	return false, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) GetTypeStatistics(ctx cd.RpcContext, typeId int32) (process bool, result *data.ItemTypeStatistics) {
	switch typeId {
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, nil
		}

		ret := m.mutableVirtualItemStatistics(typeId)
		ret.TotalCount = redirMgr.GetUserExp()
		return true, ret
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_READONLY_USER_LEVEL):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, nil
		}

		ret := m.mutableVirtualItemStatistics(typeId)
		ret.TotalCount = int64(redirMgr.GetUserLevel())
		return true, ret
	default:
		break
	}
	return false, nil
}

func (m *UserVirtualItemManager) GetNotEnoughErrorCode(typeId int32) int32 {
	switch typeId {
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_COIN):
		return int32(pp_pbdesc.EnErrorCode_EN_ERR_MONEY_COIN_NOT_ENOUGH)
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_CASH):
		return int32(pp_pbdesc.EnErrorCode_EN_ERR_MONEY_CASH_NOT_ENOUGH)
	// case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
	// 	return int32(pp_pbdesc.EnErrorCode_EN_ERR_USER_MIN_LEVEL_LIMIT)
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_READONLY_USER_LEVEL):
		return int32(pp_pbdesc.EnErrorCode_EN_ERR_USER_MIN_LEVEL_LIMIT)
	default:
		break
	}
	return m.owner.UserItemManagerBase.GetNotEnoughErrorCode(typeId)
}

func (m *UserVirtualItemManager) GetItemFromBasic(ctx cd.RpcContext, itemBasic *ppc.DItemBasic) (process bool, instance *ppc.DItemInstance, reulst data.Result) {
	if itemBasic == nil {
		return true, nil, cd.CreateRpcResultError(fmt.Errorf("itemBasic is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	switch itemBasic.GetTypeId() {
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, nil, cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}

		ret := m.mutableVirtualItemInstance(itemBasic.GetTypeId())
		ret.MutableItemBasic().Count = redirMgr.GetUserExp()
		ret.MutableItemBasic().Guid = 0
		return true, ret, cd.CreateRpcResultOk()
	case int32(ppc.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_READONLY_USER_LEVEL):
		redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
		if redirMgr == nil {
			return true, nil, cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"), pp_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}

		ret := m.mutableVirtualItemInstance(itemBasic.GetTypeId())
		ret.MutableItemBasic().Count = int64(redirMgr.GetUserLevel())
		ret.MutableItemBasic().Guid = 0
		return true, ret, cd.CreateRpcResultOk()
	default:
		break
	}
	if itemBasic.GetCount() < 0 {
		return true, nil, cd.CreateRpcResultError(nil, pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return false, nil, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) ForeachItem(fn func(item *ppc.DItemInstance) bool) bool {
	redirMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.owner.GetOwner())
	if redirMgr != nil {
		if !redirMgr.ForeachItem(fn) {
			return false
		}
	}

	return true
}
