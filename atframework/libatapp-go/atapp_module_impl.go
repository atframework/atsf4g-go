package libatapp

import (
	"context"
)

// App 应用接口
type AppModuleImpl interface {
	GetApp() *AppImpl

	// Call this callback when a module is added into atapp for the first time
	OnBind()

	// Call this callback when a module is removed from atapp
	OnUnbind()

	// This callback is called after load configure and before initialization(include log)
	Setup(parent context.Context) error

	// This function will be called after reload and before init
	SetupLog(parent context.Context) error

	// This callback is called to initialize a module
	Init(parent context.Context) error

	// This callback is called after all modules are initialized successfully and the atapp is ready to run
	Ready()

	// This callback is called after configure is reloaded
	Reload() error

	// This callback may be called more than once, when the first return false, this module will be disabled.
	Stop() (bool, error)

	// This callback only will be call once after all module stopped
	Cleanup()

	// This callback be called if the module can not be stopped even in a long time.
	// After this event, all module and atapp will be forced stopped.
	Timeout()

	IsActived() bool
	Active()
	Unactive()
}

type AppModuleBase struct {
	actived bool
}

func (m *AppModuleBase) IsActived() bool {
	return m.actived
}

func (m *AppModuleBase) Active() {
	m.actived = true
}

func (m *AppModuleBase) Unactive() {
	m.actived = false
}
