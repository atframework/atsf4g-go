// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform_type

type userBasicProfileImpl struct {
	NickName string // 用户昵称
	Avatar   string // 用户头像URL
	OpenId   string // 应用用户唯一ID
	UnionId  string // 厂商用户唯一ID，只有部分平台提供
}

type UserBasicProfile interface {
	GetNickName() string // 用户昵称
	GetAvatar() string   // 用户头像URL
	GetOpenId() string   // 应用用户唯一ID
	GetUnionId() string  // 厂商用户唯一ID，只有部分平台提供
}

func (p *userBasicProfileImpl) GetNickName() string {
	if p == nil {
		return ""
	}
	return p.NickName
}

func (p *userBasicProfileImpl) GetAvatar() string {
	if p == nil {
		return ""
	}
	return p.Avatar
}

func (p *userBasicProfileImpl) GetOpenId() string {
	if p == nil {
		return ""
	}
	return p.OpenId
}

func (p *userBasicProfileImpl) GetUnionId() string {
	if p == nil {
		return ""
	}
	return p.UnionId
}

func MakeUserBasicProfile(nickName, avatar, openId, unionId string) UserBasicProfile {
	return &userBasicProfileImpl{
		NickName: nickName,
		Avatar:   avatar,
		OpenId:   openId,
		UnionId:  unionId,
	}
}
