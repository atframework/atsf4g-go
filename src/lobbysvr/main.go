package main

import (
	"os"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"

	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

func main() {
	app := ssc.CreateServiceApplication()

	// CS消息WebSocket分发器
	app.AddModule(uc.WebsocketDispatcherCreateCSMessage(app))

	err := app.Run(os.Args[1:])
	if err != nil {
		println("%s", err.Error())
	}
}
