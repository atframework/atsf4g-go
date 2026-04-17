// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform_type

import (
	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	libatapp "github.com/atframework/libatapp-go"
)

type OpenPlatformAccountType = public_protocol_pbdesc.EnAccountTypeID

type OpenPlatformManager interface {
	libatapp.AppModuleImpl

	GetConfigure() *private_protocol_config.Readonly_OpenPlatformManagerCfg
	CreateChannelDelegate(channel_type OpenPlatformAccountType) OpenPlatformChannelDelegate
}
