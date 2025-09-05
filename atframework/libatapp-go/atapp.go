package libatapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/panjf2000/ants/v2"
)

// App 应用模式
type AppMode int

const (
	AppModeCustom AppMode = iota
	AppModeStart
	AppModeStop
	AppModeReload
	AppModeInfo
	AppModeHelp
)

// App 状态标志
type AppFlag uint64

const (
	AppFlagInitializing AppFlag = 1 << iota
	AppFlagInitialized
	AppFlagRunning
	AppFlagStopping
	AppFlagStopped
	AppFlagTimeout
	AppFlagInCallback
	AppFlagInTick
	AppFlagDestroying
)

// App 配置
type AppConfig struct {
	AppId        uint64
	TypeId       uint64
	TypeName     string
	AppName      string
	AppIdentity  string
	HashCode     string
	AppVersion   string
	BuildVersion string
	ConfigFile   string
	PidFile      string
	ExecutePath  string

	// 定时器配置
	TickInterval     time.Duration
	TickRoundTimeout time.Duration
	StopTimeout      time.Duration
	InitTimeout      time.Duration

	// 日志配置
	StartupLog       []string
	StartupErrorFile string
}

// 消息类型
type Message struct {
	Type       int32
	Sequence   uint64
	Data       []byte
	Metadata   map[string]string
	SourceId   uint64
	SourceName string
}

// 事件处理函数类型
type EventHandler func(*AppInstance, *AppActionSender) int

// App 应用接口
type AppImpl interface {
	Run(arguments []string) error

	Init(arguments []string) error
	Stop() error
	Reload() error
	GetAppId() uint64
	GetTypeId() uint64
	GetTypeName() string
	GetAppName() string
	GetAppIdentity() string
	GetHashCode() string
	GetAppVersion() string
	GetBuildVersion() string

	AddModule(module AppModuleImpl) error

	// 消息相关
	SendMessage(targetId uint64, msgType int32, data []byte) error
	SendMessageByName(targetName string, msgType int32, data []byte) error

	// 事件相关
	SetEventHandler(eventType string, handler EventHandler)
	TriggerEvent(eventType string, args *AppActionSender) int

	// 配置相关
	GetConfig() *AppConfig
	LoadConfig(configFile string) error

	// 状态相关
	IsInited() bool
	IsRunning() bool
	IsClosing() bool
	IsClosed() bool
	CheckFlag(flag AppFlag) bool
	SetFlag(flag AppFlag, value bool) bool
}

type AppInstance struct {
	// 基础配置
	config AppConfig
	flags  uint64 // 原子操作的状态标志
	mode   AppMode

	// 命令行参数处理
	flagSet        *flag.FlagSet
	commandManager *CommandManager
	lastCommand    []string

	// 模块管理
	modules      []AppModuleImpl
	modulesMutex sync.RWMutex

	// 生命周期控制
	ctx    context.Context
	cancel context.CancelFunc

	// 定时器和事件
	tickTimer *time.Timer
	stopTimer *time.Timer
	tickMutex sync.Mutex

	// 事件处理
	eventHandlers map[string]EventHandler
	eventMutex    sync.RWMutex

	// 信号处理
	signalChan chan os.Signal

	// 日志
	logger *slog.Logger

	// 协程池
	workerPool *ants.PoolWithFunc

	// 统计信息
	stats struct {
		lastProcEventCount uint64
		tickCount          uint64
		moduleReloadCount  uint64
	}
}

