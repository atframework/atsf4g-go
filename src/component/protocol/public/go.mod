module github.com/atframework/atsf4g-go/component-protocol-public

go 1.25.1

replace github.com/atframework/atframe-utils-go => ../../../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go/protocol => ../../../../atframework/libatapp-go/protocol

replace github.com/xresloader/xresloader => ../../../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../../../third_party/xresloader/protocols/code

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/xresloader/xres-code-generator v0.0.0-00010101000000-000000000000
	github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.9
)
