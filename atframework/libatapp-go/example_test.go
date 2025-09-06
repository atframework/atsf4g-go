package libatapp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// 示例模块实现
type ExampleModule struct {
	AppModuleBase

	name   string
	inited bool
}

func NewExampleModule(name string) *ExampleModule {
	return &ExampleModule{
		name: name,
	}
}

func (m *ExampleModule) Name() string {
	return m.name
}

func (m *ExampleModule) OnBind() {
	fmt.Printf("Module %s: OnBind called\n", m.name)
}

func (m *ExampleModule) OnUnbind() {
	fmt.Printf("Module %s: OnUnbind called\n", m.name)
}

func (m *ExampleModule) Setup(parent context.Context) error {
	fmt.Printf("Module %s: Setup called\n", m.name)
	return nil
}

func (m *ExampleModule) SetupLog(parent context.Context) error {
	fmt.Printf("Module %s: SetupLog called\n", m.name)
	return nil
}

func (m *ExampleModule) Init(parent context.Context) error {
	fmt.Printf("Module %s: Init called\n", m.name)
	m.inited = true
	return nil
}

func (m *ExampleModule) Ready() {
	fmt.Printf("Module %s: Ready called\n", m.name)
}

func (m *ExampleModule) Reload() error {
	fmt.Printf("Module %s: Reload called\n", m.name)
	return nil
}

func (m *ExampleModule) Stop() (bool, error) {
	fmt.Printf("Module %s: Stop called\n", m.name)
	return true, nil
}

func (m *ExampleModule) Cleanup() {
	fmt.Printf("Module %s: Cleanup called\n", m.name)
	m.inited = false
}

func (m *ExampleModule) Timeout() {
	fmt.Printf("Module %s: Timeout called\n", m.name)
}

func (m *ExampleModule) Tick(parent context.Context) bool {
	fmt.Printf("Module %s: Tick called\n", m.name)
	return false
}

// 测试基本功能
func TestAppBasicFunctionality(t *testing.T) {
	app := CreateAppInstance().(*AppInstance)

	// 设置基本配置
	config := app.GetConfig()
	config.AppId = 0x12345678
	config.TypeId = 0x1001
	config.TypeName = "TestApp"
	config.AppName = "test-app"
	config.AppVersion = "1.0.0-test"

	// 添加示例模块
	module1 := NewExampleModule("Module1")
	module2 := NewExampleModule("Module2")

	if err := app.AddModule(module1); err != nil {
		t.Fatalf("Failed to add module1: %v", err)
	}

	if err := app.AddModule(module2); err != nil {
		t.Fatalf("Failed to add module2: %v", err)
	}

	// 测试初始化 - 传入空参数避免测试标志冲突
	if err := app.Init(nil); err != nil {
		t.Fatalf("Failed to initialize app: %v", err)
	}

	// 验证状态
	if !app.IsInited() {
		t.Error("App should be initialized")
	}

	if app.GetAppId() != 0x12345678 {
		t.Errorf("Expected AppId 0x12345678, got 0x%x", app.GetAppId())
	}

	if app.GetAppName() != "test-app" {
		t.Errorf("Expected AppName 'test-app', got '%s'", app.GetAppName())
	}

	// 测试状态标志
	if !app.CheckFlag(AppFlagInitialized) {
		t.Error("App should have initialized flag set")
	}

	// 测试停止
	if err := app.Stop(); err != nil {
		t.Fatalf("Failed to stop app: %v", err)
	}

	if !app.IsClosed() {
		t.Error("App should be closed after stop")
	}
}

// 测试配置管理
func TestConfigManagement(t *testing.T) {
	app := CreateAppInstance().(*AppInstance)

	config := app.GetConfig()

	// 测试默认值
	if config.TickInterval != 8*time.Millisecond {
		t.Errorf("Expected TickInterval 8ms, got %v", config.TickInterval)
	}

	if config.StopTimeout != 30*time.Second {
		t.Errorf("Expected StopTimeout 30s, got %v", config.StopTimeout)
	}

	// 测试配置加载
	if err := app.LoadConfig("test.conf"); err != nil {
		t.Errorf("LoadConfig failed: %v", err)
	}

	if config.ConfigFile != "test.conf" {
		t.Errorf("Expected ConfigFile 'test.conf', got '%s'", config.ConfigFile)
	}
}
