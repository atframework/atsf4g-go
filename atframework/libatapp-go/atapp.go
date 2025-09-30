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
	"slices"
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
	StopInterval     time.Duration
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
	RunOnce(tickTimer *time.Ticker) error
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

	AddModule(module AppModuleImpl) (int, error)

	GetModule(moduleId int) AppModuleImpl

	// 消息相关
	SendMessage(targetId uint64, msgType int32, data []byte) error
	SendMessageByName(targetName string, msgType int32, data []byte) error

	// 事件相关
	SetEventHandler(eventType string, handler EventHandler)
	TriggerEvent(eventType string, args *AppActionSender) int

	// 自定义Action
	PushAction(callback func(action *AppActionData) error, message_data []byte, private_data interface{}) error

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

	// 生命周期管理
	GetAppContext() context.Context

	// Logger
	GetLogger() *slog.Logger
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
	modules []AppModuleImpl

	// 生命周期控制
	appContext    context.Context
	stopAppHandle context.CancelFunc

	// 定时器和事件
	tickTimer     *time.Ticker
	stopTimepoint time.Time
	stopTimeout   time.Time

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
		mode:          AppModeHelp,
		stopTimepoint: time.Time{},
		stopTimeout:   time.Time{},
		eventHandlers: make(map[string]EventHandler),
		signalChan:    make(chan os.Signal, 1),
		logger:        slog.Default(),
	}

	ret.flagSet = flag.NewFlagSet(
		fmt.Sprintf("%s [options...] <start|stop|reload|run> [<custom command> [command args...]]", filepath.Base(os.Args[0])), flag.ContinueOnError)
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

	ret.appContext, ret.stopAppHandle = context.WithCancel(context.Background())

	// 设置默认配置
	ret.config.TickInterval = 8 * time.Millisecond
	ret.config.TickRoundTimeout = 128 * time.Millisecond
	ret.config.StopTimeout = 30 * time.Second
	ret.config.StopInterval = 100 * time.Millisecond
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

	// TODO: 内置公共引用层模块

	return ret
}

