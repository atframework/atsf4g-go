package atsf4g_go_robot_user

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pu "github.com/atframework/atframe-utils-go/proto_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	robot_protocol "github.com/atframework/atsf4g-go/robot/protocol"
	robot_protocol_user "github.com/atframework/atsf4g-go/robot/protocol/user"
	libatapp "github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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
	dispatcherLock     sync.Mutex
	rpcChan            map[uint64]chan string
	csLog              *libatapp.LogBufferedRotatingWriter
}

func CreateUser(openId string, conn *websocket.Conn, bufferWriter *libatapp.LogBufferedRotatingWriter) *User {
	return &User{
		OpenId:             openId,
		UserId:             0,
		ZoneId:             1,
		AccessToken:        fmt.Sprintf("access-token-for-%s", openId),
		connectionSequence: 99,
		connection:         conn,
		rpcChan:            make(map[uint64]chan string),
		csLog:              bufferWriter,
	}
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
		err := robot_protocol_user.PingRpc(u)
		if err != nil {
			log.Println("ping error stop check")
			return
		}
	}
	time.AfterFunc(5*time.Second, u.CheckPingTask)
}

func (u *User) Logout() {
	if !u.IsLogin() {
		return
	}
	u.Logined = false
	u.Closed.Store(true)

	if u.connection != nil {
		// Close our websocket connection
		err := u.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error during closing websocket:", err)
			return
		}
	}
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
		user.Closed.Store(true)
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
		user.processResponse(rpcName, csMsg, csBody)
	}
}

func (user *User) processResponse(rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	user.dispatcherLock.Lock()
	defer user.dispatcherLock.Unlock()

	if handle, ok := robot_protocol.ProcessResponseHandles[rpcName]; ok {
		handle(user, rpcName, msg, rawBody)
		// 通知收包
		c, ok := user.rpcChan[msg.GetHead().GetClientSequence()]
		if ok {
			c <- ""
		}
	}
}

func (user *User) SendReq(csMsg *public_protocol_extension.CSMsg, csBody proto.Message, await bool) error {
	if user == nil {
		return fmt.Errorf("need login")
	}

	if user.connection == nil {
		return fmt.Errorf("connection not found")
	}

	if user.Closed.Load() {
		return fmt.Errorf("connection lost")
	}

	var csBin []byte
	csBin, _ = proto.Marshal(csMsg)
	titleString := fmt.Sprintf(">>>>>>>>>>>>>>>>>>>> Sending: %s >>>>>>>>>>>>>>>>>>>>", csMsg.Head.GetRpcRequest().GetRpcName())
	log.Printf("%s\n", titleString)

	fmt.Fprintf(user.csLog, "%s %s\n", time.Now().Format(time.DateTime), titleString)
	fmt.Fprintf(user.csLog, "Head:{\n%s}\n", pu.MessageReadableText(csMsg.Head))
	fmt.Fprintf(user.csLog, "Body:{\n%s}\n\n", pu.MessageReadableText(csBody))

	if await {
		user.rpcChan[csMsg.GetHead().GetClientSequence()] = make(chan string)
		defer delete(user.rpcChan, csMsg.GetHead().GetClientSequence())
	}

	// Send an echo packet every second
	user.dispatcherLock.Lock()
	err := user.connection.WriteMessage(websocket.BinaryMessage, csBin)
	user.dispatcherLock.Unlock()
	if err != nil {
		log.Println("Error during writing to websocket:", err)
		return err
	}

	if await {
		select {
		case <-user.rpcChan[csMsg.GetHead().GetClientSequence()]:
			// 收到回包
			return nil
		case <-time.After(time.Second * 3):
			log.Println(csMsg.GetHead().GetRpcRequest().GetTypeUrl(), " Timeout")
			return fmt.Errorf("Timeout")
		}
	}
	return nil
}

func (user *User) GetLoginCode() string {
	return user.LoginCode
}
func (user *User) GetLogined() bool {
	return user.Logined
}
func (user *User) GetOpenId() string {
	return user.OpenId
}
func (user *User) GetAccessToken() string {
	return user.AccessToken
}
func (user *User) GetUserId() uint64 {
	return user.UserId
}
func (user *User) GetZoneId() uint32 {
	return user.ZoneId
}
func (user *User) SetLoginCode(d string) {
	user.LoginCode = d
}
func (user *User) SetUserId(d uint64) {
	user.UserId = d
}
func (user *User) SetZoneId(d uint32) {
	user.ZoneId = d
}
func (user *User) SetLogined(d bool) {
	user.Logined = d
}
func (user *User) SetHeartbeatInterval(d time.Duration) {
	user.HeartbeatInterval = d
}
func (user *User) SetLastPingTime(d time.Time) {
	user.LastPingTime = d
}
func (user *User) SetHasGetInfo(d bool) {
	user.HasGetInfo = d
}
