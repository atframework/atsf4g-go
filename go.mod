module github.com/atframework/atsf4g-go

go 1.27.4

tool (
	github.com/atframework/atframe-utils-go/buildkit/cmd/atbuild
	github.com/atframework/atframe-utils-go/protoc-gen-mutable
	github.com/atframework/atsf4g-go/tools/dev
	google.golang.org/protobuf/cmd/protoc-gen-go
)

replace (
	github.com/atframework/atframe-utils-go => ./atframework/atframe-utils-go
	github.com/atframework/atframe-utils-go/buildkit => ./atframework/atframe-utils-go/buildkit
	github.com/atframework/libatapp-go => ./atframework/libatapp-go
	github.com/atframework/libatbus-go => ./atframework/libatbus-go
	github.com/xresloader/xres-code-generator => ./third_party/xresloader/protocols/code
	github.com/xresloader/xresloader => ./third_party/xresloader/protocols/core
)

require (
	github.com/atframework/atframe-utils-go v1.0.5-0.20260416024202-66c04636f055
	github.com/atframework/atframe-utils-go/buildkit v0.1.0
	github.com/atframework/libatapp-go v1.0.1
	github.com/ebitengine/purego v0.10.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	github.com/xresloader/xres-code-generator v0.0.0-20260303071244-1796ac848341
	github.com/xresloader/xresloader v2.23.5+incompatible
	golang.org/x/crypto v0.49.0
	golang.org/x/sync v0.20.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/panjf2000/ants/v2 v2.11.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.3 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
