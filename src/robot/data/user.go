package atsf4g_go_robot_user

import (
	"fmt"
	"strconv"
	"sync"
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
	SendReq(csMsg *public_protocol_extension.CSMsg, csBody proto.Message) error

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

var LoginUserMap = make(map[uint64]User, 0)
var LoginUserMapLock sync.Mutex

func AddLoginUser(user User) {
	LoginUserMapLock.Lock()
	defer LoginUserMapLock.Unlock()
	LoginUserMap[user.GetUserId()] = user
}

func RemoveLoginUser(user User) {
	LoginUserMapLock.Lock()
	defer LoginUserMapLock.Unlock()
	delete(LoginUserMap, user.GetUserId())
}

func init() {
	utils.RegisterCommand([]string{"user", "show_all_login_user"}, func([]string) string {
		LoginUserMapLock.Lock()
		defer LoginUserMapLock.Unlock()
		for _, v := range LoginUserMap {
			fmt.Printf("%d\n", v.GetUserId())
		}
		return ""
	}, "", "显示所有登录User")
	utils.RegisterCommand([]string{"user", "switch"}, func(cmd []string) string {
		if len(cmd) < 1 {
			return "Need User Id"
		}

		userId, err := strconv.ParseInt(cmd[0], 10, 64)
		if err != nil {
			return err.Error()
		}

		LoginUserMapLock.Lock()
		v, ok := LoginUserMap[uint64(userId)]
		LoginUserMapLock.Unlock()
		if !ok {
			return "not found user"
		}

		SetCurrentUser(v)
		return ""
	}, "<userId>", "切换登录User")
}

type ResponseHandle = func(user User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message)

var responseHandles map[string]ResponseHandle

func GetResponseHandles() map[string]ResponseHandle {
	if responseHandles == nil {
		responseHandles = make(map[string]ResponseHandle)
	}
	return responseHandles
}

func RegisterResponseHandle(rpcName string, handle ResponseHandle) {
	if responseHandles == nil {
		responseHandles = make(map[string]ResponseHandle)
	}
	responseHandles[rpcName] = handle
}
