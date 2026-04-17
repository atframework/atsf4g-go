// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform_type

import (
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
)

type openPlatformUserKeyImpl struct {
	OpenId string // 平台用户唯一ID
}

type OpenPlatformUserKey interface {
	GetOpenId() string // 平台用户唯一ID
}

func (k *openPlatformUserKeyImpl) GetOpenId() string {
	if k == nil {
		return ""
	}
	return k.OpenId
}

func MakeOpenPlatformUserKey(openId string) OpenPlatformUserKey {
	return &openPlatformUserKeyImpl{
		OpenId: openId,
	}
}

type userAuthDataImpl struct {
	userKey             OpenPlatformUserKey
	accountData         *public_protocol_pbdesc.DAccountData
	platformParameter   map[string]string // 平台相关参数
	platformPrivateData interface{}       // 平台私有数据
}

type UserAuthData interface {
	GetAccountData() *public_protocol_pbdesc.DAccountData // 账户信息
	GetUserKey() OpenPlatformUserKey                      // 用户Key
	GetOpenId() string                                    // 应用用户唯一ID
	GetAccessToken() string                               // 访问令牌
	GetPlatformParameter() map[string]string              // 平台相关参数
	GetPlatformPrivateData() interface{}                  // 平台私有数据
	SetPlatformPrivateData(data interface{})              // 设置平台私有数据
}

func (d *userAuthDataImpl) GetAccountData() *public_protocol_pbdesc.DAccountData {
	if d == nil {
		return nil
	}
	return d.accountData
}

func (d *userAuthDataImpl) GetUserKey() OpenPlatformUserKey {
	if d == nil {
		return nil
	}
	return d.userKey
}

func (d *userAuthDataImpl) GetOpenId() string {
	if d == nil {
		return ""
	}

	return d.userKey.GetOpenId()
}

func (d *userAuthDataImpl) GetAccessToken() string {
	if d == nil {
		return ""
	}

	if d.accountData == nil {
		return ""
	}

	return d.accountData.GetAccess()
}

func (d *userAuthDataImpl) GetPlatformParameter() map[string]string {
	if d == nil {
		return nil
	}

	return d.platformParameter
}

func (d *userAuthDataImpl) GetPlatformPrivateData() interface{} {
	if d == nil {
		return nil
	}

	return d.platformPrivateData
}

func (d *userAuthDataImpl) SetPlatformPrivateData(data interface{}) {
	if d == nil {
		return
	}

	d.platformPrivateData = data
}

func MakeUserAuthData(userKey OpenPlatformUserKey,
	accountData *public_protocol_pbdesc.DAccountData,
	platformParameter map[string]string,
) UserAuthData {
	return &userAuthDataImpl{
		userKey:           userKey,
		accountData:       accountData,
		platformParameter: platformParameter,
	}
}

func GetUserAuthDataPlatformPrivate[T any](userAuth UserAuthData) T {
	if userAuth == nil {
		var zero T
		return zero
	}

	data, ok := userAuth.GetPlatformPrivateData().(T)
	if !ok {
		var zero T
		return zero
	}

	return data
}

func SetUserAuthDataPlatformPrivate[T any](userAuth UserAuthData, data T) {
	if userAuth == nil {
		return
	}

	userAuth.SetPlatformPrivateData(data)
}
