module github.com/atframework/atsf4g-go/component-user_controller

go 1.25.1

replace github.com/atframework/atframe-utils-go => ../../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../../atframework/libatapp-go

replace github.com/atframework/libatapp-go/protocol => ../../../atframework/libatapp-go/protocol

replace github.com/atframework/atsf4g-go/component-protocol-public => ../protocol/public

replace github.com/atframework/atsf4g-go/component-protocol-private => ../protocol/private

replace github.com/xresloader/xresloader => ../../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../../third_party/xresloader/protocols/code

replace github.com/atframework/atsf4g-go/component-dispatcher => ../dispatcher

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-dispatcher v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-protocol-private v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000
	github.com/atframework/libatapp-go v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.9
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	golang.org/x/sync v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
