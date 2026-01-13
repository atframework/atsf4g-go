package atframework_component_config

import (
	"context"
	"fmt"
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

	originConfigData        interface{}
	overwriteResourceDir    string
	serverConfigureLoadFunc generate_config.ServerConfigureLoadFuncType
	logger                  *libatapp.Logger
	app                     libatapp.AppImpl
}

// 管理所有配置
var globalConfigManagerInst = ConfigManager{}

func (configManagerInst *ConfigManager) SetConfigOriginData(data interface{}) {
	configManagerInst.originConfigData = data
}

func (configManagerInst *ConfigManager) SetResourceDir(path string) {
	configManagerInst.overwriteResourceDir = path
}

func (configManagerInst *ConfigManager) SetServerConfigureLoadFunc(serverConfigureLoadFunc generate_config.ServerConfigureLoadFuncType) {
	configManagerInst.serverConfigureLoadFunc = serverConfigureLoadFunc
}

func GetServerConfig[T any](configGroup *generate_config.ConfigGroup) T {
	if configGroup == nil {
		var zero T
		return zero
	}
	return configGroup.GetServerConfig().(T)
}

func (configManagerInst *ConfigManager) GetLogger() *libatapp.Logger {
	if configManagerInst.logger == nil {
		return nil
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
		configManagerInst.GetLogger().LogInfo("Reload not Finish")
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
	configManagerInst.GetLogger().LogInfo("Excel Loading Begin")
	var callback ExcelConfigCallback
	err := loadConfigGroup.Init(configManagerInst.originConfigData, callback, configManagerInst.serverConfigureLoadFunc)
	if err != nil {
		return err
	}
	configManagerInst.GetLogger().LogInfo("Excel Loading End")
	return nil
}

func (configManagerInst *ConfigManager) Tick(parent context.Context) bool {
	ExcelConfigCallbackRebuild(configManagerInst.GetCurrentConfigGroup(), configManagerInst.GetLogger())
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
	return configManagerInst.GetCurrentConfigGroup().GetSectionConfig().GetZoneId()
}

func (configManagerInst *ConfigManager) GetWorldId() uint32 {
	return configManagerInst.GetCurrentConfigGroup().GetSectionConfig().GetWorldId()
}

func (configManagerInst *ConfigManager) GetLogicId() uint32 {
	return configManagerInst.GetCurrentConfigGroup().GetSectionConfig().GetLogicId()
}

func (configManagerInst *ConfigManager) GetApp() libatapp.AppImpl {
	return configManagerInst.app
}

func GetConfigManager() *ConfigManager {
	return &globalConfigManagerInst
}

type ExcelConfigCallback struct{}

func (callback ExcelConfigCallback) LoadFile(prefixPath string, pbinName string) ([]byte, error) {
	filePath := path.Join(prefixPath, pbinName)

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		callback.GetLogger().LogError("File Not Found", "filePath", filePath)
		return nil, fmt.Errorf("file not found %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		callback.GetLogger().LogError("File Read Failed", "error", err)
		return nil, err
	}
	return content, nil
}

func (callback ExcelConfigCallback) GetLogger() *libatapp.Logger {
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
	GetConfigManager().SetConfigOriginData(m.GetApp().GetConfig().ConfigOriginData)
	m.SharedConfigManager.logger = m.GetApp().GetDefaultLogger()
	m.SharedConfigManager.app = m.GetApp()
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
