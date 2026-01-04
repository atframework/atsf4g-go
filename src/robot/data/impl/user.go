package atsf4g_go_robot_user_impl

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	pu "github.com/atframework/atframe-utils-go/proto_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	base "github.com/atframework/atsf4g-go/robot/base"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	libatapp "github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	logical_time "github.com/atframework/atsf4g-go/component-logical_time"

	user_data "github.com/atframework/atsf4g-go/robot/data"
)

type User struct {
	OpenId string
	UserId uint64
	ZoneId uint32

	AccessToken string
	LoginCode   string

	Logined           bool
	HasGetInfo        bool
	HeartbeatInterval time.Duration
	LastPingTime      time.Time
	Closed            atomic.Bool

	connectionSequence uint64
	connection         *websocket.Conn
	rpcAwaitTask       sync.Map

	csLog       *libatapp.LogBufferedRotatingWriter
	heartbeatFn func(user user_data.User) error

	onClosed        []func(user user_data.User)
	taskManager     *base.TaskActionManager
	taskActionGuard sync.Mutex
}

type CmdAction struct {
	cmdFn           func(user user_data.User)
	allowedNotLogin bool
}

func CreateUser(openId string, conn *websocket.Conn, bufferWriter *libatapp.LogBufferedRotatingWriter,
	heartbeatFn func(user user_data.User) error,
) *User {
	var _ user_data.User = &User{}
	ret := &User{
		OpenId:             openId,
		UserId:             0,
		ZoneId:             1,
		AccessToken:        fmt.Sprintf("access-token-for-%s", openId),
		connectionSequence: 99,
		connection:         conn,
		csLog:              bufferWriter,
		heartbeatFn:        heartbeatFn,
		taskManager:        base.NewTaskActionManager(),
	}

	var _ user_data.User = ret
	return ret
}

func (u *User) AddOnClosedHandler(f func(user user_data.User)) {
	if f == nil {
		return
	}
	u.onClosed = append(u.onClosed, f)
}

func (u *User) IsLogin() bool {
	if u == nil {
		return false
	}
	if u.Closed.Load() {
		return false
	}
	if !u.Logined {
		return false
	}
	return true
}

func (u *User) CheckPingTask() {
	if !u.IsLogin() {
		return
	}
	if u.LastPingTime.Add(u.HeartbeatInterval).Before(logical_time.GetSysNow()) {
		if u.heartbeatFn != nil {
			err := u.heartbeatFn(u)
			if err != nil {
				utils.StdoutLog("ping error stop check\n")
				return
			}
		}
	}
	time.AfterFunc(5*time.Second, u.CheckPingTask)
}

func (u *User) Logout() {
	if !u.IsLogin() {
		return
	}
	u.Logined = false
	u.Close()
}

func (u *User) TakeActionGuard() {
	u.taskActionGuard.Lock()
}

func (u *User) ReleaseActionGuard() {
	u.taskActionGuard.Unlock()
}

func (user *User) MakeMessageHead(rpcName string, typeName string) *public_protocol_extension.CSMsgHead {
	user.connectionSequence++
	return &public_protocol_extension.CSMsgHead{
		Timestamp:      logical_time.GetLogicalNow().Unix(),
		ClientSequence: user.connectionSequence,
		RpcType: &public_protocol_extension.CSMsgHead_RpcRequest{
			RpcRequest: &public_protocol_extension.RpcRequestMeta{
				RpcName: rpcName,
				TypeUrl: typeName,
			},
		},
	}
}

func (user *User) RunTask(timeout time.Duration, f func(*user_data.TaskActionUser)) *user_data.TaskActionUser {
	if user == nil {
		utils.StdoutLog("User nil")
		return nil
	}
	task := &user_data.TaskActionUser{
		TaskActionBase: *base.NewTaskActionBase(timeout),
		User:           user,
		Fn:             f,
	}
	task.TaskActionBase.Impl = task

	user.taskManager.RunTaskAction(task)
	return task
}

func (user *User) RunTaskDefaultTimeout(f func(*user_data.TaskActionUser)) *user_data.TaskActionUser {
	return user.RunTask(time.Duration(8)*time.Second, f)
}