func CreateAppInstance() AppImpl {
	ret := &AppInstance{
		mode:          AppModeCustom,
		eventHandlers: make(map[string]EventHandler),
		signalChan:    make(chan os.Signal, 1),
		logger:        slog.Default(),
	}

	ret.flagSet = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	ret.flagSet.Bool("v", false, "print version and exit")
	ret.flagSet.Bool("version", false, "print version and exit")
	ret.flagSet.Bool("h", false, "print help and exit")
	ret.flagSet.Bool("help", false, "print help and exit")
	ret.flagSet.String("c", "", "config file path")
	ret.flagSet.String("conf", "", "config file path")
	ret.flagSet.String("config", "", "config file path")
	ret.flagSet.String("p", "", "pid file path")
	ret.flagSet.String("pid", "", "pid file path")
	ret.flagSet.String("startup-log", "", "startup log file")
	ret.flagSet.String("startup-error-file", "", "startup error file")

	ret.ctx, ret.cancel = context.WithCancel(context.Background())

	// 设置默认配置
	ret.config.TickInterval = 8 * time.Millisecond
	ret.config.TickRoundTimeout = 128 * time.Millisecond
	ret.config.StopTimeout = 30 * time.Second
	ret.config.InitTimeout = 30 * time.Second
	ret.config.ExecutePath = os.Args[0]
	ret.config.AppVersion = "1.0.0"
	ret.config.BuildVersion = fmt.Sprintf("libatapp-go based atapp %s", ret.config.AppVersion)

	// 生成默认的应用名称和标识
	if ret.config.AppName == "" {
		ret.config.AppName = fmt.Sprintf("%s-0x%x", ret.config.TypeName, ret.config.AppId)
	}

	// 生成哈希码
	ret.generateHashCode()

	runtime.SetFinalizer(ret, func(app *AppInstance) {
		app.destroy()
	})

	return ret
}

func (app *AppInstance) destroy() {
	if app.IsInited() && !app.IsClosed() {
		app.close()
	}

	for _, m := range app.modules {
		if m.IsActived() {
			m.Unactive()
		}
		m.OnUnbind()
	}

	// TODO: endpoint 断开连接
	// TODO: connector 清理
}

// 生成哈希码
func (app *AppInstance) generateHashCode() {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s_%d_%s", app.config.AppName, app.config.AppId, app.config.ExecutePath)))
	app.config.HashCode = hex.EncodeToString(hasher.Sum(nil))
}

// 状态管理方法
func (app *AppInstance) CheckFlag(flag AppFlag) bool {
	return atomic.LoadUint64(&app.flags)&uint64(flag) != 0
}

func (app *AppInstance) SetFlag(flag AppFlag, value bool) bool {
	for {
		old := atomic.LoadUint64(&app.flags)
		var new uint64
		if value {
			new = old | uint64(flag)
		} else {
			new = old &^ uint64(flag)
		}
		if atomic.CompareAndSwapUint64(&app.flags, old, new) {
			return old&uint64(flag) != 0
		}
	}
}

func (app *AppInstance) IsInited() bool  { return app.CheckFlag(AppFlagInitialized) }
func (app *AppInstance) IsRunning() bool { return app.CheckFlag(AppFlagRunning) }
func (app *AppInstance) IsClosing() bool { return app.CheckFlag(AppFlagStopping) }
func (app *AppInstance) IsClosed() bool  { return app.CheckFlag(AppFlagStopped) }

func (app *AppInstance) AddModule(module AppModuleImpl) error {
	app.modulesMutex.Lock()
	defer app.modulesMutex.Unlock()

	app.modules = append(app.modules, module)
	module.OnBind()
	return nil
}

