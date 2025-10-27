package lobbysvr_logic_user

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserBasicManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}
