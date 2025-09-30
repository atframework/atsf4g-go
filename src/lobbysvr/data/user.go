package lobbysvr_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type Result = cd.DispatcherErrorResult

type User struct {
	uc.UserCache
}

func (u *User) Init() {
	u.UserCache.Init(u)

	// TODO: 初始化各类Manager
}

func createUser(zoneId uint32, userId uint64, openId string) *User {
	return &User{
		UserCache: uc.CreateUserCache(zoneId, userId, openId),
	}
}

func init() {
	uc.GlobalUserManager.SetCreateUserCallback(func(zoneId uint32, userId uint64, openId string) uc.UserImpl {
		ret := createUser(zoneId, userId, openId)
		return ret
	})
}
