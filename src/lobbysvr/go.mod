module github.com/atframework/atsf4g-go/service-lobbysvr

go 1.25.1

replace github.com/atframework/atsf4g-go/component => ../component

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

require (
	github.com/atframework/atframe-utils-go v1.0.3
	github.com/atframework/atsf4g-go/component v0.0.0-00010101000000-000000000000
	github.com/atframework/libatapp-go v1.0.1
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.49.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/panjf2000/ants/v2 v2.12.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/redis/go-redis/v9 v9.18.0 // indirect
	github.com/xresloader/xres-code-generator v0.0.0-20260303071244-1796ac848341 // indirect
	github.com/xresloader/xresloader v2.23.5+incompatible // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
