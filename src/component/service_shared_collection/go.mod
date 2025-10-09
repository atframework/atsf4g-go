module github.com/atframework/atsf4g-go/component-service_shared_collection

go 1.25.1

replace github.com/atframework/libatapp-go => ../../../atframework/libatapp-go

replace github.com/atframework/atsf4g-go/component-config => ../config

replace github.com/atframework/atsf4g-go/component-protocol-public => ../protocol/public

replace github.com/xresloader/xresloader => ../../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../../third_party/xresloader/protocols/code

require (
	github.com/atframework/atsf4g-go/component-config v0.0.0-00010101000000-000000000000
	github.com/atframework/libatapp-go v0.0.0-00010101000000-000000000000
)

require (
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000 // indirect
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	github.com/xresloader/xres-code-generator v0.0.0-00010101000000-000000000000 // indirect
	github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/sync v0.17.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
