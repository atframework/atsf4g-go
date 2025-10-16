package lobbysvr_logic_item

import (
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
)

type UserBuildingManager struct {
	owner *data.User

	data.UserModuleManagerBase
}

func CreateUserBuildingManager(owner *data.User) *UserBuildingManager {
	return &UserBuildingManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}
}

func (m *UserBuildingManager) InitFromDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserBuildingManager) DumpToDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserBuildingManager) BuildingPlace() int32 { return 0 }

func (m *UserBuildingManager) BuildingStore() int32 { return 0 }

func (m *UserBuildingManager) BuildingMove() int32 { return 0 }
