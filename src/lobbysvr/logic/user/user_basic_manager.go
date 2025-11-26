package lobbysvr_logic_user

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserBasicManager interface {
	data.UserModuleManagerImpl

	DumpUserInfo() *public_protocol_pbdesc.DUserInfo
	DumpUserOptions() *public_protocol_pbdesc.DUserOptions

	ForeachItem(_fn func(item *public_protocol_common.DItemInstance) bool) bool

	CheckAddUserExp(ctx cd.RpcContext, v int64) data.Result
	CheckSubUserExp(ctx cd.RpcContext, v int64) data.Result
	AddUserExp(ctx cd.RpcContext, v int64) data.Result
	SubUserExp(ctx cd.RpcContext, v int64) data.Result
	GetUserExp() int64
	GetUserLevel() uint32
}
