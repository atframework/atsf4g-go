package main

import (
	"os"

	ssc "github.com/atframework/atsf4g-go/component-service_shared_collection"
)

func main() {
	app := ssc.CreateServiceApplication()
	err := app.Run(os.Args[1:])
	if err != nil {
		println("%s", err.Error())
	}
}
