package lobbysvr_logic_building

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserBuildingManager interface {
	data.UserModuleManagerImpl

	BuildingPlace() int32
	BuildingStore() int32
	BuildingMove() int32
}
