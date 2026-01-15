package main

import (
	"os"

	"github.com/atframework/libatapp-go"
	atapp "github.com/atframework/libatapp-go"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"

	log "github.com/atframework/atframe-utils-go/log"
	config "github.com/atframework/atsf4g-go/component-config"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_config "github.com/atframework/atsf4g-go/component-protocol-private/config/protocol/config"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	uc_d "github.com/atframework/atsf4g-go/component-user_controller/dispatcher"
	lobbysvr_app "github.com/atframework/atsf4g-go/service-lobbysvr/app"
)

func main() {
	// 处理 --info 标志（在应用初始化前）
	atapp.RegisterBuildInfoCommand()

	app := ssc.CreateServiceApplication()

	uc.InitUserRouterManager(app)

	config.GetConfigManager().SetServerConfigureLoadFunc(func(originConfigData interface{}, callback generate_config.ConfigCallback) (interface{}, error) {
		serverConfig := &private_protocol_config.LobbyServerCfg{}
		err := libatapp.LoadConfigFromOriginDataByPath(callback.GetLogger(),
			originConfigData, serverConfig, "lobbysvr", "", nil, nil, "")
		if err != nil {
			callback.GetLogger().LogError("Load config failed", "error", err)
			return nil, err
		}
		return serverConfig.ToReadonly(), nil
	})

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
	log.CloseAllLogWriters()
}
