package atframework_component_service_shared_collection

import (
	config "github.com/atframework/atsf4g-go/component-config"
	libatapp "github.com/atframework/libatapp-go"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

func CreateServiceApplication() libatapp.AppImpl {
	app := libatapp.CreateAppInstance()

	// 内置公共逻辑层模块
	libatapp.AtappAddModule(app, config.CreateConfigManagerModule(app))

	libatapp.AtappAddModule(app, cd.CreateNoMessageDispatcher(app))
	return app
}
