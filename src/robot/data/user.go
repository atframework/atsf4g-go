package atsf4g_go_robot_user

import (
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	"google.golang.org/protobuf/proto"
)

type UserBase interface {
	IsLogin() bool
	CheckPingTask()
	Logout()
	MakeMessageHead(rpcName string, typeName string) *public_protocol_extension.CSMsgHead
	ReceiveHandler()
	SendReq(csMsg *public_protocol_extension.CSMsg, csBody proto.Message, await bool) error

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
