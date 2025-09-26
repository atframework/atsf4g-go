package main

import (
	"os"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"

	uc "github.com/atframework/atsf4g-go/component-user_controller"

	lobbysvr_app "github.com/atframework/atsf4g-go/service-lobbysvr/app"
)

func main() {
	app := ssc.CreateServiceApplication()

	// CS消息WebSocket分发器
	csDispatcher := uc.WebsocketDispatcherCreateCSMessage(app)
	app.AddModule(csDispatcher)

	lobbysvr_app.RegisterLobbyClientService(csDispatcher, uc.WebsocketDispatcherFindSessionFromMessage)

	err := app.Run(os.Args[1:])
	if err != nil {
		println("%s", err.Error())
	}
}