func (app *AppInstance) Init(arguments []string) error {
	if app.IsInited() {
		return nil
	}

	if app.CheckFlag(AppFlagInitializing) {
		return fmt.Errorf("recursive initialization detected")
	}

	app.SetFlag(AppFlagInitializing, true)
	defer app.SetFlag(AppFlagInitializing, false)

	// 解析命令行参数
	if err := app.setupOptions(arguments); err != nil {
		return fmt.Errorf("setup options failed: %w", err)
	}
	app.setupCommandManager()

	if app.mode == AppModeInfo {
		return nil
	}

	if app.mode == AppModeHelp {
		app.flagSet.PrintDefaults()
		return nil
	}

	// 初始化启动流程日志
	if err := app.setupStartupLog(); err != nil {
		return fmt.Errorf("setup startup log failed: %w", err)
	}

	// 设置信号处理
	if app.mode != AppModeCustom && app.mode != AppModeStop && app.mode != AppModeReload {
		if err := app.setupSignal(); err != nil {
			app.writeStartupErrorFile(err)
			return fmt.Errorf("setup signal failed: %w", err)
		}
	}

	// 加载配置
	if err := app.LoadConfig(app.config.ConfigFile); err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	if app.mode != AppModeCustom && app.mode != AppModeStop && app.mode != AppModeReload {
		return app.sendLastCommand()
	}

	// 初始化日志
	if err := app.setupLog(); err != nil {
		return fmt.Errorf("setup log failed: %w", err)
	}

	// 设置定时器
	if err := app.setupTickTimer(); err != nil {
		return fmt.Errorf("setup timer failed: %w", err)
	}

	// TODO: 初始化协程池大小可配置
	var err error
	app.workerPool, err = ants.NewPoolWithFunc(20480, func(args interface{}) {
		sender, ok := args.(*AppActionSender)
		if !ok {
			app.logger.Error("routine pool args type error, shouldn't happen!")
			return
		}
		app.processAction(sender)
	},
	// , ants.WithNonblocking(true)
	)
	if err != nil {
		return err
	}

	// 初始化所有模块
	app.modulesMutex.RLock()
	modules := make([]AppModuleImpl, len(app.modules))
	copy(modules, app.modules)
	app.modulesMutex.RUnlock()

	// Setup phase
	for _, m := range modules {
		if err := m.Setup(); err != nil {
			return fmt.Errorf("module setup failed: %w", err)
		}
	}

	// Setup log phase
	for _, m := range modules {
		if err := m.SetupLog(); err != nil {
			return fmt.Errorf("module setup log failed: %w", err)
		}
	}

	// Init phase
	for _, m := range modules {
		if err := m.Init(); err != nil {
			return fmt.Errorf("module init failed: %w", err)
		}
	}

	// Ready phase
	for _, m := range modules {
		if err := m.Ready(); err != nil {
			return fmt.Errorf("module ready failed: %w", err)
		}
	}

	app.SetFlag(AppFlagInitialized, true)
	app.SetFlag(AppFlagStopped, false)
	app.SetFlag(AppFlagStopping, false)

	return nil
}

func (app *AppInstance) Run(arguments []string) error {
	if !app.IsInited() {
		if err := app.Init(arguments); err != nil {
			return err
		}
	}

	app.SetFlag(AppFlagRunning, true)
	defer app.SetFlag(AppFlagRunning, false)

	// 主事件循环
	tickTimer := time.NewTicker(app.config.TickInterval)
	defer tickTimer.Stop()

	for !app.IsClosing() && !app.IsClosed() {
		select {
		case <-app.ctx.Done():
			return nil
		case sig := <-app.signalChan:
			if sig == syscall.SIGTERM || sig == syscall.SIGQUIT {
				app.logger.Info("Received signal: %v, stopping...", sig)
				app.Stop()
			}
		case <-tickTimer.C:
			if err := app.tick(); err != nil {
				app.logger.Error("Tick error: %v", err)
			}
		}
	}

	return nil
}

func (app *AppInstance) close() error {
	// TODO: all modules stop
	// TODO: all modules cleanup

	// TODO: cleanup event

	// TODO: close tick timer

	// TODO: cleanup pidfile

	// TODO: finally callback
	return nil
}

func (app *AppInstance) RunCommand(arguments []string) error {
	// TODO: 实现命令处理逻辑
	return nil
}

func (app *AppInstance) sendLastCommand() error {
	// TODO: 发送远程指令
	return nil
}

func (app *AppInstance) Stop() error {
	if app.IsClosing() || app.IsClosed() {
		return nil
	}

	app.SetFlag(AppFlagStopping, true)
	app.cancel()

	app.modulesMutex.RLock()
	modules := make([]AppModuleImpl, len(app.modules))
	copy(modules, app.modules)
	app.modulesMutex.RUnlock()

	// 停止所有模块
	for _, m := range modules {
		m.Stop()
	}

	// 清理所有模块
	for _, m := range modules {
		m.Cleanup()
	}

	app.SetFlag(AppFlagStopped, true)
	return nil
}

