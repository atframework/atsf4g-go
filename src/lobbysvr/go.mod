module github.com/atframework/atsf4g-go/service-lobbysvr

go 1.25.1

replace github.com/atframework/atsf4g-go/component-service_shared_collection => ../component/service_shared_collection

replace github.com/atframework/atsf4g-go/component-user_controller => ../component/user_controller

replace github.com/atframework/atsf4g-go/component-dispatcher => ../component/dispatcher

replace github.com/atframework/atsf4g-go/component-config => ../component/config

replace github.com/atframework/atsf4g-go/component-logical_time => ../component/logical_time

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

replace github.com/atframework/atsf4g-go/component-db => ../component/db

replace github.com/atframework/atsf4g-go/component-router => ../component/router

replace github.com/atframework/atsf4g-go/component-operation-support-system => ../component/operation_support_system

replace github.com/atframework/atsf4g-go/component-uuid => ../component/uuid

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/atframework/atsf4g-go/component-protocol-private => ../component/protocol/private

replace github.com/atframework/atsf4g-go/component-protocol-public => ../component/protocol/public

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-config v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-db v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-dispatcher v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-logical_time v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-operation-support-system v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-protocol-private v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-router v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-service_shared_collection v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-user_controller v0.0.0-00010101000000-000000000000
	github.com/atframework/libatapp-go v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	github.com/redis/go-redis/v9 v9.16.0 // indirect
	github.com/xresloader/xres-code-generator v0.0.0-00010101000000-000000000000 // indirect
	github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/sync v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
