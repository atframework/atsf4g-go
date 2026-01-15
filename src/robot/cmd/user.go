package atsf4g_go_robot_cmd

import (
	"fmt"
	"strconv"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	protocol "github.com/atframework/atsf4g-go/robot/protocol"
	task "github.com/atframework/atsf4g-go/robot/task"
	base "github.com/atframework/robot-go/base"
	robot_cmd "github.com/atframework/robot-go/cmd"
	user_data "github.com/atframework/robot-go/data"
	utils "github.com/atframework/robot-go/utils"
)

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommandDefaultTimeout([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议", nil)
	robot_cmd.RegisterUserCommand([]string{"user", "logout"}, LogoutCmd, "", "登出协议", nil)
	robot_cmd.RegisterUserCommand([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息", nil)
	robot_cmd.RegisterUserCommand([]string{"user", "benchmark"}, BenchmarkCmd, "", "压测协议", nil)
	robot_cmd.RegisterUserCommand([]string{"user", "useItem"}, UseItemCmd, "<itemId> <count> [use_param_index]", "使用道具协议", nil)
	robot_cmd.RegisterUserCommand([]string{"gm"}, GMCmd, "<args...>", "GM", nil)
}

func LogoutCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	err := action.AwaitTask(user.RunTaskDefaultTimeout(task.LogoutTask, "Logout Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func LoginCmd(action base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 1 {
		return "Need OpenId"
	}

	openId := cmd[0]
	u, err := robot_cmd.CmdCreateUser(action, openId)
	if err != nil {
		return err.Error()
	}

	err = action.AwaitTask(u.RunTaskDefaultTimeout(task.LoginTask, "Login Task"))
	if err != nil {
		return err.Error()
	}
	robot_cmd.SetCurrentUser(u)
	return ""
}

func GetInfoCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	// 发送登录请求
	err := action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		errCode, _, rpcErr := protocol.GetInfoRpc(task, cmd)
		if rpcErr != nil {
			return rpcErr
		}
		if errCode < 0 {
			return fmt.Errorf("get info failed, errCode: %d", errCode)
		}
		task.User.SetHasGetInfo(true)
		return nil
	}, "GetInfo Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func UseItemCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	if len(cmd) < 2 {
		return "Args Error"
	}

	itemId, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}
	count, err := strconv.ParseInt(cmd[1], 10, 64)
	if err != nil {
		return err.Error()
	}

	item := &public_protocol_common.DItemBasic{
		TypeId: int32(itemId),
		Count:  int64(count),
	}
	param := &public_protocol_common.DItemUseParam{}
	for index := 2; index < len(cmd); index++ {
		index, err := strconv.ParseInt(cmd[index], 10, 32)
		if err != nil {
			return err.Error()
		}
		param.RandomPoolIndex = append(param.RandomPoolIndex, int32(index))
	}

	err = action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.UsingItemRpc(task, item, param)
		return rpcErr
	}, "UsingItem Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func BenchmarkCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	var count int64 = 1000
	if len(cmd) >= 1 {
		count, _ = strconv.ParseInt(cmd[0], 10, 32)
	}

	for range count {
		user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
			return protocol.PingRpc(task)
		}, "Ping Task")
	}
	return ""
}

func GMCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	// 发送登录请求
	err := action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.GMRpc(task, cmd)
		return rpcErr
	}, "GM Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}