func (app *AppInstance) Reload() error {
	// 重新加载配置
	if err := app.LoadConfig(app.config.ConfigFile); err != nil {
		return fmt.Errorf("reload config failed: %w", err)
	}

	app.modulesMutex.RLock()
	modules := make([]AppModuleImpl, len(app.modules))
	copy(modules, app.modules)
	app.modulesMutex.RUnlock()

	// 重新加载所有模块
	for _, m := range modules {
		if err := m.Reload(); err != nil {
			return fmt.Errorf("module reload failed: %w", err)
		}
	}

	atomic.AddUint64(&app.stats.moduleReloadCount, 1)
	return nil
}

// Getter methods
func (app *AppInstance) GetAppId() uint64        { return app.config.AppId }
func (app *AppInstance) GetTypeId() uint64       { return app.config.TypeId }
func (app *AppInstance) GetTypeName() string     { return app.config.TypeName }
func (app *AppInstance) GetAppName() string      { return app.config.AppName }
func (app *AppInstance) GetAppIdentity() string  { return app.config.AppIdentity }
func (app *AppInstance) GetHashCode() string     { return app.config.HashCode }
func (app *AppInstance) GetAppVersion() string   { return app.config.AppVersion }
func (app *AppInstance) GetBuildVersion() string { return app.config.BuildVersion }
func (app *AppInstance) GetConfig() *AppConfig   { return &app.config }

// 配置管理
func (app *AppInstance) LoadConfig(configFile string) error {
	if configFile != "" {
		app.config.ConfigFile = configFile
		// TODO: 实际的配置文件解析逻辑
		app.logger.Info("Loading config from: %s", configFile)
	}
	return nil
}

// 消息相关
func (app *AppInstance) SendMessage(targetId uint64, msgType int32, data []byte) error {
	// TODO: 实现消息发送逻辑
	app.logger.Debug("Sending message to %d, type: %d, size: %d", targetId, msgType, len(data))
	return nil
}

func (app *AppInstance) SendMessageByName(targetName string, msgType int32, data []byte) error {
	// TODO: 实现按名称发送消息逻辑
	app.logger.Debug("Sending message to %s, type: %d, size: %d", targetName, msgType, len(data))
	return nil
}

// 事件相关
func (app *AppInstance) SetEventHandler(eventType string, handler EventHandler) {
	app.eventMutex.Lock()
	defer app.eventMutex.Unlock()
	app.eventHandlers[eventType] = handler
}

func (app *AppInstance) TriggerEvent(eventType string, args *AppActionSender) int {
	app.eventMutex.RLock()
	handler, exists := app.eventHandlers[eventType]
	app.eventMutex.RUnlock()

	if exists {
		return handler(app, args)
	}
	return 0
}

