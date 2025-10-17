package lobbysvr_logic_item

import (
	"fmt"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	pp_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type UserVirtualItemManager struct {
	owner *UserInventoryManager
}

func createVirtualItemManager(owner *UserInventoryManager) *UserVirtualItemManager {
	return &UserVirtualItemManager{
		owner: owner,
	}
}

func (m *UserVirtualItemManager) RefreshLimitSecond(_ctx *cd.RpcContext) {
}

func (m *UserVirtualItemManager) AddItem(ctx *cd.RpcContext, itemOffset *data.ItemAddGuard, reason *data.ItemFlowReason) (bool, data.Result) {
	return false, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) SubItem(ctx *cd.RpcContext, itemOffset *data.ItemSubGuard, reason *data.ItemFlowReason) (bool, data.Result) {
	return false, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) CheckAddItem(ctx *cd.RpcContext, itemOffset *ppc.DItemInstance) data.Result {
	if itemOffset == nil {
		return cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	switch itemOffset.GetItemBasic().GetTypeId() {
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_COIN):
	case int32(ppc.EnItemMoneyType_EN_ITEM_MONEY_TYPE_CASH):
		return cd.CreateRpcResultOk()
	default:
		return cd.CreateRpcResultError(nil, pp_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) CheckSubItem(_ctx *cd.RpcContext, itemOffset *ppc.DItemBasic) data.Result {
	if itemOffset == nil {
		return cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), pp_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) GetTypeStatistics(_typeId int32) (bool, *data.ItemTypeStatistics) {
	return false, nil
}

func (m *UserVirtualItemManager) GetNotEnoughErrorCode(typeId int32) int32 {
	return m.owner.UserItemManagerBase.GetNotEnoughErrorCode(typeId)
}

func (m *UserVirtualItemManager) GetItemFromBasic(_itemBasic *ppc.DItemBasic) (bool, *ppc.DItemInstance, data.Result) {
	return false, nil, cd.CreateRpcResultOk()
}

func (m *UserVirtualItemManager) ForeachItem(_fn func(item *ppc.DItemInstance) bool) {
}
