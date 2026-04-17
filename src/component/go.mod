module github.com/atframework/atsf4g-go/component

go 1.25.1

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

require (
	github.com/atframework/atframe-utils-go v1.0.5-0.20260416024202-66c04636f055
	github.com/atframework/libatapp-go v1.0.1
	github.com/ebitengine/purego v0.10.0
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	github.com/xresloader/xres-code-generator v0.0.0-20260303071244-1796ac848341
	github.com/xresloader/xresloader v2.23.5+incompatible
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/panjf2000/ants/v2 v2.12.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