// 内部辅助方法
func (app *AppInstance) setupOptions(arguments []string) error {
	// 在测试环境中，如果 arguments 为 nil，跳过参数解析
	if arguments == nil {
		return nil
	}

	// 避免与测试标志冲突，创建新的 FlagSet 而不是使用全局的
	if len(arguments) > 0 {
		// 过滤掉测试相关的标志
		var filteredArgs []string
		for _, arg := range arguments {
			if !contains([]string{"-test.v", "-test.run", "-test.testlogfile", "-test.timeout", "-test.count", "-test.cpu", "-test.parallel", "-test.bench", "-test.benchmem", "-test.short"}, arg) &&
				!startsWith(arg, "-test.") {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		return app.flagSet.Parse(filteredArgs)
	}

	// 检查是否在测试环境中
	for _, arg := range os.Args {
		if startsWith(arg, "-test.") {
			// 在测试环境中，不解析 os.Args
			return nil
		}
	}

	if err := app.flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	if app.flagSet.Lookup("v").Value.String() == "true" || app.flagSet.Lookup("version").Value.String() == "true" {
		app.mode = AppModeInfo
		println(app.GetBuildVersion())
		return nil
	}

	if app.flagSet.Lookup("h").Value.String() == "true" || app.flagSet.Lookup("help").Value.String() == "true" {
		app.mode = AppModeHelp
		return nil
	}

	// 检查位置参数以确定命令
	args := app.flagSet.Args()
	if len(args) > 0 {
		switch args[0] {
		case "start":
			app.mode = AppModeStart
		case "stop":
			app.mode = AppModeStop
			app.lastCommand = []string{"stop"}
		case "reload":
			app.mode = AppModeReload
			app.lastCommand = []string{"reload"}
		case "run":
			app.mode = AppModeCustom
			app.lastCommand = args[1:]
		}
	}

	return nil
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func (app *AppInstance) setupSignal() error {
	signal.Notify(app.signalChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGPIPE, syscall.SIGTERM, syscall.SIGQUIT)
	return nil
}

func (app *AppInstance) setupStartupLog() error {
	// TODO: 根据配置设置启动流程日志
	if len(app.config.StartupLog) > 0 {
		for _, logFile := range app.config.StartupLog {
			app.logger.Info("Setting up startup log: %s", logFile)
		}
	}
	return nil
}

func (app *AppInstance) setupLog() error {
	// TODO: 根据配置设置日志
	return nil
}

func (app *AppInstance) setupTickTimer() error {
	// 定时器在 Run 方法中设置
	if app.tickTimer == nil {
		app.tickTimer := time.NewTicker(app.config.TickInterval)
	} else {
		app.tickTimer.Reset(app.config.TickInterval)
	}
	return nil
}

func (app *AppInstance) tick() error {
	if app.CheckFlag(AppFlagInTick) {
		return nil
	}

	app.SetFlag(AppFlagInTick, true)
	defer app.SetFlag(AppFlagInTick, false)

	atomic.AddUint64(&app.stats.tickCount, 1)

	// 处理模块的tick
	app.modulesMutex.RLock()
	modules := make([]AppModuleImpl, len(app.modules))
	copy(modules, app.modules)
	app.modulesMutex.RUnlock()

	// TODO: 调用模块的tick方法（如果模块接口支持的话）

	return nil
}

// 辅助方法：写入PID文件
func (app *AppInstance) writePidFile() error {
	if app.config.PidFile == "" {
		return nil
	}

	pid := os.Getpid()
	pidData := fmt.Sprintf("%d\n", pid)

	return os.WriteFile(app.config.PidFile, []byte(pidData), 0644)
}

// 辅助方法：清理PID文件
func (app *AppInstance) cleanupPidFile() error {
	if app.config.PidFile == "" {
		return nil
	}

	if _, err := os.Stat(app.config.PidFile); err == nil {
		return os.Remove(app.config.PidFile)
	}
	return nil
}

// 辅助方法：写入启动失败标记文件
func (app *AppInstance) writeStartupErrorFile(err error) {
	if app.config.StartupErrorFile == "" {
		return
	}

	pidData := fmt.Sprintf("%v\n", err)
	os.WriteFile(app.config.StartupErrorFile, []byte(pidData), 0644)
}

// 辅助方法：设置默认值
func (app *AppInstance) setDefaults() {
	if app.config.AppName == "" && app.config.TypeName != "" {
		app.config.AppName = fmt.Sprintf("%s-0x%x", app.config.TypeName, app.config.AppId)
	}

	if app.config.AppIdentity == "" {
		// 生成基于可执行文件路径和配置的唯一标识
		absPath, _ := filepath.Abs(app.config.ExecutePath)
		identity := fmt.Sprintf("%s\n%s\nid: %d\nname: %s",
			absPath, app.config.ConfigFile, app.config.AppId, app.config.AppName)
		hasher := sha256.New()
		hasher.Write([]byte(identity))
		app.config.AppIdentity = hex.EncodeToString(hasher.Sum(nil))
	}
}

// 命令处理相关
type CommandHandler func(*AppInstance, []string) error

type CommandManager struct {
	commands map[string]CommandHandler
	mutex    sync.RWMutex
}

func NewCommandManager() *CommandManager {
	return &CommandManager{
		commands: make(map[string]CommandHandler),
	}
}

func (cm *CommandManager) RegisterCommand(name string, handler CommandHandler) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.commands[name] = handler
}

func (cm *CommandManager) ExecuteCommand(app *AppInstance, command string, args []string) error {
	cm.mutex.RLock()
	handler, exists := cm.commands[command]
	cm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("unknown command: %s", command)
	}

	return handler(app, args)
}

func (cm *CommandManager) ListCommands() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	commands := make([]string, 0, len(cm.commands))
	for name := range cm.commands {
		commands = append(commands, name)
	}
	return commands
}

