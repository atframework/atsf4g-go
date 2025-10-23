// client.go
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

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
