package atsf4g_go_robot_protocol

import (
	"fmt"
	"log"
	"runtime"
	"time"

	config "github.com/atframework/atsf4g-go/robot/config"
	user "github.com/atframework/atsf4g-go/robot/data/impl"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	"github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"

	robot_protocol_user "github.com/atframework/atsf4g-go/robot/protocol/user"
)

func init() {
	utils.RegisterCommand([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议")
	utils.RegisterCommand([]string{"user", "logout"}, LogoutCmd, "", "登出协议")
	utils.RegisterCommand([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息")
	utils.RegisterCommand([]string{"gm"}, GMCmd, "<args...>", "GM")
}

var CurrentUser *user.User

func GetCurrentUser() *user.User {
	return CurrentUser
}

func CreateUser(openId string, socketUrl string) {
	bufferWriter, _ := libatapp.NewlogBufferedRotatingWriter(
		"../log", openId, 1*1024*1024, 3, time.Second*3, false, false)
	runtime.SetFinalizer(bufferWriter, func(writer *libatapp.LogBufferedRotatingWriter) {
		writer.Close()
	})

	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}

	CurrentUser = user.CreateUser(openId, conn, bufferWriter)

	go CurrentUser.ReceiveHandler()
	log.Println("Create User:", openId)
}

func LogoutCmd(cmd []string) string {
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
	CreateUser(cmd[0], config.SocketUrl)

	// 发送登录请求
	err := robot_protocol_user.LoginAuthRpc(GetCurrentUser())
	if err != nil {
		return err.Error()
	}

	err = robot_protocol_user.LoginRpc(GetCurrentUser())
	if err != nil {
		return err.Error()
	}

	// 创建Ping流程
	GetCurrentUser().CheckPingTask()
	return ""
}

func GetInfoCmd(cmd []string) string {
	// 发送登录请求
	err := robot_protocol_user.GetInfoRpc(GetCurrentUser(), cmd)
	if err != nil {
		return err.Error()
	}
	return ""
}

func GMCmd(cmd []string) string {
	// 发送登录请求
	err := robot_protocol_user.GMRpc(GetCurrentUser(), cmd)
	if err != nil {
		return err.Error()
	}
	return ""
}
