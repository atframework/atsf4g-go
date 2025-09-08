package atframework_component_service_shared_collection

import (
	libapp "github.com/atframework/libatapp-go"
)

func CreateServiceApplication() libapp.AppImpl {
	app := libapp.CreateAppInstance()
	// TODO: 内置公共逻辑层模块
	return app
}
