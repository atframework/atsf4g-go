// client.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	cmd "github.com/atframework/atsf4g-go/robot/cmd"
	config "github.com/atframework/atsf4g-go/robot/config"
	utils "github.com/atframework/atsf4g-go/robot/utils"
)

func main() {
	flagSet := flag.NewFlagSet(
		fmt.Sprintf("%s [options...]", filepath.Base(os.Args[0])), flag.ContinueOnError)
	flagSet.String("url", "ws://localhost:7001/ws/v1", "server socket url")
	flagSet.Bool("h", false, "show help")
	flagSet.Bool("help", false, "show help")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Println(err)
		return
	}

	if flagSet.Lookup("help").Value.String() == "true" ||
		flagSet.Lookup("h").Value.String() == "true" {
		flagSet.PrintDefaults()
		return
	}

	config.SocketUrl = flagSet.Lookup("url").Value.String()
	fmt.Println("URL:", config.SocketUrl)

	utils.ReadLine()

	log.Println("Closing all pending connections")
	currentUser := cmd.GetCurrentUser()
	if currentUser != nil {
		currentUser.Logout()
		<-time.After(1 * time.Second)
		log.Println("Exiting....")
	}
}
