package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"
)

func ReadLine() {
	config := &readline.Config{
		Prompt:       ">>:",             // 设置提示符
		AutoComplete: createCompleter(), // 设置自动补全
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		log.Println("无法创建 readline 实例:", err)
		return
	}
	defer rl.Close()

	for {
		cmdInfo := CurrentUser.CmdHelpInfo()
		if cmdInfo != "" {
			fmt.Println(cmdInfo)
		}
		cmd, err := rl.Readline()
		if err != nil {
			// 读取错误退出
			if err.Error() == "EOF" {
				log.Println("EOF")
				break
			}
			log.Println("End:", err)
			break
		}
		cmdInfo = onRecvCmd(cmd)
		if cmdInfo != "" {
			fmt.Println(cmdInfo)
		}
	}

	// We received a SIGINT (Ctrl + C). Terminate gracefully...
	log.Println("Closing all pending connections")

	currentUser := GetCurrentUser()
	if currentUser != nil && CurrentUser.connection != nil {
		// Close our websocket connection
		err = currentUser.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error during closing websocket:", err)
			return
		}

		<-time.After(1 * time.Second)
		log.Println("Timeout in closing receiving channel. Exiting....")
	}
}
