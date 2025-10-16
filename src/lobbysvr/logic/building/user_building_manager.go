package lobbysvr_logic_building

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/building/internal"
)

type UserBuildingManager interface {
	data.UserModuleManagerImpl

	BuildingPlace() int32
	BuildingStore() int32
	BuildingMove() int32
}

func init() {
	data.RegisterUserModuleManagerCreator[impl.UserBuildingManager](func(owner *data.User) data.UserModuleManagerImpl {
		return impl.CreateUserBuildingManager(owner)
	})
}
