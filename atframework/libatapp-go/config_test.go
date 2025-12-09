package libatapp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	atframe_protocol "github.com/atframework/libatapp-go/protocol/atframe"
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

// loadEnvFile 从文件加载环境变量，返回已设置的环境变量key列表以便清理
func loadEnvFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和注释
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// 解析 KEY=VALUE 格式
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		// 保留 value 的尾部空格，只截断左边的空格
		value := strings.TrimLeft(parts[1], " \t")

		if err := os.Setenv(key, value); err != nil {
			return keys, fmt.Errorf("failed to set env %s: %w", key, err)
		}
		keys = append(keys, key)
	}

	if err := scanner.Err(); err != nil {
		return keys, fmt.Errorf("error reading env file: %w", err)
	}

	return keys, nil
}

// clearEnvVars 清理指定的环境变量
func clearEnvVars(keys []string) {
	for _, key := range keys {
		os.Unsetenv(key)
	}
}

// verifyRootFields 验证根级别字段
func verifyRootFields(t *testing.T, cfg *atframe_protocol.AtappConfigure) {
	t.Helper()
	assert := assert.New(t)
	assert.Equal(uint64(0x00001234), cfg.GetId(), "atapp.id should match")
	assert.Equal("sample_echo_svr-1", cfg.GetName(), "atapp.name should match")
	assert.Equal(uint64(1), cfg.GetTypeId(), "atapp.type_id should match")
	assert.Equal("sample_echo_svr", cfg.GetTypeName(), "atapp.type_name should match")
}

// verifyBusConfig 验证 bus 配置
func verifyBusConfig(t *testing.T, cfg *atframe_protocol.AtappConfigure) {
	t.Helper()
	assert := assert.New(t)
	bus := cfg.GetBus()
	if !assert.NotNil(bus, "bus config should not be nil") {
		return
	}

	assert.Equal([]string{"ipv6://:::21437"}, bus.GetListen(), "bus.listen should contain single endpoint")
	assert.Equal([]string{"0/16"}, bus.GetSubnets(), "bus.subnets should contain single subnet")
	assert.Equal("", bus.GetProxy(), "bus.proxy should be empty string")
	assert.Equal(int32(256), bus.GetBacklog(), "bus.backlog should match")
	assert.Equal(uint64(5), bus.GetAccessTokenMaxNumber(), "bus.access_token_max_number should match")
	if assert.NotNil(bus.GetFirstIdleTimeout(), "first_idle_timeout should be parsed") {
		assert.Equal(30*time.Second, bus.GetFirstIdleTimeout().AsDuration(), "bus.first_idle_timeout should match")
	}
	if assert.NotNil(bus.GetPingInterval(), "ping_interval should be parsed") {
		assert.Equal(60*time.Second, bus.GetPingInterval().AsDuration(), "bus.ping_interval should match")
	}
	if assert.NotNil(bus.GetRetryInterval(), "retry_interval should be parsed") {
		assert.Equal(3*time.Second, bus.GetRetryInterval().AsDuration(), "bus.retry_interval should match")
	}
	assert.Equal(uint64(3), bus.GetFaultTolerant(), "bus.fault_tolerant should match")
	assert.Equal(uint64(256*1024), bus.GetMsgSize(), "bus.msg_size should match")
	assert.Equal(uint64(8*1024*1024), bus.GetRecvBufferSize(), "bus.recv_buffer_size should match")
	assert.Equal(uint64(2*1024*1024), bus.GetSendBufferSize(), "bus.send_buffer_size should match")
	assert.Equal(uint64(0), bus.GetSendBufferNumber(), "bus.send_buffer_number should match")

	// 默认值验证
	assert.Equal(int32(1000), bus.GetLoopTimes(), "bus.loop_times should be 1000 by default")
}

// verifyTimerConfig 验证 timer 配置
func verifyTimerConfig(t *testing.T, cfg *atframe_protocol.AtappConfigure) {
	t.Helper()
	assert := assert.New(t)
	timer := cfg.GetTimer()
	if !assert.NotNil(timer, "timer config should not be nil") {
		return
	}

	if assert.NotNil(timer.GetTickInterval(), "tick_interval should be parsed") {
		assert.Equal(32*time.Millisecond, timer.GetTickInterval().AsDuration(), "timer.tick_interval should match")
	}
	if assert.NotNil(timer.GetStopTimeout(), "stop_timeout should be parsed") {
		assert.Equal(10*time.Second, timer.GetStopTimeout().AsDuration(), "timer.stop_timeout should match")
	}
	if assert.NotNil(timer.GetStopInterval(), "stop_interval should be parsed") {
		assert.Equal(256*time.Millisecond, timer.GetStopInterval().AsDuration(), "timer.stop_interval should match")
	}
}

