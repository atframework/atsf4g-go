module github.com/atframework/libatapp-go

go 1.25.1

replace github.com/atframework/atframe-utils-go => ../atframe-utils-go

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/panjf2000/ants/v2 v2.11.3
	google.golang.org/protobuf v1.36.9
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/sync v0.11.0 // indirect
