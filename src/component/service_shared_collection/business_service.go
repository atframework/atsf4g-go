package atframework_component_service_shared_collection

import (
	config "github.com/atframework/atsf4g-go/component-config"
	libatapp "github.com/atframework/libatapp-go"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	router "github.com/atframework/atsf4g-go/component-router"
)

func CreateServiceApplication() libatapp.AppImpl {
	app := libatapp.CreateAppInstance()

	// 内置公共逻辑层模块
	libatapp.AtappAddModule(app, config.CreateConfigManagerModule(app))
	libatapp.AtappAddModule(app, cd.CreateNoMessageDispatcher(app))
	libatapp.AtappAddModule(app, cd.CreateTaskManager(app))
	libatapp.AtappAddModule(app, router.CreateRouterManagerSet(app))
	return app
}
