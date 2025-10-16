package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

type UserInventoryManager struct {
	owner *data.User

	data.UserModuleManagerBase
	data.UserItemManagerBase
}

func CreateUserInventoryManager(owner *data.User) *UserInventoryManager {
	return &UserInventoryManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
	}
}

func (m *UserInventoryManager) GetOwner() *data.User { return m.owner }

func (m *UserInventoryManager) AddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance, reason *data.ItemFlowReason) data.Result {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) SubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic, reason *data.ItemFlowReason) data.Result {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) CheckAddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance) data.Result {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) CheckSubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic) data.Result {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) GetItemFromBasic(itemBasic *ppc.DItemBasic) (*ppc.DItemInstance, data.Result) {
	return nil, cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}
