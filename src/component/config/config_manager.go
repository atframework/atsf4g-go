package atframework_component_config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"reflect"
	"sync"
	"sync/atomic"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"
)

var configManagerModuleReflectType reflect.Type

func init() {
	configManagerModuleReflectType = lu.GetStaticReflectType[ConfigManagerModule]()
}

type ConfigManager struct {
	currentConfigGroup        *generate_config.ConfigGroup
	loadingConfigGroup        *generate_config.ConfigGroup
	currentConfigGroupRwMutex sync.RWMutex

	init         bool
	reloading    atomic.Bool
	reloadFinish atomic.Bool

	configFile           string
	overwriteResourceDir string
	logger               *slog.Logger
}

// 管理所有配置
var globalConfigManagerInst = ConfigManager{}

func (configManagerInst *ConfigManager) SetConfigFile(path string) {
	configManagerInst.configFile = path
}

func (configManagerInst *ConfigManager) SetResourceDir(path string) {
	configManagerInst.overwriteResourceDir = path
}

func (configManagerInst *ConfigManager) GetLogger() *slog.Logger {
	if configManagerInst.logger == nil {
		return slog.Default()
	}

	return configManagerInst.logger
}

func (configManagerInst *ConfigManager) Init(parent context.Context) error {
	configManagerInst.currentConfigGroup = new(generate_config.ConfigGroup)
	configManagerInst.currentConfigGroup.ExcelResourceDir = configManagerInst.overwriteResourceDir
	return configManagerInst.loadImpl(configManagerInst.currentConfigGroup)
}

// 同步接口
func (configManagerInst *ConfigManager) Reload() error {
	if !configManagerInst.init {
		return nil
	}
	return configManagerInst.reloadImpl(nil)
}

// 异步接口
func (configManagerInst *ConfigManager) AsyncReload(resultChan chan error) {
	go configManagerInst.reloadImpl(resultChan)
}

func (configManagerInst *ConfigManager) reloadImpl(resultChan chan error) error {
	// 设置标识
	if configManagerInst.reloading.Load() {
		if resultChan != nil {
			resultChan <- fmt.Errorf("Reload not Finish")
		}
		configManagerInst.GetLogger().Info("Reload not Finish")
		return fmt.Errorf("Reload not Finish")
	}

	configManagerInst.reloading.Store(true)
	configManagerInst.reloadFinish.Store(false)

	newConfigGroup := new(generate_config.ConfigGroup)
	newConfigGroup.ExcelResourceDir = configManagerInst.overwriteResourceDir
	err := configManagerInst.loadImpl(newConfigGroup)
	defer func() {
		if resultChan != nil {
			resultChan <- err
			close(resultChan)
			resultChan = nil
		}
		configManagerInst.reloading.Store(false)
	}()

	if err != nil {
		return err
	}

	configManagerInst.loadingConfigGroup = newConfigGroup
	configManagerInst.reloadFinish.Store(true)
	configManagerInst.init = true
	return nil
}

func (configManagerInst *ConfigManager) loadImpl(loadConfigGroup *generate_config.ConfigGroup) error {
	// 加载配置逻辑
	configManagerInst.GetLogger().Info("Excel Loading Begin")
	var callback ExcelConfigCallback
	err := loadConfigGroup.Init(configManagerInst.configFile, callback)
	if err != nil {
		return err
	}
	configManagerInst.GetLogger().Info("Excel Loading End")
	return nil
}

func (configManagerInst *ConfigManager) Tick(parent context.Context) bool {
	configManagerInst.checkReloadFinish()
	return true
}

func (configManagerInst *ConfigManager) checkReloadFinish() {
	if configManagerInst.reloadFinish.Load() {
		// 替换
		configManagerInst.currentConfigGroupRwMutex.Lock()
		configManagerInst.currentConfigGroup = configManagerInst.loadingConfigGroup
		configManagerInst.currentConfigGroupRwMutex.Unlock()
		// 标识完成
		configManagerInst.reloadFinish.Store(false)
	}
}

func (configManagerInst *ConfigManager) GetCurrentConfigGroup() *generate_config.ConfigGroup {
	// 加写锁
	configManagerInst.currentConfigGroupRwMutex.RLock()
	defer configManagerInst.currentConfigGroupRwMutex.RUnlock()

	return configManagerInst.currentConfigGroup
}

func (configManagerInst *ConfigManager) GetZoneId() uint32 {
	return configManagerInst.GetCurrentConfigGroup().GetServerConfig().GetZoneId()
}

func (configManagerInst *ConfigManager) GetWorldId() uint32 {
	return configManagerInst.GetCurrentConfigGroup().GetServerConfig().GetWorldId()
}

func (configManagerInst *ConfigManager) GetLogicId() uint32 {
	return configManagerInst.GetCurrentConfigGroup().GetServerConfig().GetLogicId()
}

func GetConfigManager() *ConfigManager {
	return &globalConfigManagerInst
}

type ExcelConfigCallback struct{}

func (callback ExcelConfigCallback) LoadFile(prefixPath string, pbinName string) ([]byte, error) {
	filePath := path.Join(prefixPath, pbinName)

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		callback.GetLogger().Error("File Not Found", "filePath", filePath)
		return nil, fmt.Errorf("file not found %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		callback.GetLogger().Error("File Read Failed", "error", err)
		return nil, err
	}
	return content, nil
}

func (callback ExcelConfigCallback) GetLogger() *slog.Logger {
	return GetConfigManager().GetLogger()
}

func (callback ExcelConfigCallback) OnLoaded(config_group *generate_config.ConfigGroup) error {
	return ExcelConfigCallbackOnLoad(config_group, callback.GetLogger())
}

type ConfigManagerModule struct {
	libatapp.AppModuleBase

	SharedConfigManager *ConfigManager
}

func CreateConfigManagerModule(app libatapp.AppImpl) *ConfigManagerModule {
	return &ConfigManagerModule{
		AppModuleBase:       libatapp.CreateAppModuleBase(app),
		SharedConfigManager: GetConfigManager(),
	}
}

func (m *ConfigManagerModule) Init(parent context.Context) error {
	GetConfigManager().SetConfigFile(m.GetApp().GetConfigFile())
	m.SharedConfigManager.logger = m.GetApp().GetDefaultLogger()
	return m.SharedConfigManager.Init(parent)
}

func (m *ConfigManagerModule) Name() string {
	return "ConfigManagerModule"
}

func (m *ConfigManagerModule) GetReflectType() reflect.Type {
	return configManagerModuleReflectType
}

// 同步接口
func (m *ConfigManagerModule) Reload() error {
	return m.SharedConfigManager.Reload()
}

func (m *ConfigManagerModule) Tick(parent context.Context) bool {
	return m.SharedConfigManager.Tick(parent)
}
