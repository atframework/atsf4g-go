package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserItemManager struct {
	data.UserModuleManagerBase
}

func CreateUserItemManager(owner *data.User) *UserItemManager {
	return &UserItemManager{
		UserModuleManagerBase: data.CreateUserModuleManagerBase(owner),
	}
}

func init() {
	data.RegisterUserModuleManagerCreator[UserItemManager](func(owner *data.User) data.UserModuleManagerImpl {
		return CreateUserItemManager(owner)
	})
}
