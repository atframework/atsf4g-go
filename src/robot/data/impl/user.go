package atsf4g_go_robot_user_impl

import (
	"container/list"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	pu "github.com/atframework/atframe-utils-go/proto_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	libatapp "github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

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

	rpcTimeout list.List
	rpcSeq     map[uint64]struct{}

	csLog       *libatapp.LogBufferedRotatingWriter
	heartbeatFn func(user user_data.User) error

	onClosed []func(user user_data.User)

	receiveAction chan func()
	sendAction    chan func()
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
		rpcSeq:             make(map[uint64]struct{}),
		receiveAction:      make(chan func(), 50),
		sendAction:         make(chan func(), 50),
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
	if u.LastPingTime.Add(u.HeartbeatInterval).Before(time.Now()) {
		if u.heartbeatFn != nil {
			err := u.heartbeatFn(u)
			if err != nil {
				log.Println("ping error stop check")
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

func (user *User) MakeMessageHead(rpcName string, typeName string) *public_protocol_extension.CSMsgHead {
	user.connectionSequence++
	return &public_protocol_extension.CSMsgHead{
		Timestamp:      time.Now().Unix(),
		ClientSequence: user.connectionSequence,
		RpcType: &public_protocol_extension.CSMsgHead_RpcRequest{
			RpcRequest: &public_protocol_extension.RpcRequestMeta{
				RpcName: rpcName,
				TypeUrl: typeName,
			},
		},
	}
}

func (user *User) ReceiveHandler() {
	defer func() {
		log.Printf("User %v:%v connection closed.\n", user.ZoneId, user.UserId)
		user.receiveAction <- func() {
			user.connection = nil
			user.Close()
		}
		close(user.receiveAction)
	}()
	for {
		_, bytes, err := user.connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}

		csMsg := &public_protocol_extension.CSMsg{}
		err = proto.Unmarshal(bytes, csMsg)
		if err != nil {
			log.Println("Error in Unmarshal:", err)
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
			log.Printf("<<<<<<<<<<<<<<<<<<<< Received: Unsupport RpcType <<<<<<<<<<<<<<<<<<<<\n")
			log.Println(prototext.Format(csMsg.Head))
			continue
		}

		log.Printf("Code: %d <<<<<<<<<<<<<<<< Received: %s <<<<<<<<<<<<<<<<<<<\n", csMsg.Head.ErrorCode, rpcName)

		fmt.Fprintf(user.csLog, "%s %s\n", time.Now().Format(time.DateTime), fmt.Sprintf("<<<<<<<<<<<<<<<<<<<< Received: %s <<<<<<<<<<<<<<<<<<<", rpcName))
		fmt.Fprintf(user.csLog, "Head:{\n%s}\n", pu.MessageReadableText(csMsg.Head))

		messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
		if err != nil {
			log.Println("Unsupport in TypeName:", typeName)
			continue
		}
		csBody := messageType.New().Interface()

		err = proto.Unmarshal(csMsg.BodyBin, csBody)
		if err != nil {
			log.Println("Error in Unmarshal:", err)
			return
		}
		fmt.Fprintf(user.csLog, "Body:{\n%s}\n\n", pu.MessageReadableText(csBody))
		user.receiveAction <- func() {
			// 通知收包
			delete(user.rpcSeq, csMsg.Head.GetClientSequence())
			if handle, ok := user_data.GetResponseHandles()[rpcName]; ok {
				handle(user, rpcName, csMsg, csBody)
			}
		}
	}
}

type RpcTimeout struct {
	sendTime time.Time
	rpcName  string
	seq      uint64
}

func (user *User) ActionHandler() {
	for {
		select {
		case f := <-user.receiveAction:
			if f == nil {
				return
			}
			f()
		case f := <-user.sendAction:
			if f == nil {
				return
			}
			f()
		case <-time.After(time.Second):
			for {
				if user.rpcTimeout.Len() == 0 {
					break
				}
				firstRpc := user.rpcTimeout.Front().Value.(RpcTimeout)
				if time.Now().After(firstRpc.sendTime.Add(8 * time.Second)) {
					_, ok := user.rpcSeq[firstRpc.seq]
					if ok {
						fmt.Printf("timeout rpc %s\n", firstRpc.rpcName)
						delete(user.rpcSeq, firstRpc.seq)
					}
					user.rpcTimeout.Remove(user.rpcTimeout.Front())
				} else {
					break
				}
			}
		}
	}
}

// 这个接口对于同一个User不能并发
func (user *User) SendReq(csMsg *public_protocol_extension.CSMsg, csBody proto.Message) error {
	if user == nil {
		return fmt.Errorf("no login")
	}

	if user.connection == nil {
		return fmt.Errorf("connection not found")
	}

	if user.Closed.Load() {
		return fmt.Errorf("connection lost")
	}

	user.sendAction <- func() {
		var csBin []byte
		csBin, _ = proto.Marshal(csMsg)
		titleString := fmt.Sprintf(">>>>>>>>>>>>>>>>>>>> Sending: %s >>>>>>>>>>>>>>>>>>>>", csMsg.Head.GetRpcRequest().GetRpcName())
		log.Printf("%s\n", titleString)

		fmt.Fprintf(user.csLog, "%s %s\n", time.Now().Format(time.DateTime), titleString)
		fmt.Fprintf(user.csLog, "Head:{\n%s}\n", pu.MessageReadableText(csMsg.Head))
		fmt.Fprintf(user.csLog, "Body:{\n%s}\n\n", pu.MessageReadableText(csBody))

		// Send an echo packet every second
		err := user.connection.WriteMessage(websocket.BinaryMessage, csBin)
		if err != nil {
			log.Println("Error during writing to websocket:", err)
			return
		}

		user.rpcTimeout.PushBack(RpcTimeout{
			sendTime: time.Now(),
			rpcName:  csMsg.Head.GetRpcRequest().GetRpcName(),
			seq:      csMsg.Head.GetClientSequence(),
		})
		user.rpcSeq[csMsg.Head.GetClientSequence()] = struct{}{}
	}
	return nil
}

func (user *User) Close() {
	if user.Closed.CompareAndSwap(false, true) {
		user_data.RemoveLoginUser(user)
		for _, f := range user.onClosed {
			f(user)
		}
		if user.connection != nil {
			// Close our websocket connection
			err := user.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket:", err)
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
