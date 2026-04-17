package lobbysvr_logic_open_platform

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	component_open_platform "github.com/atframework/atsf4g-go/component/open_platform"
)

type UserOpenPlatformManager interface {
	data.UserModuleManagerImpl

	AwaitIoTask(ctx cd.AwaitableContext) cd.RpcResult

	UpdateAccessToken(ctx cd.RpcContext, accessToken string, authData *public_protocol_pbdesc.DClientAuthAccessData)

	UpdateAuthData(ctx cd.RpcContext, channelDelegate component_open_platform.OpenPlatformChannelDelegate, authData component_open_platform.UserAuthData)
}
