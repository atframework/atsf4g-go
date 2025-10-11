package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

type UserInventoryManager struct {
	owner *data.User

	data.UserModuleManagerBase
	data.UserItemManagerBase
}

func CreateUserInventoryManager(owner *data.User) impl.UserInventoryManager {
	return &UserInventoryManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
	}
}

func init() {
	data.RegisterUserModuleManagerCreator[impl.UserInventoryManager](func(owner *data.User) data.UserModuleManagerImpl {
		return CreateUserInventoryManager(owner)
	})

	// var owner *data.User
	// mgr := data.GetModuleManager[impl.UserInventoryManager](owner)
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
