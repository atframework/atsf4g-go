// client.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	component_config "github.com/atframework/atsf4g-go/component-config"

	config "github.com/atframework/atsf4g-go/robot/config"
	utils "github.com/atframework/atsf4g-go/robot/utils"

	user_data "github.com/atframework/atsf4g-go/robot/data"

	_ "github.com/atframework/atsf4g-go/robot/protocol"
)

func guessResourceDir() string {
	cwd, _ := os.Getwd()
	// Check cwd
	baseDir := cwd

	currentCheckDir := baseDir
	for i := 0; i < 5; i++ {
		resourceDir := filepath.Join(currentCheckDir, "resource")
		_, err := os.Stat(filepath.Join(resourceDir, "pbdesc", "public-config.pb"))
		if err == nil {
			if ret, err := filepath.Abs(resourceDir); err == nil {
				return ret
			}

			return resourceDir
		}
		currentCheckDir = filepath.Join(currentCheckDir, "..")
	}

	// Check executable dir
	baseDir = filepath.Join(filepath.Dir(os.Args[0]))
	currentCheckDir = baseDir
	for i := 0; i < 5; i++ {
		resourceDir := filepath.Join(currentCheckDir, "resource")
		_, err := os.Stat(filepath.Join(resourceDir, "pbdesc", "public-config.pb"))
		if err == nil {
			if ret, err := filepath.Abs(resourceDir); err == nil {
				return ret
			}

			return resourceDir
		}
		currentCheckDir = filepath.Join(currentCheckDir, "..")
	}

	if resourceDir, err := filepath.Abs(filepath.Join(cwd, "..", "..", "resource")); err == nil {
		return resourceDir
	}

	return filepath.Join(filepath.Dir(os.Args[0]), "..", "..", "resource")
}

func main() {
	flagSet := flag.NewFlagSet(
		fmt.Sprintf("%s [options...]", filepath.Base(os.Args[0])), flag.ContinueOnError)
	flagSet.String("url", "ws://localhost:7001/ws/v1", "server socket url")
	flagSet.Bool("h", false, "show help")
	flagSet.Bool("help", false, "show help")

	flagSet.String("resource", "", "resource directory")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Println(err)
		return
	}

	var resourceDir string
	if flagSet.Lookup("resource").Value.String() != "" {
		resourceDir = flagSet.Lookup("resource").Value.String()
	} else {
		resourceDir = guessResourceDir()
	}
	_, err := os.Stat(filepath.Join(resourceDir, "pbdesc", "public-config.pb"))
	if err == nil {
		component_config.GetConfigManager().SetResourceDir(path.Join(resourceDir, "excel"))
		component_config.GetConfigManager().Init(context.Background())
		component_config.GetConfigManager().Reload()
	} else {
		fmt.Printf("Resource dir %s not found or invalid\n", resourceDir)
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

	utils.StdoutLog("Closing all pending connections\n")
	currentUser := user_data.GetCurrentUser()
	if currentUser != nil {
		currentUser.Logout()
		<-time.After(1 * time.Second)
		utils.StdoutLog("Exiting....\n")
	}
}
