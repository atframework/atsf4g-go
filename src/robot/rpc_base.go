// client.go
package main

import (
	"fmt"
	"log"
	"time"

	pu "github.com/atframework/atframe-utils-go/proto_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

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
		processResponse(user, rpcName, csMsg, csBody)
	}
}

func processResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	user.dispatcherLock.Lock()
	defer user.dispatcherLock.Unlock()

	if handle, ok := processResponseHandles[rpcName]; ok {
		handle(user, rpcName, msg, rawBody)
		// 通知收包
		c, ok := user.rpcChan[msg.GetHead().GetClientSequence()]
		if ok {
			c <- ""
		}
	}
}

func sendReq(user *User, csMsg *public_protocol_extension.CSMsg, csBody proto.Message, await bool) error {
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
