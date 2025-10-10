module github.com/atframework/atsf4g-go/component-protocol-private

go 1.25.0

replace github.com/atframework/atframe-utils-go => ../../../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../../../atframework/libatapp-go

replace github.com/atframework/atsf4g-go/component-protocol-public => ../public

replace github.com/xresloader/xresloader => ../../../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../../../third_party/xresloader/protocols/code

require google.golang.org/protobuf v1.36.9

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000
	github.com/atframework/libatapp-go v0.0.0-00010101000000-000000000000
)

require (
	github.com/xresloader/xres-code-generator v0.0.0-00010101000000-000000000000 // indirect
	github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000 // indirect
)
