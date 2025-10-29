package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chzyer/readline"
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

	fmt.Println("Enter 'quit' to Exit")
	fmt.Println(CurrentUser.CmdHelpInfo())

	for {
		cmd, err := rl.Readline()
		if err != nil {
			continue
		}
		if cmd == "quit" {
			break
		}
		cmdInfo := onRecvCmd(cmd)
		if cmdInfo != "" {
			fmt.Println(cmdInfo)
		}
	}

	log.Println("Closing all pending connections")

	currentUser := GetCurrentUser()
	if currentUser != nil {
		currentUser.Logout()
		<-time.After(1 * time.Second)
		log.Println("Exiting....")
	}
}
