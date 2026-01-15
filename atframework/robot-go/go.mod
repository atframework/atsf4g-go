module github.com/atframework/robot-go

go 1.25.1

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/chzyer/readline v1.5.1
	github.com/gorilla/websocket v1.5.3
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5 // indirect