// verifyEtcdConfig 验证 etcd 配置
func verifyEtcdConfig(t *testing.T, etcd *atframe_protocol.AtappEtcd) {
	t.Helper()
	assert := assert.New(t)
	if !assert.NotNil(etcd, "etcd config should not be nil") {
		return
	}

	assert.False(etcd.GetEnable(), "etcd.enable should be false")
	assert.Equal([]string{"http://127.0.0.1:2375", "http://127.0.0.1:2376", "http://127.0.0.1:2377"}, etcd.GetHosts(), "etcd.hosts should match order")
	assert.Equal("/atapp/services/astf4g/", etcd.GetPath(), "etcd.path should match")
	assert.Equal("", etcd.GetAuthorization(), "etcd.authorization should be empty string")
	if initCfg := etcd.GetInit(); assert.NotNil(initCfg, "etcd.init should not be nil") {
		if assert.NotNil(initCfg.GetTickInterval(), "etcd.init.tick_interval should be parsed") {
			assert.Equal(16*time.Millisecond, initCfg.GetTickInterval().AsDuration(), "etcd.init.tick_interval should match min bound")
		}
		// etcd.init.timeout is commented out in yaml, so it uses default value from proto extension
		// We only verify it's not nil if it exists
		// Note: The default value comes from proto extension, which may or may not be set
	}
}

// verifyLogConfig 验证 log 配置
func verifyLogConfig(t *testing.T, logCfg *atframe_protocol.AtappLog) {
	t.Helper()
	assert := assert.New(t)
	if !assert.NotNil(logCfg, "log config should not be nil") {
		return
	}

	assert.Equal("debug", logCfg.GetLevel(), "log.level should match")
	categories := logCfg.GetCategory()
	assert.Len(categories, 2, "log.category length should match")

	for _, cat := range categories {
		switch cat.GetName() {
		case "db":
			assert.Equal(int32(1), cat.GetIndex(), "db category index should match")
			assert.Equal("[Log %L][%F %T.%f]: ", cat.GetPrefix(), "db prefix should match")
			if stack := cat.GetStacktrace(); assert.NotNil(stack) {
				assert.Equal("disable", stack.GetMin(), "db stacktrace.min should match")
				assert.Equal("disable", stack.GetMax(), "db stacktrace.max should match")
			}
		case "default":
			assert.Equal(int32(0), cat.GetIndex(), "default category index should match")
			assert.Equal("[Log %L][%F %T.%f][%s:%n(%C)]: ", cat.GetPrefix(), "default prefix should match")
			if stack := cat.GetStacktrace(); assert.NotNil(stack) {
				assert.Equal("error", stack.GetMin(), "default stacktrace.min should match")
				assert.Equal("fatal", stack.GetMax(), "default stacktrace.max should match")
			}
			sinks := cat.GetSink()
			assert.Len(sinks, 4, "default category sinks should match count")
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
							assert.Equal(uint32(10), rotate.GetNumber(), "file rotate.number should match")
							assert.Equal(uint64(10485760), rotate.GetSize(), "file rotate.size should match")
						}
						key := sinkKey{typeName: "file", fileName: fileBackend.GetFile()}
						found[key] = true
						switch fileBackend.GetFile() {
						case "../log/sample_echo_svr.error.%N.log":
							assert.Equal("../log/sample_echo_svr.error.log", fileBackend.GetWritingAlias(), "error file alias should match")
						case "../log/sample_echo_svr.all.%N.log":
							assert.Equal("../log/sample_echo_svr.all.log", fileBackend.GetWritingAlias(), "all file alias should match")
						default:
							t.Errorf("unexpected file sink %s", fileBackend.GetFile())
						}
					}
				case "stderr", "stdout":
					found[sinkKey{typeName: sink.GetType()}] = true
				default:
					t.Errorf("unexpected sink type %s", sink.GetType())
				}
			}
			assert.True(found[sinkKey{typeName: "file", fileName: "../log/sample_echo_svr.error.%N.log"}], "first file sink missing")
			assert.True(found[sinkKey{typeName: "file", fileName: "../log/sample_echo_svr.all.%N.log"}], "second file sink missing")
			assert.True(found[sinkKey{typeName: "stderr"}], "stderr sink missing")
			assert.True(found[sinkKey{typeName: "stdout"}], "stdout sink missing")
		default:
			t.Errorf("unexpected log category %s", cat.GetName())
		}
	}
}

// verifyMetadata 验证 metadata 配置 (仅 YAML 支持)
func verifyMetadata(t *testing.T, cfg *atframe_protocol.AtappConfigure) {
	t.Helper()
	assert := assert.New(t)
	metadata := cfg.GetMetadata()
	if assert.NotNil(metadata, "metadata must be parsed") {
		assert.Equal("v1", metadata.GetApiVersion(), "metadata.api_version should match")
		labels := metadata.GetLabels()
		assert.Len(labels, 2, "metadata.labels size should match")
		assert.Equal("test", labels["deployment.environment"], "metadata.labels.deployment.environment should match")
		assert.Equal("cn", labels["deployment.region"], "metadata.labels.deployment.region should match")
	}
}

