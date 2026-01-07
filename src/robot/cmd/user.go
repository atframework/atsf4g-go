package atsf4g_go_robot_cmd

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	base "github.com/atframework/atsf4g-go/robot/base"
	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"

	protocol "github.com/atframework/atsf4g-go/robot/protocol"
	task "github.com/atframework/atsf4g-go/robot/task"
	utils "github.com/atframework/atsf4g-go/robot/utils"
)

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommandDefaultTimeout([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "logout"}, LogoutCmd, "", "登出协议", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "benchmark"}, BenchmarkCmd, "", "压测协议", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "useItem"}, UseItemCmd, "<itemId> <count> [use_param_index]", "使用道具协议", nil)
	utils.RegisterCommandDefaultTimeout([]string{"gm"}, GMCmd, "<args...>", "GM", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "show_all_login_user"}, func(action base.TaskActionImpl, cmd []string) string {
		userMapLock.Lock()
		defer userMapLock.Unlock()
		for _, v := range userMapContainer {
			action.Log("%d", v.GetUserId())
		}
		return ""
	}, "", "显示所有登录User", nil)
	utils.RegisterCommandDefaultTimeout([]string{"user", "switch"}, func(action base.TaskActionImpl, cmd []string) string {
		if len(cmd) < 1 {
			return "Need User Id"
		}

		userId, err := strconv.ParseInt(cmd[0], 10, 64)
		if err != nil {
			return err.Error()
		}

		userMapLock.Lock()
		v, ok := userMapContainer[strconv.FormatUint(uint64(userId), 10)]
		userMapLock.Unlock()
		if !ok {
			return "not found user"
		}

		SetCurrentUser(v)
		return ""
	}, "<userId>", "切换登录User", AutoCompleteUseIdWithoutCurrent)
}

func LogoutCmd(action base.TaskActionImpl, cmd []string) string {
	if GetCurrentUser() != nil {
		err := action.AwaitTask(GetCurrentUser().RunTaskDefaultTimeout(task.LogoutTask, "Logout Task"))
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

var currentUser atomic.Pointer[user_data.User]

func GetCurrentUser() user_data.User {
	ret := currentUser.Load()
	if ret == nil {
		return nil
	}

	return *ret
}

func SetCurrentUser(user user_data.User) {
	if user == nil {
		currentUser.Store(nil)
	} else {
		currentUser.Store(&user)
	}

	rlInst := utils.GetCurrentReadlineInstance()
	if rlInst != nil {
		if user != nil {
			rlInst.SetPrompt("\033[32m" + strconv.FormatUint(user.GetUserId(), 10) + " »\033[0m ")
			rlInst.Refresh()
		} else {
			rlInst.SetPrompt("\033[32m»\033[0m ")
			rlInst.Refresh()
		}
	}
}

var (
	userMapContainer map[string]user_data.User
	userMapLock      sync.Mutex
)

func MutableUserMapContainer() map[string]user_data.User {
	if userMapContainer == nil {
		userMapContainer = make(map[string]user_data.User)
	}
	return userMapContainer
}

func AutoCompleteUseId(string) []string {
	userMapLock.Lock()
	defer userMapLock.Unlock()
	var res []string
	for _, k := range userMapContainer {
		res = append(res, strconv.FormatUint(k.GetUserId(), 10))
	}
	return res
}

func AutoCompleteUseIdWithoutCurrent(string) []string {
	userMapLock.Lock()
	defer userMapLock.Unlock()
	var res []string
	for _, k := range userMapContainer {
		if k.GetUserId() == GetCurrentUser().GetUserId() {
			continue
		}
		res = append(res, strconv.FormatUint(k.GetUserId(), 10))
	}
	return res
}

func CurrentUserRunTaskDefaultTimeout(f func(*user_data.TaskActionUser) error, name string) *user_data.TaskActionUser {
	user := GetCurrentUser()
	if user == nil {
		utils.StdoutLog("GetCurrentUser: User nil")
		return nil
	}
	return user.RunTaskDefaultTimeout(f, name)
}

func LogoutAllUsers() {
	userMapLock.Lock()
	userMapContainerCopy := userMapContainer
	userMapContainer = make(map[string]user_data.User)
	userMapLock.Unlock()

	for _, v := range userMapContainerCopy {
		v.Logout()
	}
	for _, v := range userMapContainerCopy {
		v.AwaitReceiveHandlerClose()
	}
}

func LoginCmd(action base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 1 {
		return "Need OpenId"
	}

	openId := cmd[0]

	userMapLock.Lock()
	if existingUser, ok := MutableUserMapContainer()[openId]; ok && existingUser != nil {
		userMapLock.Unlock()
		return "User already logged in"
	}
	userMapLock.Unlock()

	// 创建角色
	u := user_data.CreateUser(openId, config.SocketUrl, action.Log, true)
	if u == nil {
		return "Failed to create user"
	}

	userMapLock.Lock()
	MutableUserMapContainer()[openId] = u
	userMapLock.Unlock()

	u.AddOnClosedHandler(func(user user_data.User) {
		userMapLock.Lock()
		defer userMapLock.Unlock()

		u, ok := MutableUserMapContainer()[openId]
		if !ok || u != user {
			return
		}
		delete(MutableUserMapContainer(), openId)
		user.Log("Remove User: %s", openId)

		if GetCurrentUser() == user {
			SetCurrentUser(nil)
		}
	})

	err := action.AwaitTask(u.RunTaskDefaultTimeout(task.LoginTask, "Login Task"))
	if err != nil {
		return err.Error()
	}
	SetCurrentUser(u)
	return ""
}

func GetInfoCmd(action base.TaskActionImpl, cmd []string) string {
	// 发送登录请求
	err := action.AwaitTask(CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
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

func UseItemCmd(action base.TaskActionImpl, cmd []string) string {
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

	err = action.AwaitTask(CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.UsingItemRpc(task, item, param)
		return rpcErr
	}, "UsingItem Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func BenchmarkCmd(action base.TaskActionImpl, cmd []string) string {
	var count int64 = 1000
	if len(cmd) >= 1 {
		count, _ = strconv.ParseInt(cmd[0], 10, 32)
	}

	for range count {
		CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
			return protocol.PingRpc(task)
		}, "Ping Task")
	}
	return ""
}

func GMCmd(action base.TaskActionImpl, cmd []string) string {
	// 发送登录请求
	err := action.AwaitTask(CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.GMRpc(task, cmd)
		return rpcErr
	}, "GM Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}
