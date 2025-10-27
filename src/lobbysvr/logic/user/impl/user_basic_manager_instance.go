package lobbysvr_logic_user_impl

import (
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func init() {
	data.RegisterUserModuleManagerCreator[logic_user.UserBasicManager](func(_ctx *cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserBasicManager(owner)
	})
}

type UserBasicManager struct {
	data.UserModuleManagerBase
}

func CreateUserBasicManager(owner *data.User) *UserBasicManager {
	ret := &UserBasicManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserBasicManager) InitFromDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) DumpToDB(_ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) RefreshLimitSecond(_ctx *cd.RpcContext) {
}
