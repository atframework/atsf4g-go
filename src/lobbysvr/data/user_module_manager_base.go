package lobbysvr_data

import (
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type UserModuleManagerImpl interface {
	GetOwner() *User

	RefreshLimit(*cd.RpcContext)
	RefreshLimitSecond(*cd.RpcContext)
	RefreshLimitMinute(*cd.RpcContext)

	InitFromDB(*private_protocol_pbdesc.DatabaseTableUser) error
	DumpToDB(*private_protocol_pbdesc.DatabaseTableUser) error

	CreateInit(ctx *cd.RpcContext, versionType uint32)
	LoginInit(*cd.RpcContext)

	OnLogin(*cd.RpcContext)
	OnLogout(*cd.RpcContext)
	OnSaved(*cd.RpcContext)
	OnUpdateSession(ctx *cd.RpcContext, from *uc.Session, to *uc.Session)

	SyncDirtyCache()
	CleanupDirtyCache()
}

type UserModuleManagerBase struct {
	owner *User
}
