package atsf4g_go_robot_user

import (
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	"google.golang.org/protobuf/proto"
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
	RunTask(timeout time.Duration, f func(*TaskActionUser) error, name string) *TaskActionUser
	RunTaskDefaultTimeout(f func(*TaskActionUser) error, name string) *TaskActionUser
	AddOnClosedHandler(f func(User))
	Log(format string, a ...any)
	AwaitReceiveHandlerClose()

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
	RegisterMessageHandler(rpcName string, f func(*TaskActionUser, proto.Message, int32) error)
}

var createUserFn func(openId string, socketUrl string, logHandler func(format string, a ...any), enableActorLog bool) User

func RegisterCreateUser(f func(openId string, socketUrl string, logHandler func(format string, a ...any), enableActorLog bool) User) {
	createUserFn = f
}

func CreateUser(openId string, socketUrl string, logHandler func(format string, a ...any), enableActorLog bool) User {
	if createUserFn == nil {
		return nil
	}
	return createUserFn(openId, socketUrl, logHandler, enableActorLog)
}
