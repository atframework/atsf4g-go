// client.go
package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var done chan interface{}
var interrupt chan os.Signal

func receiveHandler(connection *websocket.Conn) {
	defer close(done)
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			return
		}
		log.Printf("Received: %s\n", msg)
	}
}

func main() {
	done = make(chan interface{})    // Channel to indicate that the receiverHandler is done
	interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	socketUrl := "ws://localhost:7001/ws/v1"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()
	go receiveHandler(conn)

	// Our main loop for the client
	// We send our relevant packets here
	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 1000):
			csMsg := public_protocol_extension.CSMsg{
				Head: &public_protocol_extension.CSMsgHead{
					RpcType: &public_protocol_extension.CSMsgHead_RpcRequest{
						RpcRequest: &public_protocol_extension.RpcRequestMeta{
							RpcName: "proy.LobbyClientService.login_auth",
						},
					},
				},
			}
			csMsg.BodyBin, _ = proto.Marshal(&public_protocol_pbdesc.CSLoginAuthReq{
				OpenId: "123444444",
			})

			csBin, _ := proto.Marshal(&csMsg)

			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, csBin)
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

			select {
			case <-done:
				log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}