// 为 AppInstance 添加命令管理器
func (app *AppInstance) setupCommandManager() {
	// 这里可以添加一个 commandManager 字段到 AppInstance 结构体中
	// 为了简化，这里返回一个新的实例
	cm := NewCommandManager()

	// 注册默认命令（使用包装函数来匹配 CommandHandler 签名）
	cm.RegisterCommand("start", func(app *AppInstance, args []string) error {
		return app.handleStartCommand(args)
	})
	cm.RegisterCommand("stop", func(app *AppInstance, args []string) error {
		return app.handleStopCommand(args)
	})
	cm.RegisterCommand("reload", func(app *AppInstance, args []string) error {
		return app.handleReloadCommand(args)
	})

	app.commandManager = cm
}

// 默认命令处理器
func (app *AppInstance) handleStartCommand(args []string) error {
	app.logger.Info("Handling start command")
	return app.Run(args)
}

func (app *AppInstance) handleStopCommand(args []string) error {
	app.logger.Info("Handling stop command")
	return app.Stop()
}

func (app *AppInstance) handleReloadCommand(args []string) error {
	app.logger.Info("Handling reload command")
	return app.Reload()
}

// 运行模式处理
func (app *AppInstance) RunWithMode(mode AppMode, arguments []string) error {
	app.mode = mode

	switch mode {
	case AppModeStart:
		return app.Run(arguments)
	case AppModeStop:
		return app.handleStopCommand(arguments)
	case AppModeReload:
		return app.handleReloadCommand(arguments)
	case AppModeInfo, AppModeHelp:
		return app.handleVersionCommand(arguments)
	default:
		return app.RunCommand(app.flagSet.Args()[1:])
	}
}

// 解析命令行参数并确定运行模式
func (app *AppInstance) ParseArgumentsAndRun(arguments []string) error {
	if err := app.setupOptions(arguments); err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	// 根据参数确定运行模式
	mode := AppModeStart
	if app.flagSet.Lookup("v").Value.String() == "true" {
		mode = AppModeInfo
	}

	// 检查位置参数以确定命令
	args := app.flagSet.Args()
	if len(args) > 0 {
		switch args[0] {
		case "start":
			mode = AppModeStart
		case "stop":
			mode = AppModeStop
		case "reload":
			mode = AppModeReload
		case "help":
			mode = AppModeHelp
		case "version":
			mode = AppModeInfo
		case "run":
			mode = AppModeCustom
		}
	}

	return app.RunWithMode(mode, arguments)
}

func (app *AppInstance) MakeAction(callback func(action *AppActionData) error, message_data []byte, private_data interface{}) *AppActionSender {
	sender := globalAppActionSenderPool.Get().(*AppActionSender)
	sender.callback = callback
	sender.data.MessageData = message_data
	sender.data.PrivateData = private_data
	sender.data.App = app
	return sender
}

// 辅助方法：写入PID文件
func (app *AppInstance) PushAction(callback func(action *AppActionData) error, message_data []byte, private_data interface{}) error {
	sender := app.MakeAction(callback, message_data, private_data)
	if err := app.workerPool.Invoke(sender); err != nil {
		app.logger.Error("failed to invoke action: %w", err)
		return err
	}

	return nil
}

func (app *AppInstance) processAction(sender *AppActionSender) {
	err := sender.callback(&sender.data)
	if err != nil {
		app.logger.Error("Action callback error: %v", err)
	}

	sender.reset()
	globalAppActionSenderPool.Put(sender)
}

// 便利函数：创建并运行应用
func RunApp(arguments []string) error {
	app := CreateAppInstance().(*AppInstance)
	return app.ParseArgumentsAndRun(arguments)
}

// AppAction对象
type AppActionData struct {
	App         AppImpl
	MessageData []byte
	PrivateData interface{}
}

type AppActionSender struct {
	callback func(action *AppActionData) error
	data     AppActionData
}

func (s *AppActionSender) reset() {
	s.callback = nil
	s.data.App = nil
	s.data.MessageData = nil
	s.data.PrivateData = nil
}

var globalAppActionSenderPool = sync.Pool{
	New: func() interface{} {
		return new(AppActionSender)
	},
}