type rpcResumeData struct {
	body    proto.Message
	rspCode int32
}

func (user *User) ReceiveHandler() {
	defer func() {
		utils.StdoutLog(fmt.Sprintf("User %v:%v connection closed.\n", user.ZoneId, user.UserId))
		user.RunTaskDefaultTimeout(func(action *user_data.TaskActionUser) {
			user.connection = nil
			user.Close()
		})
	}()
	for {
		_, bytes, err := user.connection.ReadMessage()
		if err != nil {
			utils.StdoutLog(fmt.Sprintf("Error in receive: %v", err))
			return
		}

		csMsg := &public_protocol_extension.CSMsg{}
		err = proto.Unmarshal(bytes, csMsg)
		if err != nil {
			utils.StdoutLog(fmt.Sprintf("Error in Unmarshal: %v", err))
			return
		}

		var rpcName string
		var typeName string
		switch csMsg.Head.GetRpcType().(type) {
		case *public_protocol_extension.CSMsgHead_RpcResponse:
			rpcName = csMsg.Head.GetRpcResponse().GetRpcName()
			typeName = csMsg.Head.GetRpcResponse().GetTypeUrl()
		case *public_protocol_extension.CSMsgHead_RpcStream:
			rpcName = csMsg.Head.GetRpcStream().GetRpcName()
			typeName = csMsg.Head.GetRpcStream().GetTypeUrl()
		default:
			utils.StdoutLog("<<<<<<<<<<<<<<<<<<<< Received: Unsupport RpcType <<<<<<<<<<<<<<<<<<<<\n")
			utils.StdoutLog(fmt.Sprintf("%s\n", prototext.Format(csMsg.Head)))
			continue
		}

		utils.StdoutLog(fmt.Sprintf("User: %d Code: %d <<<<<<<<<<<<<<<< Received: %s <<<<<<<<<<<<<<<<<<<\n", user.GetUserId(), csMsg.Head.ErrorCode, rpcName))

		messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
		if err != nil {
			utils.StdoutLog(fmt.Sprintf("Unsupport in TypeName: %s \n", typeName))
			fmt.Fprintf(user.csLog, "%s %s\nHead:%s", logical_time.GetSysNow().Format("2006-01-02 15:04:05.000"),
				fmt.Sprintf("<<<<<<<<<<<<<<<<<<<< Unsupport Received: %s <<<<<<<<<<<<<<<<<<<", rpcName), pu.MessageReadableTextIndent(csMsg.Head))
			continue
		}
		csBody := messageType.New().Interface()

		err = proto.Unmarshal(csMsg.BodyBin, csBody)
		if err != nil {
			utils.StdoutLog(fmt.Sprintf("Error in Unmarshal: %v", err))
			fmt.Fprintf(user.csLog, "%s %s\nHead:%s", logical_time.GetSysNow().Format("2006-01-02 15:04:05.000"),
				fmt.Sprintf("<<<<<<<<<<<<<<<<<<<< Unmarshal Error Received: %s <<<<<<<<<<<<<<<<<<<", rpcName), pu.MessageReadableTextIndent(csMsg.Head))
			return
		}

		fmt.Fprintf(user.csLog, "%s %s\nHead:%s\nBody:%s", logical_time.GetSysNow().Format("2006-01-02 15:04:05.000"),
			fmt.Sprintf("<<<<<<<<<<<<<<<<<<<< Received: %s <<<<<<<<<<<<<<<<<<<", rpcName), pu.MessageReadableTextIndent(csMsg.Head), pu.MessageReadableTextIndent(csBody))
		task, ok := user.rpcAwaitTask.Load(csMsg.Head.ClientSequence)
		if ok {
			user.rpcAwaitTask.Delete(csMsg.Head.ClientSequence)
			task.(*user_data.TaskActionUser).Resume(&base.TaskActionAwaitData{
				WaitingType: base.TaskActionAwaitTypeRPC,
				WaitingId:   csMsg.Head.ClientSequence,
			}, &base.TaskActionResumeData{
				Err: nil,
				Data: rpcResumeData{
					body:    csBody,
					rspCode: csMsg.Head.ErrorCode,
				},
			})
		}
	}
}

