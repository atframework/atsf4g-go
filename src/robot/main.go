// client.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"

	component_config "github.com/atframework/atsf4g-go/component-config"

	config "github.com/atframework/atsf4g-go/robot/config"
	utils "github.com/atframework/atsf4g-go/robot/utils"

	robot_case "github.com/atframework/atsf4g-go/robot/case"
	cmd "github.com/atframework/atsf4g-go/robot/cmd"
	_ "github.com/atframework/atsf4g-go/robot/data/impl"
	_ "github.com/atframework/atsf4g-go/robot/protocol"
	_ "github.com/atframework/atsf4g-go/robot/task"
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

	flagSet.String("case_file", "", "case file path")
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

	caseFile := flagSet.Lookup("case_file").Value.String()
	if caseFile != "" {
		err := robot_case.RunCaseFile(caseFile)
		if err != nil {
			fmt.Println("Run case file error:", err)
			os.Exit(1)
		}
	} else {
		utils.ReadLine()
	}

	utils.StdoutLog("Closing all pending connections")
	cmd.LogoutAllUsers()
	utils.StdoutLog("Exiting....")
}
