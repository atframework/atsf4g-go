module github.com/atframework/atsf4g-go/component-protocol-private

go 1.25.0

replace github.com/atframework/libatapp-go/protocol => ../../../../atframework/libatapp-go/protocol

replace github.com/atframework/atsf4g-go/component-protocol-public => ../public

require (
	github.com/atframework/libatapp-go/protocol v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.9
)
