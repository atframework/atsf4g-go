package lobbysvr_logic_inventory

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserInventoryManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}
