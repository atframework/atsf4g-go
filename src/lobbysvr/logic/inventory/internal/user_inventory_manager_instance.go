package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory"
)

type UserInventoryManager struct {
	data.UserModuleManagerBase
}

func CreateUserInventoryManager(owner *data.User) *UserInventoryManager {
	return &UserInventoryManager{
		UserModuleManagerBase: data.CreateUserModuleManagerBase(owner),
	}
}

func init() {
	data.RegisterUserModuleManagerCreator[impl.UserInventoryManager](func(owner *data.User) data.UserModuleManagerImpl {
		return CreateUserInventoryManager(owner)
	})

	// var owner *data.User
	// mgr := data.GetModuleManager[impl.UserInventoryManager](owner)
}
