package lobbysvr_logic_item

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user/internal"
)

type UserBasicManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}

func init() {
	data.RegisterUserModuleManagerCreator[UserBasicManager](func(_ctx *cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return impl.CreateUserBasicManager(owner)
	})
}
