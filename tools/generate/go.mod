module github.com/atframework/atsf4g-go/tools/generate

go 1.25.1

replace github.com/atframework/libatapp-go => ../../atframework/libatapp-go

replace github.com/atframework/atframe-utils-go => ../../atframework/atframe-utils-go

replace github.com/atframework/atsf4g-go/tools/project-settings => ../project-settings

require (
	github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/tools/project-settings v0.0.0-20251011042359-6106ca1e1da6
)

require golang.org/x/sys v0.37.0 // indirect
