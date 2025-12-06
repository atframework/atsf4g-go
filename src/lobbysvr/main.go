package main

import (
	"os"

	atapp "github.com/atframework/libatapp-go"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	uc_d "github.com/atframework/atsf4g-go/component-user_controller/dispatcher"

	lobbysvr_app "github.com/atframework/atsf4g-go/service-lobbysvr/app"
)

func main() {
	// 处理 --info 标志（在应用初始化前）
	atapp.RegisterBuildInfoCommand()

	app := ssc.CreateServiceApplication()

	uc.InitUserRouterManager(app)

	sessionManager := uc.CreateSessionManager(app)
	atapp.AtappAddModule(app, sessionManager)

	userManager := uc.CreateUserManager(app)
	atapp.AtappAddModule(app, userManager)

	// CS消息WebSocket分发器
	csDispatcher := uc_d.WebsocketDispatcherCreateCSMessage(app, "lobbysvr.webserver", "lobbysvr.websocket")
	atapp.AtappAddModule(app, csDispatcher)

	redisDispatcher := cd.CreateRedisMessageDispatcher(app)
	atapp.AtappAddModule(app, redisDispatcher)

	if err := lobbysvr_app.RegisterLobbyClientService(csDispatcher, uc_d.WebsocketDispatcherFindSessionFromMessage); err != nil {
		println("RegisterLobbyClientService fail: %s", err.Error())
		return
	}

	err := app.Run(os.Args[1:])
	if err != nil {
		println("%s", err.Error())
	}
}
