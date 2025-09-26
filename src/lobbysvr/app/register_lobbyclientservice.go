package lobbysvr_app

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"

	sp "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"

	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func RegisterLobbyClientService(
	rd cd.DispatcherImpl, findSessionFn uc.FindCSMessageSession,
) error {
	svc := sp.File_protocol_pbdesc_lobby_client_service_proto.Services().ByName("proy.LobbyClientService")
	if svc == nil {
		rd.GetApp().GetLogger().Error("lobbysvr_app.RegisterLobbyClientService no service proy.LobbyClientService")
		return fmt.Errorf("no service proy.LobbyClientService")
	}

	uc.RegisterCSMessageAction(
		rd, findSessionFn, svc, "proy.LobbyClientService.login_auth",
		func(base cd.TaskActionCSBase[*sp.CSLoginAuthReq, *sp.SCLoginAuthRsp]) cd.TaskActionImpl {
			return &logic_user.TaskActionLoginAuth{TaskActionCSBase: base}
		},
	)

	return nil
}
