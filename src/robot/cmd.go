package main

import (
	"fmt"
)

func init() {
	RegisterCommand([]string{"user", "login"}, LoginCmd, "<openid>")
	RegisterCommand([]string{"user", "logout"}, Logout, "")
	RegisterCommand([]string{"user", "getInfo"}, GetInfoCmd, "")
	RegisterCommand([]string{"misc", "gm"}, GMCmd, "<args...>")
}

func Logout(cmd []string) string {
	GetCurrentUser().Logout()
	CurrentUser = nil
	fmt.Println("user logout")
	return ""
}

func LoginCmd(cmd []string) string {
	if len(cmd) < 1 {
		return "Need OpenId"
	}
	// 创建角色
	CreateUser(cmd[0])

	// 发送登录请求
	err := LoginAuthRpc(GetCurrentUser())
	if err != nil {
		return err.Error()
	}

	err = LoginRpc(GetCurrentUser())
	if err != nil {
		return err.Error()
	}

	// 创建Ping流程
	GetCurrentUser().CheckPingTask()
	return ""
}

func GetInfoCmd(cmd []string) string {
	// 发送登录请求
	err := GetInfoRpc(GetCurrentUser())
	if err != nil {
		return err.Error()
	}
	return ""
}

func GMCmd(cmd []string) string {
	// 发送登录请求
	err := GMRpc(GetCurrentUser(), cmd)
	if err != nil {
		return err.Error()
	}
	return ""
}
