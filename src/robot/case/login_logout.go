package atsf4g_go_robot_case

import (
	"fmt"
	"time"

	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	task "github.com/atframework/atsf4g-go/robot/task"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterCase("login-logout", LoginLogoutCase, 0)
}

func LoginLogoutCase(action *TaskActionCase, openId string) error {
	// 创建角色
	u := user_data.CreateUser(openId, config.SocketUrl, action.Log, false)
	if u == nil {
		return fmt.Errorf("Failed to create user")
	}

	u.RegisterMessageHandler(user_data.GetUserDirtyChgSyncResponseRpcName(),
		func(action *user_data.TaskActionUser, msg proto.Message, errCode int32) error {
			// 处理脏数据变更通知
			return nil
		})

	err := action.AwaitTask(u.RunTask(time.Second*15, task.LoginTask, "Login Task"))
	if err != nil {
		return err
	}

	err = action.AwaitTask(u.RunTaskDefaultTimeout(task.LogoutTask, "Logout Task"))
	if err != nil {
		return err
	}

	u.AwaitReceiveHandlerClose()
	return nil
}