type RpcTimeout struct {
	sendTime time.Time
	rpcName  string
	seq      uint64
}

func (user *User) SendReq(action *user_data.TaskActionUser, csMsg *public_protocol_extension.CSMsg, csBody proto.Message, needRsp bool) (int32, proto.Message, error) {
	if user == nil {
		return 0, nil, fmt.Errorf("no login")
	}

	if user.connection == nil {
		return 0, nil, fmt.Errorf("connection not found")
	}

	if user.Closed.Load() {
		return 0, nil, fmt.Errorf("connection lost")
	}

	var csBin []byte
	csBin, _ = proto.Marshal(csMsg)
	titleString := fmt.Sprintf("User: %d >>>>>>>>>>>>>>>>>>>> Sending: %s >>>>>>>>>>>>>>>>>>>>", user.GetUserId(), csMsg.Head.GetRpcRequest().GetRpcName())
	utils.StdoutLog(fmt.Sprintf("%s\n", titleString))
	fmt.Fprintf(user.csLog, "%s %s\nHead:%s\nBody:%s", time.Now().Format("2006-01-02 15:04:05.000"),
		titleString, pu.MessageReadableTextIndent(csMsg.Head), pu.MessageReadableTextIndent(csBody))

	// Send an echo packet every second
	err := user.connection.WriteMessage(websocket.BinaryMessage, csBin)
	if err != nil {
		utils.StdoutLog(fmt.Sprintf("Error during writing to websocket: %v", err))
		return 0, nil, err
	}

	if needRsp {
		user.rpcAwaitTask.Store(csMsg.Head.ClientSequence, action)
		resumeData := action.Yield(base.TaskActionAwaitData{
			WaitingType: base.TaskActionAwaitTypeRPC,
			WaitingId:   csMsg.Head.ClientSequence,
		})
		if resumeData.Err != nil {
			return 0, nil, resumeData.Err
		}
		data := resumeData.Data.(rpcResumeData)
		return data.rspCode, data.body, nil
	}
	return 0, nil, nil
}

func (user *User) Close() {
	if user.Closed.CompareAndSwap(false, true) {
		for _, f := range user.onClosed {
			f(user)
		}
		if user.connection != nil {
			// Close our websocket connection
			err := user.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				utils.StdoutLog(fmt.Sprintf("Error during closing websocket: %v", err))
				return
			}
		}
	}
}

func (user *User) GetLoginCode() string {
	if user == nil {
		return ""
	}
	return user.LoginCode
}

func (user *User) GetLogined() bool {
	if user == nil {
		return false
	}
	return user.Logined
}

func (user *User) GetOpenId() string {
	if user == nil {
		return ""
	}
	return user.OpenId
}

func (user *User) GetAccessToken() string {
	if user == nil {
		return ""
	}
	return user.AccessToken
}

func (user *User) GetUserId() uint64 {
	if user == nil {
		return 0
	}
	return user.UserId
}

func (user *User) GetZoneId() uint32 {
	if user == nil {
		return 0
	}
	return user.ZoneId
}

func (user *User) SetLoginCode(d string) {
	if user == nil {
		return
	}
	user.LoginCode = d
}

func (user *User) SetUserId(d uint64) {
	if user == nil {
		return
	}
	user.UserId = d
}

func (user *User) SetZoneId(d uint32) {
	if user == nil {
		return
	}
	user.ZoneId = d
}

func (user *User) SetLogined(d bool) {
	if user == nil {
		return
	}
	user.Logined = d
}

func (user *User) SetHeartbeatInterval(d time.Duration) {
	if user == nil {
		return
	}
	user.HeartbeatInterval = d
}

func (user *User) SetLastPingTime(d time.Time) {
	if user == nil {
		return
	}
	user.LastPingTime = d
}

func (user *User) SetHasGetInfo(d bool) {
	if user == nil {
		return
	}
	user.HasGetInfo = d
}
