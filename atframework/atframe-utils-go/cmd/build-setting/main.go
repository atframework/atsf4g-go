package main

import (
	"flag"
	"fmt"
	"os"

	libatframe_utils "github.com/atframework/atframe-utils-go"
)

type buildMgr libatframe_utils.BuildMananger

// 创建 manager（支持 -settings-file 参数）
func createManager() buildMgr {
	settingsFile := ""

	// 扫描命令行参数查找 -settings-file
	for i, arg := range os.Args {
		if arg == "-settings-file" && i+1 < len(os.Args) {
			settingsFile = os.Args[i+1]
			break
		}
	}

	var manager buildMgr
	var err error

	// 如果没有指定，使用当前目录
	if settingsFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: failed to get current working directory: %v\n", err)
			os.Exit(1)
		}
		settingsFile = cwd + "/build-settings.json"

		manager, err = libatframe_utils.NewBuildManager(settingsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: failed to create build manager: %v\n", err)
			os.Exit(1)
		}
	} else {
		manager, err = libatframe_utils.BuildManagerLoad(settingsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: failed to create build manager with settings file '%s': %v\n", settingsFile, err)
			os.Exit(1)
		}
	}
	return manager
}

// init 命令：删除老文件并生成新的
func cmdInit(manager buildMgr) error {
	if err := manager.Init(); err != nil {
		return err
	}
	return nil
}

// get 命令：获取工具路径
func cmdGet(manager buildMgr, toolName string) error {
	if toolName == "" {
		return fmt.Errorf("tool name required")
	}

	path, err := manager.GetToolPath(toolName)
	if err != nil {
		return err
	}

	fmt.Print(path)
	return nil
}

// set 命令：设置工具信息
func cmdSet(manager buildMgr, toolName, version, path string) error {
	if toolName == "" || path == "" {
		return fmt.Errorf("toolName %s path %s tool name and path required", toolName, path)
	}

	if err := manager.SetTool(toolName, version, path); err != nil {
		return err
	}

	fmt.Printf("✅ Tool '%s' configured:\n", toolName)
	fmt.Printf("   Path:    %s\n", path)
	fmt.Printf("   Version: %s\n", version)
	return nil
}

// reset 命令：重置某个工具配置
func cmdReset(manager buildMgr, toolName string) error {
	if toolName == "" {
		return fmt.Errorf("tool name required")
	}

	if err := manager.ResetTool(toolName); err != nil {
		return err
	}

	fmt.Printf("✅ Tool '%s' reset\n", toolName)
	return nil
}

// list 命令：列出所有工具
func cmdList(manager buildMgr) error {
	tools, err := manager.ListTools()
	if err != nil {
		return err
	}

	fmt.Println("🔧 Installed Tools:")
	for name, toolInfo := range tools {
		fmt.Printf("  %s: %s\n", name, toolInfo)
	}
	fmt.Println()
	return nil
}

// setdir 命令：设置当前文档目录（如果不存在配置文件则自动初始化）
func cmdSetDir(manager buildMgr, dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path required")
	}

	if err := manager.SetDocDir(dir); err != nil {
		return err
	}

	fmt.Printf("✅ Documentation directory set to: %s\n", dir)
	return nil
}

// removeSettingsFileArg 从参数列表中移除 -settings-file 及其值
func removeSettingsFileArg(args []string) []string {
	for i, arg := range args {
		if arg == "-settings-file" {
			// 移除 -settings-file 及其值
			if i+1 < len(args) {
				return append(args[:i], args[i+2:]...)
			}
			return args[:i]
		}
	}
	return args
}

// parseGetCmd 解析 get 命令
func parseGetCmd(args []string) (string, error) {
	args = removeSettingsFileArg(args)
	getCmd := flag.NewFlagSet("get", flag.ContinueOnError)
	getCmd.Parse(args)

	if getCmd.NArg() < 1 {
		return "", fmt.Errorf("tool name required")
	}

	return getCmd.Args()[0], nil
}

// parseSetCmd 解析 set 命令
func parseSetCmd(args []string) (toolName, version, path string, err error) {
	args = removeSettingsFileArg(args)

	// 第一个参数是工具名称
	if len(args) < 1 {
		return "", "", "", fmt.Errorf("tool name required")
	}

	toolName = args[0]
	flagArgs := args[1:] // 跳过工具名称，只解析后面的标志

	setCmd := flag.NewFlagSet("set", flag.ContinueOnError)
	setPath := setCmd.String("path", "", "Tool path")
	setVersion := setCmd.String("version", "", "Tool version")

	if err := setCmd.Parse(flagArgs); err != nil {
		return "", "", "", err
	}

	// 验证必需的标志
	if *setPath == "" {
		return "", "", "", fmt.Errorf("path required")
	}
	if *setVersion == "" {
		return "", "", "", fmt.Errorf("version required")
	}

	return toolName, *setVersion, *setPath, nil
}

// parseResetCmd 解析 reset 命令
func parseResetCmd(args []string) (string, error) {
	args = removeSettingsFileArg(args)
	resetCmd := flag.NewFlagSet("reset", flag.ContinueOnError)
	resetCmd.Parse(args)

	if resetCmd.NArg() < 1 {
		return "", fmt.Errorf("tool name required")
	}

	return resetCmd.Args()[0], nil
}

// executeCommand 执行命令
func executeCommand(command string, manager buildMgr, args []string) error {
	switch command {
	case "init":
		return cmdInit(manager)

	case "get":
		toolName, err := parseGetCmd(args)
		if err != nil {
			return err
		}
		return cmdGet(manager, toolName)

	case "set":
		toolName, version, path, err := parseSetCmd(args)
		if err != nil {
			return err
		}
		return cmdSet(manager, toolName, version, path)

	case "reset":
		toolName, err := parseResetCmd(args)
		if err != nil {
			return err
		}
		return cmdReset(manager, toolName)

	case "list":
		return cmdList(manager)

	case "setdir":
		if len(args) < 1 {
			return fmt.Errorf("directory path required")
		}
		return cmdSetDir(manager, args[0])

	case "-h", "--help", "help":
		printUsage()
		return nil

	default:
		return fmt.Errorf("unknown command '%s'", command)
	}
}

func printUsage() {
	fmt.Print(`Usage: build-setting <command> [options]

Commands:
  init                           Initialize build-settings.json (deletes old file if exists)
  get <tool-name>               Get the path of a tool
  set <tool-name>               Set tool path
  reset <tool-name>             Reset a tool configuration
  setdir <directory>            Set current documentation directory (auto-init if needed)
  list                          List all tools and their status

Options:
  -settings-file <path>         Specify custom settings file path (optional)

Examples:
  build-setting init
  build-setting init -settings-file "/path/to/settings/build-settings.json"
  
  build-setting get protoc
  build-setting get protoc -settings-file "/path/to/settings/build-settings.json"
  
  build-setting set protoc -path "/usr/bin/protoc" -version "32.1"
  build-setting set protoc -path "/usr/bin/protoc" -version "32.1" -settings-file "/path/to/settings/build-settings.json"
  
  build-setting reset protoc
  build-setting setdir "/path/to/doc/directory"
  build-setting list

Default: Settings file is build-settings.json in current working directory
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]
	manager := createManager()

	err := executeCommand(command, manager, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
