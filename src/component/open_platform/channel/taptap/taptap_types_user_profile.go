// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

type TapTapUserProfile struct {
	OpenID  string `json:"openid"`  // 授权用户唯一标识，每个玩家在每个游戏中的 openid 都是不一样的，同一游戏获取同一玩家的 openid 总是相同
	Avatar  string `json:"avatar"`  // 头像
	Name    string `json:"name"`    // 昵称
	UnionID string `json:"unionid"` // 授权用户唯一标识，一个玩家在一个厂商的所有游戏中 unionid 都是一样的，不同厂商 unionid 不同
}
