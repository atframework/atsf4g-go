package libatapp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
// func TestAppBasicFunctionality(t *testing.T) {
// 	app := CreateAppInstance().(*AppInstance)

// 	// 设置基本配置
// 	config := app.GetConfig()
// 	config.AppVersion = "1.0.0-test"

// 	// 添加示例模块
// 	module1 := NewExampleModule("Module1")
// 	module2 := NewExampleModule("Module2")

// 	if err := AtappAddModule(app, module1); err != nil {
// 		t.Fatalf("Failed to add module1: %v", err)
// 	}

// 	if err := AtappAddModule(app, module2); err != nil {
// 		t.Fatalf("Failed to add module2: %v", err)
// 	}

// 	// 测试初始化 - 传入空参数避免测试标志冲突
// 	if err := app.Init(nil); err != nil {
// 		t.Fatalf("Failed to initialize app: %v", err)
// 	}

// 	// 验证状态
// 	if !app.IsInited() {
// 		t.Error("App should be initialized")
// 	}

// 	// 测试状态标志
// 	if !app.CheckFlag(AppFlagInitialized) {
// 		t.Error("App should have initialized flag set")
// 	}

// 	// 测试停止
// 	if err := app.Stop(); err != nil {
// 		t.Fatalf("Failed to stop app: %v", err)
// 	}

// 	if !app.IsClosed() {
// 		t.Error("App should be closed after stop")
// 	}
// }

