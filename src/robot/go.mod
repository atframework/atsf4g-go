module github.com/atframework/atsf4g-go/robot

go 1.25.1

require (
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/service-lobbysvr v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.3
	google.golang.org/protobuf v1.36.9
)

require github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000 // indirect

replace github.com/atframework/atsf4g-go/service-lobbysvr => ../lobbysvr

replace github.com/atframework/atsf4g-go/component-protocol-public => ../component/protocol/public

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code
