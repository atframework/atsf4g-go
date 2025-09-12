module github.com/atframework/atsf4g-go/src/lobbysvr

go 1.25.0

replace github.com/atframework/atsf4g-go/src/component/service_shared_collection => ../component/service_shared_collection

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

replace github.com/atframework/libatapp-go/protocol => ../../atframework/libatapp-go/protocol

require (
	github.com/atframework/atsf4g-go/src/component/protocol/public v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/src/component/service_shared_collection v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.9
)

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/atframework/atsf4g-go/src/component/protocol/private => ../component/protocol/private

replace github.com/atframework/atsf4g-go/src/component/protocol/public => ../component/protocol/public

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code

require github.com/atframework/libatapp-go v0.0.0-00010101000000-000000000000 // indirect

require (
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	golang.org/x/sync v0.11.0 // indirect
)
