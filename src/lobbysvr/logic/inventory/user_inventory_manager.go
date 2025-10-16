package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory/internal"
)

type UserInventoryManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}

func init() {
	data.RegisterUserModuleManagerCreator[impl.UserInventoryManager](func(owner *data.User) data.UserModuleManagerImpl {
		return impl.CreateUserInventoryManager(owner)
	})
}