func (app *AppInstance) destroy() {
	if !app.IsClosed() {
		app.close()
	}

	if app.IsInited() {
		app.cleanup()
	}

	for _, m := range slices.Backward(app.modules) {
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
func checkFlag(flags uint64, checked AppFlag) bool {
	return flags&uint64(checked) != 0
}

func (app *AppInstance) getFlags() uint64 {
	return atomic.LoadUint64(&app.flags)
}

func (app *AppInstance) CheckFlag(flag AppFlag) bool {
	return checkFlag(app.getFlags(), flag)
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

func (app *AppInstance) AddModule(module AppModuleImpl) (int, error) {
	flags := app.getFlags()
	if checkFlag(flags, AppFlagInitialized) || checkFlag(flags, AppFlagInitializing) {
		return 0, fmt.Errorf("cannot add module when app is initializing or initialized")
	}

	app.modules = append(app.modules, module)
	module.OnBind()
	return len(app.modules), nil
}

func (app *AppInstance) GetModule(moduleId int) AppModuleImpl {
	if moduleId <= 0 || moduleId > len(app.modules) {
		return nil
	}

	return app.modules[moduleId-1]
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
		if app.mode == AppModeStart {
			app.writeStartupErrorFile(err)
		}
		return fmt.Errorf("setup startup log failed: %w", err)
	}

	// 设置信号处理
	if app.mode != AppModeCustom && app.mode != AppModeStop && app.mode != AppModeReload {
		if err := app.setupSignal(); err != nil {
			if app.mode == AppModeStart {
				app.writeStartupErrorFile(err)
			}
			return fmt.Errorf("setup signal failed: %w", err)
		}
	}

	// 加载配置
	if err := app.LoadConfig(app.config.ConfigFile); err != nil {
		if app.mode == AppModeStart {
			app.writeStartupErrorFile(err)
		}
		return fmt.Errorf("load config failed: %w", err)
	}

	if app.mode == AppModeCustom || app.mode == AppModeStop || app.mode == AppModeReload {
		return app.sendLastCommand()
	}

	// 初始化日志
	if err := app.setupLog(); err != nil {
		if app.mode == AppModeStart {
			app.writeStartupErrorFile(err)
		}
		return fmt.Errorf("setup log failed: %w", err)
	}

	// 设置定时器
	if err := app.setupTickTimer(); err != nil {
		if app.mode == AppModeStart {
			app.writeStartupErrorFile(err)
		}
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
		if app.mode == AppModeStart {
			app.writeStartupErrorFile(err)
		}
		return err
	}

	initContext, initCancel := context.WithTimeout(app.appContext, app.config.InitTimeout)
	defer initCancel()

	// 初始化所有模块
	// Setup phase
	for _, m := range app.modules {
		if initContext.Err() != nil {
			break
		}
		if err := m.Setup(initContext); err != nil {
			if app.mode == AppModeStart {
				app.writeStartupErrorFile(err)
			}
			return fmt.Errorf("module setup failed: %w", err)
		}

		m.Enable()
	}

	// Setup log phase
	for _, m := range app.modules {
		if initContext.Err() != nil {
			break
		}
		if err := m.SetupLog(initContext); err != nil {
			if app.mode == AppModeStart {
				app.writeStartupErrorFile(err)
			}
			return fmt.Errorf("module setup log failed: %w", err)
		}
	}

	// Start reload phase
	for _, m := range app.modules {
		if initContext.Err() != nil {
			break
		}
		if err := m.Reload(); err != nil {
			if app.mode == AppModeStart {
				app.writeStartupErrorFile(err)
			}
			return fmt.Errorf("module reload failed: %w", err)
		}
	}

	// Init phase
	for _, m := range app.modules {
		if initContext.Err() != nil {
			break
		}
		if err := m.Init(initContext); err != nil {
			if app.mode == AppModeStart {
				app.writeStartupErrorFile(err)
			}
			return fmt.Errorf("module init failed: %w", err)
		}

		m.Active()
	}

	maybeErr := initContext.Err()
	if maybeErr == nil {
		maybeErr = app.appContext.Err()
	}

	if maybeErr == nil {
		// TODO: evt_on_all_module_inited_(*this);

		app.SetFlag(AppFlagRunning, true)
		app.SetFlag(AppFlagInitialized, true)
		app.SetFlag(AppFlagStopped, false)
		app.SetFlag(AppFlagStopping, false)

		if app.mode == AppModeStart {
			app.writePidFile()
			app.cleanupStartupErrorFile()
		}

		// Ready phase
		for _, m := range app.modules {
			m.Ready()
		}
	} else {
		// 失败处理
		app.Stop()

		for app.IsInited() && !app.IsClosed() {
			app.internalRunOnce(app.tickTimer)
		}

		app.writeStartupErrorFile(maybeErr)
	}

	return maybeErr
}

func (app *AppInstance) internalRunOnce(tickTimer *time.Ticker) error {
	if !app.IsInited() && !app.CheckFlag(AppFlagInitializing) {
		return fmt.Errorf("app is not initialized")
	}

	if app.CheckFlag(AppFlagInCallback) {
		return nil
	}

	if app.mode != AppModeCustom && app.mode == AppModeStart {
		return nil
	}

	select {
	case <-app.appContext.Done():
		app.logger.Info("Start to stop...")
	case sig := <-app.signalChan:
		if sig == syscall.SIGTERM || sig == syscall.SIGQUIT {
			app.logger.Info("Received signal, stopping...", slog.Any("signal", sig))
			app.Stop()
		}
	case <-tickTimer.C:
		if err := app.tick(); err != nil {
			app.logger.Error("Tick error", slog.Any("err", err))
		}
	}

	flags := app.getFlags()
	if checkFlag(flags, AppFlagStopping) && !checkFlag(flags, AppFlagStopped) {
		now := time.Now()
		if now.After(app.stopTimeout) {
			app.SetFlag(AppFlagTimeout, true)
		}
		forceTimeout := checkFlag(flags, AppFlagTimeout)
		if now.After(app.stopTimepoint) || forceTimeout {
			app.stopTimepoint = now.Add(app.config.StopInterval)
			app.closeAllModules(forceTimeout)
		}
	}

	if checkFlag(flags, AppFlagStopped) && !checkFlag(flags, AppFlagInitialized) {
		app.cleanup()
	}

	return nil
}

func (app *AppInstance) RunOnce(tickTimer *time.Ticker) error {
	return app.internalRunOnce(tickTimer)
}

func (app *AppInstance) Run(arguments []string) error {
	if !app.IsInited() {
		if err := app.Init(arguments); err != nil {
			return err
		}
	}

	// 主事件循环
	for app.IsInited() && !app.IsClosed() {
		app.internalRunOnce(app.tickTimer)
	}

	return nil
}

func (app *AppInstance) closeAllModules(forceTimeout bool) (bool, error) {
	// 设置停止标志
	allClosed := true
	var err error = nil

	// all modules stop
	for _, m := range slices.Backward(app.modules) {
		if !m.IsActived() {
			continue
		}

		moduleClosed, err := m.Stop()
		if err != nil {
			app.logger.Error("Module %s stop failed: %v", m.Name(), err)
			m.Unactive()
		} else if !moduleClosed {
			if forceTimeout {
				m.Timeout()
				m.Unactive()
			} else {
				allClosed = false
			}
		} else {
			m.Unactive()
		}
	}

	if allClosed {
		app.SetFlag(AppFlagStopped, true)
	}

	return allClosed, err
}

func (app *AppInstance) close() (bool, error) {
	allClosed := true
	var err error = nil

	// all modules stop
	for _, m := range slices.Backward(app.modules) {
		if !m.IsActived() {
			continue
		}

		moduleClosed, err := m.Stop()
		if err != nil {
			app.logger.Error("Module %s stop failed: %v", m.Name(), err)
			m.Unactive()
		} else if !moduleClosed {
			allClosed = false
		} else {
			m.Unactive()
		}
	}

	return allClosed, err
}

func (app *AppInstance) cleanup() error {
	// all modules cleanup
	for _, m := range slices.Backward(app.modules) {
		if m.IsEnabled() {
			m.Cleanup()
			m.Disable()
		}
	}

	// TODO: cleanup event

	// close tick timer
	if app.tickTimer != nil {
		app.tickTimer.Stop()
	}

	// cleanup pidfile
	app.cleanupPidFile()

	app.SetFlag(AppFlagRunning, false)
	app.SetFlag(AppFlagInitialized, false)

	// TODO: finally callback
	return nil
}

func (app *AppInstance) RunCommand(arguments []string) error {
	// 分发命令处理逻辑
	command := arguments[0]
	args := arguments[1:]

	app.commandManager.ExecuteCommand(app, command, args)
	return nil
}

func (app *AppInstance) sendLastCommand() error {
	if len(app.lastCommand) == 0 {
		app.GetLogger().Error("No command to send")
		return fmt.Errorf("no command to send")
	}
	// TODO: 发送远程指令
	return nil
}

func (app *AppInstance) Stop() error {
	if app.IsClosing() {
		return nil
	}

	app.stopTimeout = time.Now().Add(app.config.StopTimeout)
	app.SetFlag(AppFlagStopping, true)
	app.stopAppHandle()
	return nil
}

func (app *AppInstance) Reload() error {
	// 重新加载配置
	if err := app.LoadConfig(app.config.ConfigFile); err != nil {
		return fmt.Errorf("reload config failed: %w", err)
	}

	// 重新加载所有模块
	for _, m := range app.modules {
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
		app.logger.Info("Loading config from", "configFile", configFile)
	}
	return nil
}

// 消息相关
func (app *AppInstance) SendMessage(targetId uint64, msgType int32, data []byte) error {
	// TODO: 实现消息发送逻辑
	app.logger.Debug("Sending message",
		"targetId", targetId,
		"type", msgType,
		"size", len(data),
	)
	return nil
}

func (app *AppInstance) SendMessageByName(targetName string, msgType int32, data []byte) error {
	// TODO: 实现按名称发送消息逻辑
	app.logger.Debug("Sending message",
		"targetName", targetName,
		"type", msgType,
		"size", len(data),
	)
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

func (app *AppInstance) GetAppContext() context.Context {
	return app.appContext
}

func (app *AppInstance) GetLogger() *slog.Logger {
	return app.logger
}

// 内部辅助方法
func (app *AppInstance) setupOptions(arguments []string) error {
	// 在测试环境中，如果 arguments 为 nil，跳过参数解析
	if arguments == nil {
		return nil
	}

	// 检查是否在测试环境中
	if err := app.flagSet.Parse(arguments); err != nil {
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
		case "help":
			app.mode = AppModeHelp
		case "version":
			app.mode = AppModeInfo
		default:
			return fmt.Errorf("unknown command: %s", args[0])
		}
	}

	return nil
}

func (app *AppInstance) setupSignal() error {
	signal.Notify(app.signalChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGPIPE, syscall.SIGTERM, syscall.SIGQUIT)
	return nil
}

func (app *AppInstance) setupStartupLog() error {
	// TODO: 根据配置设置启动流程日志
	if len(app.config.StartupLog) > 0 {
		for _, logFile := range app.config.StartupLog {
			app.logger.Info("Setting up startup log", "file", logFile)
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
		app.tickTimer = time.NewTicker(app.config.TickInterval)
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
	tickContext, cancel := context.WithCancel(app.appContext)
	defer cancel()

	// 调用模块的tick方法
	for _, m := range app.modules {
		if m.IsActived() {
			m.Tick(tickContext)
		}
	}

	return nil
}

// 辅助方法：写入PID文件
func (app *AppInstance) writePidFile() error {
	if app.config.PidFile == "" {
		return nil
	}

	pid := os.Getpid()
	pidData := fmt.Sprintf("%d\n", pid)

	return os.WriteFile(app.config.PidFile, []byte(pidData), 0o644)
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
	os.WriteFile(app.config.StartupErrorFile, []byte(pidData), 0o644)
}

// 辅助方法：清理启动失败标记文件
func (app *AppInstance) cleanupStartupErrorFile() error {
	if app.config.StartupErrorFile == "" {
		return nil
	}

	return os.Remove(app.config.StartupErrorFile)
}

// 命令处理相关
type CommandHandler func(*AppInstance, string, []string) error

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
		handler, exists = cm.commands["@OnError"]
		if exists {
			return handler(app, command, args)
		} else {
			app.logger.Error("Error command executed: %s %v", command, args)
			return nil
		}
	}

	return handler(app, command, args)
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
	cm.RegisterCommand("start", func(app *AppInstance, _command string, args []string) error {
		return app.handleStartCommand(args)
	})
	cm.RegisterCommand("stop", func(app *AppInstance, _command string, args []string) error {
		return app.handleStopCommand(args)
	})
	cm.RegisterCommand("reload", func(app *AppInstance, _command string, args []string) error {
		return app.handleReloadCommand(args)
	})
	cm.RegisterCommand("@OnError", func(app *AppInstance, command string, args []string) error {
		app.logger.Error("Error command executed: %s %v", command, args)
		return nil
	})

	app.commandManager = cm
}

// 默认命令处理器
func (app *AppInstance) handleStartCommand(_args []string) error {
	app.logger.Info("======================== App start ========================")
	return nil
}

func (app *AppInstance) handleStopCommand(_args []string) error {
	app.logger.Info("======================== App received stop command ========================")
	return app.Stop()
}

func (app *AppInstance) handleReloadCommand(_args []string) error {
	app.logger.Info("======================== App received reload command ========================")
	return app.Reload()
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
		app.logger.Error("failed to invoke action", "err", err)
		return err
	}

	return nil
}

func (app *AppInstance) processAction(sender *AppActionSender) {
	err := sender.callback(&sender.data)
	if err != nil {
		app.logger.Error("Action callback error", slog.Any("err", err))
	}

	sender.reset()
	globalAppActionSenderPool.Put(sender)
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
