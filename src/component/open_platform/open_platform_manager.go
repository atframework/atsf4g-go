// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	"context"
	"fmt"
	"strings"
	"sync"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	log "github.com/atframework/atframe-utils-go/log"
	libatapp "github.com/atframework/libatapp-go"

	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	op_channel_none "github.com/atframework/atsf4g-go/component/open_platform/channel/none"
	op_channel_taptap "github.com/atframework/atsf4g-go/component/open_platform/channel/taptap"
	op_types "github.com/atframework/atsf4g-go/component/open_platform/type"
)

type openPlatformManagerImpl struct {
	libatapp.AppModuleBase

	configureLock sync.RWMutex
	configurePath string
	configureData *private_protocol_config.Readonly_OpenPlatformManagerCfg
}

func CreateOpenPlatformManager(owner libatapp.AppImpl, configurePath string) op_types.OpenPlatformManager {
	return &openPlatformManagerImpl{
		AppModuleBase: libatapp.CreateAppModuleBase(owner),
		configurePath: configurePath,
	}
}

func (m *openPlatformManagerImpl) GetConfigure() *private_protocol_config.Readonly_OpenPlatformManagerCfg {
	if m == nil {
		return nil
	}

	m.configureLock.RLock()
	defer m.configureLock.RUnlock()
	return m.configureData
}

func (m *openPlatformManagerImpl) Name() string {
	return "OpenPlatformManager"
}

func (m *openPlatformManagerImpl) Init(parent context.Context) error {
	return nil
}

func (m *openPlatformManagerImpl) GetLogger() *log.Logger {
	app := m.GetApp()
	if lu.IsNil(app) {
		return nil
	}

	return app.GetDefaultLogger()
}

func (m *openPlatformManagerImpl) Reload() error {
	if m == nil {
		return fmt.Errorf("OpenPlatformManager is nil")
	}

	err := m.AppModuleBase.Reload()
	if err != nil {
		return err
	}

	configureData := &private_protocol_config.OpenPlatformManagerCfg{}

	loadErr := m.GetApp().LoadConfigByPath(configureData, m.configurePath,
		strings.ToUpper(strings.ReplaceAll(m.configurePath, ".", "_")), nil, "")
	if loadErr != nil {
		m.GetLogger().LogError("Failed to load open platform manager config", "error", loadErr)
		return loadErr
	}

	m.configureLock.Lock()
	m.configureData = configureData.ToReadonly()
	m.configureLock.Unlock()
	return nil
}

func (m *openPlatformManagerImpl) CreateChannelDelegate(channel_type op_types.OpenPlatformAccountType) op_types.OpenPlatformChannelDelegate {
	switch channel_type {
	case public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INTERNAL:
		return &op_channel_none.NoneChannelDelegate{}
	case public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_TAPTAP:
		return &op_channel_taptap.TapTapChannelDelegate{}
	default:
		m.GetLogger().LogError("Unsupported channel type", "channel_type", channel_type)
		return nil
	}
}
