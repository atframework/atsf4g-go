// client.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var interrupt chan os.Signal

func main() {
	interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	socketUrl := "ws://localhost:7001/ws/v1"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()

	openId := "123444444"

	user := &User{
		OpenId:             openId,
		UserId:             0,
		ZoneId:             1,
		AccessToken:        fmt.Sprintf("access-token-for-%s", openId),
		connectionSequence: 99,
		connection:         conn,
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
