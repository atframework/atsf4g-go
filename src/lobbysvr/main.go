package main

import (
	"os"

	atapp "github.com/atframework/libatapp-go"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"

	uc_d "github.com/atframework/atsf4g-go/component-user_controller/dispatcher"

	lobbysvr_app "github.com/atframework/atsf4g-go/service-lobbysvr/app"
)

func main() {
	app := ssc.CreateServiceApplication()

	// CS消息WebSocket分发器
	csDispatcher := uc_d.WebsocketDispatcherCreateCSMessage(app, "lobbysvr.webserver", "lobbysvr.websocket")
	atapp.AtappAddModule(app, csDispatcher)

	if err := lobbysvr_app.RegisterLobbyClientService(csDispatcher, uc_d.WebsocketDispatcherFindSessionFromMessage); err != nil {
		println("RegisterLobbyClientService fail: %s", err.Error())
		return
	}

	err := app.Run(os.Args[1:])
	if err != nil {
		println("%s", err.Error())
	}
}
