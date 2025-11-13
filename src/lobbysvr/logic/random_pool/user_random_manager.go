package lobbysvr_logic_random_pool

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserRandomPoolManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}
