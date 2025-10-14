module github.com/atframework/atsf4g-go/robot

go 1.25.1

require (
	github.com/atframework/atsf4g-go/component-protocol-public v0.0.0-00010101000000-000000000000
	github.com/atframework/atsf4g-go/service-lobbysvr v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.3
	github.com/shirou/gopsutil/v4 v4.25.9
	google.golang.org/protobuf v1.36.9
)

require (
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/xresloader/xresloader v0.0.0-00010101000000-000000000000 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace github.com/atframework/atsf4g-go/service-lobbysvr => ../lobbysvr

replace github.com/atframework/atsf4g-go/component-protocol-public => ../component/protocol/public

replace github.com/xresloader/xresloader => ../../third_party/xresloader/protocols/core

replace github.com/xresloader/xres-code-generator => ../../third_party/xresloader/protocols/code
