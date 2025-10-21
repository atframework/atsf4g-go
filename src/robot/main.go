// client.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var interrupt chan os.Signal

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
}

func receiveHandler(user *User) {
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

		titleString := fmt.Sprintf("<<<<<<<<<<<<<<<<<<<< Received: %s <<<<<<<<<<<<<<<<<<<<", rpcName)
		log.Printf("%s\n", titleString)
		log.Println(prototext.Format(csMsg.Head))

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
		log.Println(strings.Repeat("-", len(titleString)))
		log.Printf("%s\n\n", prototext.Format(csBody))

		processResponse(user, rpcName, csMsg, csBody)
	}
}

func makeMessageHead(user *User, rpcName string, typeName string) *public_protocol_extension.CSMsgHead {
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

func makeLoginAuthMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSLoginAuthReq{
		OpenId: user.OpenId,
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.AccessToken,
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		SystemId:        public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS,
		PackageVersion:  "0.0.0.1",
		ResourceVersion: "0.0.0.1",
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.login_auth", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeLoginMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	vmem, _ := mem.VirtualMemory()
	cpuInfo, _ := cpu.Info()

	csBody := &lobysvr_protocol_pbdesc.CSLoginReq{
		LoginCode: user.LoginCode,
		OpenId:    user.OpenId,
		UserId:    user.UserId,
		ZoneId:    user.ZoneId,
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.AccessToken,
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		ClientInfo: &public_protocol_pbdesc.DClientDeviceInfo{
			SystemId:       uint32(public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS),
			ClientVersion:  "0.0.0.1",
			SystemSoftware: runtime.GOOS,
			SystemHardware: runtime.GOARCH,
			CpuInfo: func() string {
				if len(cpuInfo) > 0 {
					return fmt.Sprintf("%s - %gMHz", strings.TrimSpace(cpuInfo[0].ModelName), cpuInfo[0].Mhz)
				} else {
					return "unknown"
				}
			}(),
			Memory: uint32(vmem.Total / (1024 * 1024)),
		},
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.login", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makePingMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.ping", string(proto.MessageName(csBody))),
	}

	csMsg.BodyBin, _ = proto.Marshal(csBody)

	csMsg.BodyBin, _ = proto.Marshal(&lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	})

	return &csMsg, csBody
}

func makeUserGetInfoMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGetInfoReq{
		NeedUserInfo:      true,
		NeedUserOptions:   true,
		NeedUserInventory: true,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_get_info", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeUserGMMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: []string{"add-item", "1001", "1"},
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_send_gm_command", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func processMakeRequest(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	user.dispatcherLock.Lock()
	defer user.dispatcherLock.Unlock()

	var csMsg *public_protocol_extension.CSMsg
	var csBody proto.Message
	if user.LoginCode == "" {
		csMsg, csBody = makeLoginAuthMessage(user)
	} else if !user.Logined {
		csMsg, csBody = makeLoginMessage(user)
	} else if user.LastPingTime.Add(user.HeartbeatInterval).Before(time.Now()) {
		csMsg, csBody = makePingMessage(user)
	} else if !user.HasGetInfo {
		// Here could add more messages after login
		csMsg, csBody = makeUserGetInfoMessage(user)
	} else {
		csMsg, csBody = makeUserGMMessage(user)
	}

	return csMsg, csBody
}

func processLoginAuthResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginAuthRsp)
	if !ok {
		log.Println("Can not convert to SCLoginAuthRsp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	if body.GetLoginCode() != "" {
		user.LoginCode = body.GetLoginCode()
	}
	if body.GetUserId() != 0 {
		user.UserId = body.GetUserId()
	}
	if body.GetZoneId() != 0 {
		user.ZoneId = body.GetZoneId()
	}
}

func processLoginResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginRsp)
	if !ok {
		log.Println("Can not convert to SCLoginResp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	if body.GetZoneId() != 0 {
		user.ZoneId = body.GetZoneId()
	}
	user.Logined = true
	if body.GetHeartbeatInterval() > 0 {
		user.HeartbeatInterval = time.Duration(body.GetHeartbeatInterval()) * time.Second
	}
}

func processPongResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.LastPingTime = time.Now()
}

func processGetInfoResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	_, ok := rawBody.(*lobysvr_protocol_pbdesc.SCUserGetInfoRsp)
	if !ok {
		log.Println("Can not convert to SCUserGetInfoRsp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.HasGetInfo = true
}

func buildProcessResponseHandles() map[string]func(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	handles := make(map[string]func(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message))
	handles["proy.LobbyClientService.login_auth"] = processLoginAuthResponse
	handles["proy.LobbyClientService.login"] = processLoginResponse
	handles["proy.LobbyClientService.ping"] = processPongResponse
	handles["proy.LobbyClientService.user_get_info"] = processGetInfoResponse
	return handles
}

var processResponseHandles = buildProcessResponseHandles()

func processResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	user.dispatcherLock.Lock()
	defer user.dispatcherLock.Unlock()

	if handle, ok := processResponseHandles[rpcName]; ok {
		handle(user, rpcName, msg, rawBody)
	}
}

func main() {
	interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	socketUrl := "ws://localhost:7001/ws/v1"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()

	user := &User{
		OpenId:      "123444444",
		UserId:      0,
		ZoneId:      1,
		AccessToken: "access-token-for-123444444",

		connectionSequence: 99,

		connection: conn,
	}
	go receiveHandler(user)

	// Our main loop for the client
	// We send our relevant packets here
	nextMessageInterval := time.Microsecond * 100
	for {
		select {
		case <-time.After(nextMessageInterval):
			nextMessageInterval = time.Second * 3
			var csBin []byte
			csMsg, csBody := processMakeRequest(user)
			csBin, _ = proto.Marshal(csMsg)
			titleString := fmt.Sprintf(">>>>>>>>>>>>>>>>>>>> Sending: %s >>>>>>>>>>>>>>>>>>>>", csMsg.Head.GetRpcRequest().GetRpcName())
			log.Printf("%s\n", titleString)
			log.Println(prototext.Format(csMsg.Head))
			log.Println(strings.Repeat("=", len(titleString)))
			log.Printf("%s\n\n", prototext.Format(csBody))

			// Send an echo packet every second
			err := conn.WriteMessage(websocket.BinaryMessage, csBin)
			if err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}

		case <-interrupt:
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")

			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket:", err)
				return
			}

			<-time.After(1 * time.Second)
			log.Println("Timeout in closing receiving channel. Exiting....")
			return
		}
	}
}
