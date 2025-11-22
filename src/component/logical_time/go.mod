module github.com/atframework/atsf4g-go/component-logical_time

go 1.25.1

replace github.com/atframework/atframe-utils-go => ../../../atframework/atframe-utils-go

replace github.com/atframework/libatapp-go => ../../../atframework/libatapp-go

replace github.com/xresloader/xresloader => ../../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../../third_party/xresloader/protocols/code

replace github.com/atframework/atsf4g-go/component-protocol-public => ../protocol/public

replace github.com/atframework/atsf4g-go/component-protocol-private => ../protocol/private

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