// runConfigVerification 运行所有配置验证
func runConfigVerification(t *testing.T, cfg *atframe_protocol.AtappConfigure, logCfg *atframe_protocol.AtappLog, includeMetadata bool) {
	t.Helper()

	t.Run("root fields", func(t *testing.T) {
		verifyRootFields(t, cfg)
	})

	if includeMetadata {
		t.Run("metadata", func(t *testing.T) {
			verifyMetadata(t, cfg)
		})
	}

	t.Run("bus config", func(t *testing.T) {
		verifyBusConfig(t, cfg)
	})

	t.Run("timer config", func(t *testing.T) {
		verifyTimerConfig(t, cfg)
	})

	t.Run("etcd config", func(t *testing.T) {
		verifyEtcdConfig(t, cfg.Etcd)
	})

	t.Run("log config", func(t *testing.T) {
		verifyLogConfig(t, logCfg)
	})
}

// loadExistedKeyFile 从文件加载环境变量，返回已设置的环境变量key列表以便清理
func loadExistedKeyFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和注释
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		keys = append(keys, trimmedLine)
	}

	if err := scanner.Err(); err != nil {
		return keys, fmt.Errorf("error reading env file: %w", err)
	}

	return keys, nil
}

func verifyAppConfigExistedIndex(t *testing.T, existedAppKeys *ConfigExistedIndex) {
	appKeys, _ := loadExistedKeyFile("atapp_configure_loader_test_app_keys.txt")
	t.Helper()
	assert := assert.New(t)
	assert.NotEmpty(appKeys, "app existed keys should not be empty")

	for _, key := range appKeys {
		_, exists := existedAppKeys.MutableExistedSet()[key]
		assert.True(exists, "app config key %s should exist in index", key)
	}
}

func verifyEtcdLogConfigExistedIndex(t *testing.T, existedEtcdKeys *ConfigExistedIndex, existedLogKeys *ConfigExistedIndex) {
	etcdKeys, _ := loadExistedKeyFile("atapp_configure_loader_test_etcd_keys.txt")
	logKeys, _ := loadExistedKeyFile("atapp_configure_loader_test_log_keys.txt")

	t.Helper()
	assert := assert.New(t)
	assert.NotEmpty(etcdKeys, "etcd existed keys should not be empty")
	assert.NotEmpty(logKeys, "log existed keys should not be empty")

	for _, key := range etcdKeys {
		_, exists := existedEtcdKeys.MutableExistedSet()[key]
		assert.True(exists, "etcd config key %s should exist in index", key)
	}

	for _, key := range logKeys {
		_, exists := existedLogKeys.MutableExistedSet()[key]
		assert.True(exists, "log config key %s should exist in index", key)
	}
}

// TestConfigManagementFromYaml 测试从 YAML 文件加载配置
func TestConfigManagementFromYaml(t *testing.T) {
	app := CreateAppInstance().(*AppInstance)
	config := app.GetConfig()

	// 测试配置加载
	existedAppKeys := CreateConfigExistedIndex()
	if err := app.LoadConfig("atapp_configure_loader_test.yaml", "atapp", "ATAPP", existedAppKeys); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.ConfigFile != "atapp_configure_loader_test.yaml" {
		t.Errorf("Expected ConfigFile 'atapp_configure_loader_test.yaml', got '%s'", config.ConfigFile)
	}

	cfg := app.config.ConfigPb
	if cfg == nil {
		t.Fatalf("config proto should be initialized after LoadConfig")
	}
	logCfg := app.config.ConfigLog
	if logCfg == nil {
		t.Fatalf("log config proto should be initialized after LoadConfig")
	}

	// 运行所有验证，包括 metadata（YAML 支持 map 类型）
	runConfigVerification(t, cfg, logCfg, true)
	verifyAppConfigExistedIndex(t, existedAppKeys)

	// 验证key存在索引
	existedEtcdKeys := CreateConfigExistedIndex()
	existedLogKeys := CreateConfigExistedIndex()
	etcdCfg := &atframe_protocol.AtappEtcd{}
	logsCfg := &atframe_protocol.AtappLog{}
	app.LoadConfigByPath(etcdCfg, "atapp.etcd", "ATAPP_ETCD", existedEtcdKeys, "")
	app.LoadLogConfigByPath(logsCfg, "atapp.log", "ATAPP_LOG", existedLogKeys, "")
	verifyEtcdLogConfigExistedIndex(t, existedEtcdKeys, existedLogKeys)
	verifyEtcdConfig(t, etcdCfg)
	verifyLogConfig(t, logsCfg)

	// YAML 特有的验证
	t.Run("yaml specific - bus.overwrite_listen_path", func(t *testing.T) {
		assert := assert.New(t)
		bus := cfg.GetBus()
		if assert.NotNil(bus) {
			assert.False(bus.GetOverwriteListenPath(), "bus.overwrite_listen_path should be false per yaml")
		}
	})
}

