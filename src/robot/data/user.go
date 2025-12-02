package atsf4g_go_robot_user

import (
	"strconv"
	"sync/atomic"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	"google.golang.org/protobuf/proto"

	utils "github.com/atframework/atsf4g-go/robot/utils"
)

type User interface {
	IsLogin() bool
	CheckPingTask()
	Logout()
	MakeMessageHead(rpcName string, typeName string) *public_protocol_extension.CSMsgHead
	ReceiveHandler()
	SendReq(action *TaskActionUser, csMsg *public_protocol_extension.CSMsg, csBody proto.Message, needRsp bool) (int32, proto.Message, error)
	TakeActionGuard()
	ReleaseActionGuard()
	RunTask(timeout time.Duration, f func(*TaskActionUser)) *TaskActionUser
	RunTaskDefaultTimeout(f func(*TaskActionUser)) *TaskActionUser

	GetLoginCode() string
	GetLogined() bool
	GetOpenId() string
	GetAccessToken() string
	GetUserId() uint64
	GetZoneId() uint32

	SetLoginCode(string)
	SetUserId(uint64)
	SetZoneId(uint32)
	SetLogined(bool)
	SetHeartbeatInterval(time.Duration)
	SetLastPingTime(time.Time)
	SetHasGetInfo(bool)
}

var currentUser atomic.Pointer[User]

func GetCurrentUser() User {
	ret := currentUser.Load()
	if ret == nil {
		return nil
	}

	return *ret
}

func SetCurrentUser(user User) {
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

func CurrentUserRunTaskDefaultTimeout(f func(*TaskActionUser)) *TaskActionUser {
	user := GetCurrentUser()
	if user == nil {
		utils.StdoutLog("GetCurrentUser: User nil")
		return nil
	}
	return user.RunTaskDefaultTimeout(f)
}
