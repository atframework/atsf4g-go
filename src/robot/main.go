// client.go
package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	component_config "github.com/atframework/atsf4g-go/component-config"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	"google.golang.org/protobuf/proto"

	_ "github.com/atframework/atsf4g-go/robot/case"
	_ "github.com/atframework/atsf4g-go/robot/cmd"
	robot "github.com/atframework/robot-go"
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

func UnpackMessage(msg proto.Message) (rpcName string, typeName string, errorCode int32,
	msgHead proto.Message, bodyBin []byte, sequence uint64, err error) {
	csMsg, ok := msg.(*public_protocol_extension.CSMsg)
	if !ok {
		err = fmt.Errorf("message type invalid: %T", msg)
		return
	}
	switch csMsg.Head.GetRpcType().(type) {
	case *public_protocol_extension.CSMsgHead_RpcResponse:
		rpcName = csMsg.Head.GetRpcResponse().GetRpcName()
		typeName = csMsg.Head.GetRpcResponse().GetTypeUrl()
	case *public_protocol_extension.CSMsgHead_RpcStream:
		rpcName = csMsg.Head.GetRpcStream().GetRpcName()
		typeName = csMsg.Head.GetRpcStream().GetTypeUrl()
	default:
		err = fmt.Errorf("unsupport RpcType: %T", csMsg.Head.GetRpcType())
		return
	}
	errorCode = csMsg.Head.GetErrorCode()
	msgHead = csMsg.Head
	bodyBin = csMsg.BodyBin
	sequence = csMsg.Head.GetClientSequence()
	return
}

func main() {
	flagSet := robot.NewRobotFlagSet()
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

	robot.StartRobot(flagSet, UnpackMessage, func() proto.Message {
		return &public_protocol_extension.CSMsg{}
	})
}