// verifyLogConfigBasic 验证基本的 log 配置（不包含 sink 的详细验证）
func verifyLogConfigBasic(t *testing.T, logCfg *atframe_protocol.AtappLog) {
	t.Helper()
	assert := assert.New(t)
	if !assert.NotNil(logCfg, "log config should not be nil") {
		return
	}

	assert.Equal("debug", logCfg.GetLevel(), "log.level should match")
	categories := logCfg.GetCategory()
	assert.Len(categories, 2, "log.category length should match")

	for _, cat := range categories {
		switch cat.GetName() {
		case "db":
			assert.Equal(int32(1), cat.GetIndex(), "db category index should match")
			assert.Equal("[Log %L][%F %T.%f]: ", cat.GetPrefix(), "db prefix should match")
			if stack := cat.GetStacktrace(); assert.NotNil(stack) {
				assert.Equal("disable", stack.GetMin(), "db stacktrace.min should match")
				assert.Equal("disable", stack.GetMax(), "db stacktrace.max should match")
			}
		case "default":
			assert.Equal(int32(0), cat.GetIndex(), "default category index should match")
			assert.Equal("[Log %L][%F %T.%f][%s:%n(%C)]: ", cat.GetPrefix(), "default prefix should match")
			if stack := cat.GetStacktrace(); assert.NotNil(stack) {
				assert.Equal("error", stack.GetMin(), "default stacktrace.min should match")
				assert.Equal("fatal", stack.GetMax(), "default stacktrace.max should match")
			}
		default:
			t.Errorf("unexpected log category %s", cat.GetName())
		}
	}
}

// runConfigVerificationForEnv 运行环境变量配置验证（简化版，跳过复杂嵌套结构）
func runConfigVerificationForEnv(t *testing.T, cfg *atframe_protocol.AtappConfigure, logCfg *atframe_protocol.AtappLog) {
	t.Helper()

	t.Run("root fields", func(t *testing.T) {
		verifyRootFields(t, cfg)
	})

	t.Run("bus config", func(t *testing.T) {
		verifyBusConfig(t, cfg)
	})

	t.Run("timer config", func(t *testing.T) {
		verifyTimerConfig(t, cfg)
	})

	t.Run("etcd config", func(t *testing.T) {
		verifyEtcdConfig(t, cfg.Etcd)
	})

	t.Run("log config basic", func(t *testing.T) {
		verifyLogConfigBasic(t, logCfg)
	})
}

// TestConfigManagementFromEnvironment 测试从环境变量加载配置
func TestConfigManagementFromEnvironment(t *testing.T) {
	// 加载环境变量文件
	existedAppKeys := CreateConfigExistedIndex()
	envKeys, err := loadEnvFile("atapp_configure_loader_test.env.txt")
	if err != nil {
		t.Fatalf("Failed to load env file: %v", err)
	}
	// 测试结束后清理环境变量
	defer clearEnvVars(envKeys)

	app := CreateAppInstance().(*AppInstance)

	// 测试配置加载
	if err := app.LoadConfig("", "atapp", "ATAPP", existedAppKeys); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg := app.config.ConfigPb
	if cfg == nil {
		t.Fatalf("config proto should be initialized after LoadConfig")
	}
	logCfg := app.config.ConfigLog
	if logCfg == nil {
		t.Fatalf("log config proto should be initialized after LoadConfig")
	}

	// 运行环境变量配置验证（包含完整的 log 验证）
	runConfigVerificationForEnv(t, cfg, logCfg)
	verifyAppConfigExistedIndex(t, existedAppKeys)

	// 验证key存在索引
	existedEtcdKeys := CreateConfigExistedIndex()
	existedLogKeys := CreateConfigExistedIndex()
	etcdCfg := &atframe_protocol.AtappEtcd{}
	logsCfg := &atframe_protocol.AtappLog{}
	app.LoadConfigByPath(etcdCfg, "atapp.etcd", "ATAPP_ETCD", existedEtcdKeys, "")
	app.LoadLogConfigByPath(logsCfg, "atapp.log", "ATAPP_LOG", existedLogKeys, "")
	verifyEtcdLogConfigExistedIndex(t, existedEtcdKeys, existedLogKeys)
	verifyEtcdConfig(t, etcdCfg)
	verifyLogConfig(t, logsCfg)
}
