package atsf4g_go_robot_case

import (
	"fmt"
	"sync"
	"time"

	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	protocol "github.com/atframework/atsf4g-go/robot/protocol"
	task "github.com/atframework/atsf4g-go/robot/task"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

func init() {
	RegisterCase("login", LoginCase, time.Second*5)
	RegisterCase("logout", LogoutCase, time.Second*5)
	RegisterCase("getinfo", GetInfoCase, time.Second*5)
	RegisterCase("delaccount", DelAccountCase, time.Second*5)
	RegisterCase("enable-random-delay", EnableRandomDelayCase, time.Second*5)
	RegisterCase("disable-random-delay", DisableRandomDelayCase, time.Second*5)
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

func LoginCase(action *TaskActionCase, openId string) error {
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

func LogoutCase(action *TaskActionCase, openId string) error {
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

func GetInfoCase(action *TaskActionCase, openId string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	return action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GetInfoRpc(tau, nil)
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("get info failed, errCode: %d", errCode)
		}
		tau.User.SetHasGetInfo(true)
		return nil
	}, "GetInfo Task"))
}

func DelAccountCase(action *TaskActionCase, openId string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	err := action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GMRpc(tau, []string{"del-account"})
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("del account failed, errCode: %d", errCode)
		}
		return nil
	}, "DelAccount Task"))
	if err != nil {
		return err
	}

	u.AwaitReceiveHandlerClose()
	return nil
}

func EnableRandomDelayCase(action *TaskActionCase, openId string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	err := action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GMRpc(tau, []string{"enable-random-delay"})
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("enable random delay failed, errCode: %d", errCode)
		}
		return nil
	}, "EnableRandomDelay Task"))
	if err != nil {
		return err
	}
	return nil
}

func DisableRandomDelayCase(action *TaskActionCase, openId string) error {
	u := GetUser(openId)
	if u == nil {
		return fmt.Errorf("User Not Found")
	}

	err := action.AwaitTask(u.RunTaskDefaultTimeout(func(tau *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GMRpc(tau, []string{"disable-random-delay"})
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("disable random delay failed, errCode: %d", errCode)
		}
		return nil
	}, "DisableRandomDelay Task"))
	if err != nil {
		return err
	}
	return nil
}
