package atsf4g_go_robot_case

import (
	"fmt"
	"sync"
	"time"

	cmd "github.com/atframework/atsf4g-go/robot/cmd"
	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	protocol "github.com/atframework/atsf4g-go/robot/protocol"
	task "github.com/atframework/atsf4g-go/robot/task"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

func init() {
	RegisterCase("login", LoginCase, time.Second*5)
	RegisterCase("logout", LogoutCase, time.Second*5)
	RegisterCase("gm", GmCase, time.Second*5)
	RegisterCase("await_close", AwaitCloseCase, time.Second*5)
	RegisterCase("delay_second", DelayCase, 0)
	RegisterCase("run_cmd", RunCmdCase, time.Second*5)
}

var userMapContainer = sync.Map{}

func AddUser(u user_data.User) {
	userMapContainer.Store(u.GetOpenId(), u)
	u.AddOnClosedHandler(func(user user_data.User) {
		DelUser(user.GetOpenId())
	})
}

func DelUser(openId string) {
	userMapContainer.Delete(openId)
}

func GetUser(openId string) user_data.User {
	v, ok := userMapContainer.Load(openId)
	if !ok {
		return nil
	}
	return v.(user_data.User)
}

func DelayCase(action *TaskActionCase, openId string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid args")
	}
	duration, err := time.ParseDuration(args[0] + "s")
	if err != nil {
		return err
	}
	time.Sleep(duration)
	return nil
}

func LoginCase(action *TaskActionCase, openId string, args []string) error {
	// 创建角色
	u := user_data.CreateUser(openId, config.SocketUrl, action.Log, false)
	if u == nil {
		return fmt.Errorf("Failed to create user")
	}

	user_data.RegisterMessageHandlerUserDirtyChgSync(u,
		func(action *user_data.TaskActionUser, msg *lobysvr_protocol_pbdesc.SCUserDirtyChgSync, errCode int32) error {
			// 处理脏数据变更通知
			return nil
		})

	err := action.AwaitTask(u.RunTaskDefaultTimeout(task.LoginTask, "Login Task"))
	if err != nil {
		return err
	}

	err = action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		AddUser(tau.User)
		return nil
	}, "AddUser Task"))
	if err != nil {
		return err
	}

	return nil
}

func LogoutCase(action *TaskActionCase, openId string, args []string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	err := action.AwaitTask(u.RunTaskDefaultTimeout(task.LogoutTask, "Logout Task"))
	if err != nil {
		return err
	}

	u.AwaitReceiveHandlerClose()
	return nil
}

func GmCase(action *TaskActionCase, openId string, args []string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	if len(args) < 1 {
		return fmt.Errorf("invalid args")
	}

	return action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GMRpc(tau, args[0:])
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("gm command failed, errCode: %d", errCode)
		}
		return nil
	}, "Gm Task"))
}

func AwaitCloseCase(action *TaskActionCase, openId string, args []string) error {
	u := GetUser(openId)
	if u == nil {
		return nil
	}

	u.AwaitReceiveHandlerClose()
	return nil
}

func RunCmdCase(action *TaskActionCase, openId string, args []string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	cmdArgs, fn := cmd.GetUserCommandFunc(args)
	if fn == nil {
		return fmt.Errorf("Command Not Found")
	}

	result := fn(action, u, cmdArgs)
	if result != "" {
		return fmt.Errorf(result)
	}

	return nil
}
