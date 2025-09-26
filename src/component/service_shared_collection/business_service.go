package atframework_component_service_shared_collection

import (
	config "github.com/atframework/atsf4g-go/component-config"
	libatapp "github.com/atframework/libatapp-go"
)

func CreateServiceApplication() libatapp.AppImpl {
	app := libatapp.CreateAppInstance()
	// TODO: 内置公共逻辑层模块
	configManager := config.GetConfigManager()
	configManager.AppModuleBase = libatapp.CreateAppModuleBase(app)
	app.AddModule(configManager)
	return app
}
