// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	op_types "github.com/atframework/atsf4g-go/component/open_platform/type"
)

type (
	OpenPlatformManager         = op_types.OpenPlatformManager
	OpenPlatformAccountType     = op_types.OpenPlatformAccountType
	OpenPlatformChannelDelegate = op_types.OpenPlatformChannelDelegate
	OpenPlatformUserKey         = op_types.OpenPlatformUserKey
	OpenPlatformRpcError        = op_types.OpenPlatformRpcError
	UserAuthData                = op_types.UserAuthData
	UserBasicProfile            = op_types.UserBasicProfile
)

func MakeOpenPlatformUserKey(openId string) OpenPlatformUserKey {
	return op_types.MakeOpenPlatformUserKey(openId)
}