// 测试配置管理
func TestConfigManagement(t *testing.T) {
	app := CreateAppInstance().(*AppInstance)

	config := app.GetConfig()

	// 测试配置加载
	if err := app.LoadConfig("atapp_configure_loader_test.yaml"); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.ConfigFile != "atapp_configure_loader_test.yaml" {
		t.Errorf("Expected ConfigFile 'atapp_configure_loader_test.yaml', got '%s'", config.ConfigFile)
	}

	cfg := app.config.ConfigPb
	if cfg == nil {
		t.Fatalf("config proto should be initialized after LoadConfig")
	}

	// YAML 中的字段 id_mask 在 atapp_configure 中不存在，保留注释方便 proto 增加字段后再补测

	t.Run("root fields", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(uint64(0x00001234), cfg.GetId(), "atapp.id should match yaml")
		assert.Equal("sample_echo_svr-1", cfg.GetName(), "atapp.name should match yaml")
		assert.Equal(uint64(1), cfg.GetTypeId(), "atapp.type_id should match yaml")
		assert.Equal("sample_echo_svr", cfg.GetTypeName(), "atapp.type_name should match yaml")

		metadata := cfg.GetMetadata()
		if assert.NotNil(metadata, "metadata must be parsed") {
			assert.Equal("v1", metadata.GetApiVersion(), "metadata.api_version should match yaml")
			labels := metadata.GetLabels()
			assert.Len(labels, 2, "metadata.labels size should match yaml")
			assert.Equal("test", labels["deployment.environment"], "metadata.labels.deployment.environment should match yaml")
			assert.Equal("cn", labels["deployment.region"], "metadata.labels.deployment.region should match yaml")
		}
	})

	t.Run("bus config", func(t *testing.T) {
		assert := assert.New(t)
		bus := cfg.GetBus()
		if bus == nil {
			t.Fatalf("bus config should not be nil")
		}

		assert.Equal([]string{"ipv6://:::21437"}, bus.GetListen(), "bus.listen should contain single endpoint from yaml")
		assert.Equal([]string{"0/16"}, bus.GetSubnets(), "bus.subnets should contain single subnet from yaml")
		assert.Equal("", bus.GetProxy(), "bus.proxy should match yaml empty string")
		assert.Equal(int32(256), bus.GetBacklog(), "bus.backlog should match yaml")
		assert.Equal(uint64(5), bus.GetAccessTokenMaxNumber(), "bus.access_token_max_number should match yaml")
		assert.False(bus.GetOverwriteListenPath(), "bus.overwrite_listen_path should be false per yaml")
		if assert.NotNil(bus.GetFirstIdleTimeout(), "first_idle_timeout should be parsed") {
			assert.Equal(30*time.Second, bus.GetFirstIdleTimeout().AsDuration(), "bus.first_idle_timeout should match yaml")
		}
		if assert.NotNil(bus.GetPingInterval(), "ping_interval should be parsed") {
			assert.Equal(60*time.Second, bus.GetPingInterval().AsDuration(), "bus.ping_interval should match yaml")
		}
		if assert.NotNil(bus.GetRetryInterval(), "retry_interval should be parsed") {
			assert.Equal(3*time.Second, bus.GetRetryInterval().AsDuration(), "bus.retry_interval should match yaml")
		}
		assert.Equal(uint64(3), bus.GetFaultTolerant(), "bus.fault_tolerant should match yaml")
		assert.Equal(uint64(256*1024), bus.GetMsgSize(), "bus.msg_size should be converted from KB")
		assert.Equal(uint64(8*1024*1024), bus.GetRecvBufferSize(), "bus.recv_buffer_size should be converted from MB")
		assert.Equal(uint64(2*1024*1024), bus.GetSendBufferSize(), "bus.send_buffer_size should be converted from MB")
		assert.Equal(uint64(0), bus.GetSendBufferNumber(), "bus.send_buffer_number should match yaml explicit zero")
	})

	t.Run("timer config", func(t *testing.T) {
		assert := assert.New(t)
		timer := cfg.GetTimer()
		if timer == nil {
			t.Fatalf("timer config should not be nil")
		}
		if assert.NotNil(timer.GetTickInterval(), "tick_interval should be parsed") {
			assert.Equal(32*time.Millisecond, timer.GetTickInterval().AsDuration(), "timer.tick_interval should match yaml")
		}
		if assert.NotNil(timer.GetStopTimeout(), "stop_timeout should be parsed") {
			assert.Equal(10*time.Second, timer.GetStopTimeout().AsDuration(), "timer.stop_timeout should match yaml")
		}
		if assert.NotNil(timer.GetStopInterval(), "stop_interval should be parsed") {
			assert.Equal(256*time.Millisecond, timer.GetStopInterval().AsDuration(), "timer.stop_interval should match yaml")
		}
	})

	t.Run("etcd config", func(t *testing.T) {
		assert := assert.New(t)
		etcd := cfg.GetEtcd()
		if etcd == nil {
			t.Fatalf("etcd config should not be nil")
		}
		assert.False(etcd.GetEnable(), "etcd.enable should match yaml false")
		assert.Equal([]string{"http://127.0.0.1:2375", "http://127.0.0.1:2376", "http://127.0.0.1:2377"}, etcd.GetHosts(), "etcd.hosts should match yaml order")
		assert.Equal("/atapp/services/astf4g/", etcd.GetPath(), "etcd.path should match yaml")
		assert.Equal("", etcd.GetAuthorization(), "etcd.authorization should match yaml empty string")
		if initCfg := etcd.GetInit(); assert.NotNil(initCfg, "etcd.init should not be nil") {
			if assert.NotNil(initCfg.GetTickInterval(), "etcd.init.tick_interval should be parsed") {
				assert.Equal(16*time.Millisecond, initCfg.GetTickInterval().AsDuration(), "etcd.init.tick_interval should match yaml min bound")
			}
			if assert.NotNil(initCfg.GetTimeout(), "etcd.init.timeout should be parsed") {
				assert.Equal(10*time.Second, initCfg.GetTimeout().AsDuration(), "etcd.init.timeout should match yaml default value")
			}
		}
	})

	t.Run("log config", func(t *testing.T) {
		assert := assert.New(t)
		logCfg := cfg.GetLog()
		if logCfg == nil {
			t.Fatalf("log config should not be nil")
		}
		assert.Equal("debug", logCfg.GetLevel(), "log.level should match yaml")
		categories := logCfg.GetCategory()
		assert.Len(categories, 2, "log.category length should match yaml")
		for _, cat := range categories {
			switch cat.GetName() {
			case "db":
				assert.Equal(int32(1), cat.GetIndex(), "db category index should follow yaml reorder note")
				assert.Equal("[Log %L][%F %T.%f]: ", cat.GetPrefix(), "db prefix should match yaml")
				if stack := cat.GetStacktrace(); assert.NotNil(stack) {
					assert.Equal("disable", stack.GetMin(), "db stacktrace.min should match yaml")
					assert.Equal("disable", stack.GetMax(), "db stacktrace.max should match yaml")
				}
			case "default":
				assert.Equal(int32(0), cat.GetIndex(), "default category index should match yaml")
				assert.Equal("[Log %L][%F %T.%f][%s:%n(%C)]: ", cat.GetPrefix(), "default prefix should match yaml")
				if stack := cat.GetStacktrace(); assert.NotNil(stack) {
					assert.Equal("error", stack.GetMin(), "default stacktrace.min should match yaml")
					assert.Equal("fatal", stack.GetMax(), "default stacktrace.max should match yaml")
				}
				sinks := cat.GetSink()
				assert.Len(sinks, 4, "default category sinks should match yaml count")
				type sinkKey struct {
					typeName string
					fileName string
				}
				found := map[sinkKey]bool{}
				for _, sink := range sinks {
					switch sink.GetType() {
					case "file":
						fileBackend := sink.GetLogBackendFile()
						if assert.NotNil(fileBackend, "file backend should exist") {
							rotate := fileBackend.GetRotate()
							if assert.NotNil(rotate) {
								assert.Equal(uint32(10), rotate.GetNumber(), "file rotate.number should match yaml")
								assert.Equal(uint64(10485760), rotate.GetSize(), "file rotate.size should match yaml")
							}
							key := sinkKey{typeName: "file", fileName: fileBackend.GetFile()}
							found[key] = true
							switch fileBackend.GetFile() {
							case "../log/sample_echo_svr.error.%N.log":
								assert.Equal("../log/sample_echo_svr.error.log", fileBackend.GetWritingAlias(), "error file alias should match yaml")
							case "../log/sample_echo_svr.all.%N.log":
								assert.Equal("../log/sample_echo_svr.all.log", fileBackend.GetWritingAlias(), "all file alias should match yaml")
							default:
								t.Fatalf("unexpected file sink %s", fileBackend.GetFile())
							}
						}
					case "stderr", "stdout":
						found[sinkKey{typeName: sink.GetType()}] = true
					default:
						t.Fatalf("unexpected sink type %s", sink.GetType())
					}
				}
				assert.True(found[sinkKey{typeName: "file", fileName: "../log/sample_echo_svr.error.%N.log"}], "first file sink missing")
				assert.True(found[sinkKey{typeName: "file", fileName: "../log/sample_echo_svr.all.%N.log"}], "second file sink missing")
				assert.True(found[sinkKey{typeName: "stderr"}], "stderr sink missing")
				assert.True(found[sinkKey{typeName: "stdout"}], "stdout sink missing")
			default:
				t.Fatalf("unexpected log category %s", cat.GetName())
			}
		}
	})
}
