package atframework_component_config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"
	"sync/atomic"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"

	libatapp "github.com/atframework/libatapp-go"
)

type ConfigManager struct {
	libatapp.AppModuleBase

	currentConfigGroup        *generate_config.ConfigGroup
	loadingConfigGroup        *generate_config.ConfigGroup
	currentConfigGroupRwMutex sync.RWMutex

	init         bool
	reloading    atomic.Bool
	reloadFinish atomic.Bool
}

// 管理所有配置
var globalConfigManagerInst ConfigManager

func (configManagerInst *ConfigManager) Init(parent context.Context) error {
	configManagerInst.currentConfigGroup = new(generate_config.ConfigGroup)
	return configManagerInst.loadImpl(configManagerInst.currentConfigGroup)
}

func (configManagerInst *ConfigManager) Name() string {
	return "ConfigManager"
}

// 同步接口
func (configManagerInst *ConfigManager) Reload() error {
	if configManagerInst.init == false {
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
		configManagerInst.GetApp().GetDefaultLogger().Info("Reload not Finish")
		return fmt.Errorf("Reload not Finish")
	}

	configManagerInst.reloading.Store(true)
	configManagerInst.reloadFinish.Store(false)

	newConfigGroup := new(generate_config.ConfigGroup)
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
	configManagerInst.GetApp().GetDefaultLogger().Info("Excel Loading Begin")
	var callback ExcelConfigCallback
	err := loadConfigGroup.Init(callback)
	if err != nil {
		return err
	}
	configManagerInst.GetApp().GetDefaultLogger().Info("Excel Loading End")
	return nil
}

func (configManagerInst *ConfigManager) Tick(arent context.Context) bool {
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

func GetConfigManager() *ConfigManager {
	return &globalConfigManagerInst
}

type ExcelConfigCallback struct{}

func (callback ExcelConfigCallback) LoadFile(pbinName string) ([]byte, error) {
	filePath := path.Join("..", "..", "resource", "excel", pbinName)

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
	return GetConfigManager().GetApp().GetDefaultLogger()
}

func (callback ExcelConfigCallback) OnLoaded(config_group *generate_config.ConfigGroup) error {
	return ExcelConfigCallbackOnLoad(config_group)
}
