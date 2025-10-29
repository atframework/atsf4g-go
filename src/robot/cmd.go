package main

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
)

func createCompleter() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("login"),
		readline.PcItem("getInfo"),
		readline.PcItem("gm"),
	)
}

func onRecvCmd(cmd string) string {
	cmds := strings.Split(cmd, " ")
	if len(cmds) == 0 {
		return ""
	}
	switch cmds[0] {
	case "login":
		return LoginCmd(cmds[1:])
	case "getInfo":
		return GetInfoCmd(cmds[1:])
	case "gm":
		return GMCmd(cmds[1:])
	}
	return ""
}

func Logout() string {
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
