module github.com/atframework/atsf4g-go/component

go 1.25.1

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

require (
	github.com/atframework/atframe-utils-go v1.0.3
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
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/panjf2000/ants/v2 v2.12.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.etcd.io/etcd/api/v3 v3.6.10 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.10 // indirect
	go.etcd.io/etcd/client/v3 v3.6.10 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/grpc v1.80.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
