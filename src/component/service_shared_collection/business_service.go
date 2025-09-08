package atframework_component_service_shared_collection

import (
	libatapp "github.com/atframework/libatapp-go"
)

func CreateServiceApplication() libatapp.AppImpl {
	app := libatapp.CreateAppInstance()
	// TODO: 内置公共逻辑层模块
	return app
}
