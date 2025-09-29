// Copyright 2025 atframework
// @brief Created by mako-generator.py for proy.LobbyClientService, please don't edit it

package lobbysvr_app

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"

	sp "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"

	logic_action "github.com/atframework/atsf4g-go/service-lobbysvr/logic/action"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func RegisterLobbyClientService(
	rd cd.DispatcherImpl, findSessionFn uc.FindCSMessageSession,
) error {
	svc := sp.File_protocol_pbdesc_lobby_client_service_proto.Services().ByName("LobbyClientService")
	if svc == nil {
		rd.GetApp().GetLogger().Error("lobbysvr_app.RegisterLobbyClientService no service proy.LobbyClientService")
		return fmt.Errorf("no service proy.LobbyClientService")
	}

	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.login_auth",
		func(base cd.TaskActionCSBase[*sp.CSLoginAuthReq, *sp.SCLoginAuthRsp]) cd.TaskActionImpl {
			return &logic_action.TaskActionLoginAuth{TaskActionCSBase: base}
		},
	)
	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.login",
		func(base cd.TaskActionCSBase[*sp.CSLoginReq, *sp.SCLoginRsp]) cd.TaskActionImpl {
			return &logic_action.TaskActionLogin{TaskActionCSBase: base}
		},
	)
	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.ping",
		func(base cd.TaskActionCSBase[*sp.CSPingReq, *sp.SCPongRsp]) cd.TaskActionImpl {
			return &logic_action.TaskActionPing{TaskActionCSBase: base}
		},
	)
	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.access_update",
		func(base cd.TaskActionCSBase[*sp.CSAccessUpdateReq, *sp.SCAccessUpdateRsp]) cd.TaskActionImpl {
			return &logic_action.TaskActionAccessUpdate{TaskActionCSBase: base}
		},
	)
	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.user_get_info",
		func(base cd.TaskActionCSBase[*sp.CSUserGetInfoReq, *sp.SCUserGetInfoRsp]) cd.TaskActionImpl {
			return &logic_user.TaskActionUserGetInfo{TaskActionCSBase: base}
		},
	)

	return nil
}
