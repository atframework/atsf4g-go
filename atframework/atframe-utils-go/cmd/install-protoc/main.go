package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	libatframe_utils "github.com/atframework/atframe-utils-go"
)

// Config 命令行配置
type Config struct {
	ProtocVersion   string
	GoPluginVersion string
	ToolsDir        string
	SettingsFile    string
}

// parseFlags 解析命令行参数
func parseFlags() Config {
	cfg := Config{}
	flag.StringVar(&cfg.ProtocVersion, "protoc-version", "", "protoc version to install")
	flag.StringVar(&cfg.GoPluginVersion, "go-plugin-version", "", "protoc-go-plugin version to install")
	flag.StringVar(&cfg.ToolsDir, "tools-dir", "", "tools install dir")
	flag.StringVar(&cfg.SettingsFile, "settings-file", "", "build settings file path")
	flag.Parse()

	if cfg.ProtocVersion == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: protoc-version is required\n")
		os.Exit(1)
	}
	if cfg.GoPluginVersion == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: go-plugin-version is required\n")
		os.Exit(1)
	}

	return cfg
}

// installProtoc 主要业务逻辑：完整的 protoc 安装流程
// 该函数可被 main 和测试用例调用
func installProtoc(cfg Config) error {
	settingFile := path.Clean(cfg.SettingsFile)
	fmt.Printf("Installation of protoc version %s with go-plugin %s setting-file %s\n", cfg.ProtocVersion, cfg.GoPluginVersion, settingFile)

	// 步骤1: 加载 build settings
	buildSettingMgr, err := libatframe_utils.BuildManagerLoad(settingFile)
	if err != nil {
		return fmt.Errorf("load build manager failed: %w", err)
	}

	// 步骤2: 安装 protoc 二进制文件
	binPath, err := InstallProtocBin(cfg.ProtocVersion, cfg.ToolsDir)
	if err != nil {
		return fmt.Errorf("install protoc bin failed: %w", err)
	}

	// 步骤3: 更新 build settings
	if buildSettingMgr != nil {
		if err := buildSettingMgr.SetTool("protoc", cfg.ProtocVersion, binPath); err != nil {
			return fmt.Errorf("failed to set protoc in build settings: %w", err)
		}

	}

	// 步骤4: 安装 Go protoc 插件
	err = InstallGoProtocPlugins(cfg.GoPluginVersion)
	if err != nil {
		return fmt.Errorf("install go protoc plugins failed: %w", err)
	}

	// 步骤5: 安装自定义 protoc 插件
	err = InstallCustomProtocPlugins()
	if err != nil {
		return fmt.Errorf("install custom protoc plugins failed: %w", err)
	}

	fmt.Printf("\n🎉 All installations completed successfully!\n")
	return nil
}

func main() {
	cfg := parseFlags()

	// 调用主业务逻辑函数
	if err := installProtoc(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Installation failed: %v\n", err)
		os.Exit(1)
	}
}

func InstallProtocBin(protocVersion string, dsDir string) (binPath string, err error) {

	// 允许 tools 目录留空，默认为 <cwd>/tools/bin
	if strings.TrimSpace(dsDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory failed: %w", err)
		}
		dsDir = filepath.Join(cwd, "tools", "bin")
	}

	if err := os.MkdirAll(dsDir, 0o755); err != nil {
		return "", fmt.Errorf("create tools dir failed: %w", err)
	}

	if v := strings.TrimSpace(protocVersion); v == "" {
		return "", fmt.Errorf("protoc error version: %w", fmt.Errorf(" version: %s format error", v))
	}

	protocPath := libatframe_utils.EnsureProtocExecutableByVersion(protocVersion, dsDir)
	if protocPath == "" {
		return "", fmt.Errorf("ensure protoc executable failed")
	}

	fmt.Printf("protoc is ready at: %s\n", protocPath)
	return protocPath, nil
}

func InstallGoProtocPlugins(goPluginVersion string) error {
	exist, version := GetGoProtocPlugins()
	if !exist {
		fmt.Printf("protoc-go-plugin not installed, installing...\n")
		return InstallPlugin("google.golang.org/protobuf/cmd/protoc-gen-go", goPluginVersion)
	}

	fmt.Printf("protoc-go-plugin has installed version %s, want installing version %s\n", version, goPluginVersion)

	if strings.Compare(strings.TrimSpace(version), strings.TrimSpace(goPluginVersion)) != 0 {
		if err := uninstallPlugin("protoc-gen-go"); err != nil {
			return fmt.Errorf("failed to uninstall protoc-gen-go: %w", err)
		}
		return InstallPlugin("google.golang.org/protobuf/cmd/protoc-gen-go", goPluginVersion)
	} else {
		fmt.Printf("protoc-go-plugin already at required version %s\n", goPluginVersion)
	}

	return nil
}

func InstallCustomProtocPlugins() error {
	// 获取当前执行文件所在目录,然后找到 atframe-utils-go 根目录
	// 当前路径: atframe-utils-go/cmd/install-protoc
	// 需要回到: atframe-utils-go
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// 回到模块根目录
	moduleRoot := filepath.Join(cwd, "..", "..", "..")
	if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err != nil {
		// 如果找不到 go.mod,可能执行目录不同,尝试其他路径
		moduleRoot = filepath.Join(cwd, "..", "..")
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err != nil {
			return fmt.Errorf("failed to locate atframe-utils-go module root: %w", err)
		}
	}

	plugins := []string{
		"./protoc-gen-mutable",
	}

	for _, pluginPath := range plugins {
		pluginName := filepath.Base(pluginPath)
		fmt.Printf("Installing custom plugin %s...\n", pluginName)

		cmd := exec.Command("go", "install", pluginPath)
		cmd.Dir = moduleRoot // 在模块根目录执行
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", pluginName, err)
		}
		fmt.Printf("Successfully installed %s\n", pluginName)
	}

	return nil
}

// GetGoBinPath 获取 Go bin 目录路径
func GetGoBinPath() (string, error) {
	// 首先尝试获取 GOBIN
	gobin := os.Getenv("GOBIN")
	if gobin != "" {
		return gobin, nil
	}

	// 如果 GOBIN 未设置,使用 GOPATH/bin
	cmd := exec.Command("go", "env", "GOPATH")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOPATH: %w", err)
	}

	gopath := strings.TrimSpace(string(output))
	if gopath == "" {
		return "", fmt.Errorf("GOPATH is not set")
	}

	return filepath.Join(gopath, "bin"), nil
}

// GetInstalledPlugins 获取已安装的插件列表
func GetGoProtocPlugins() (exist bool, version string) {
	binPath, err := GetGoBinPath()
	if err != nil {
		fmt.Printf("GetGoBinPath failed: %v", err)
		return false, version
	}

	entries, err := os.ReadDir(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("protoc-gen-go not found in %s\n", binPath)
			return false, version
		}
		fmt.Printf("ReadDir failed: %v", err)
		return false, version
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 跨平台兼容: Windows 下是 protoc-gen-go.exe, Linux/Mac 下是 protoc-gen-go
		entryName := entry.Name()
		pluginName := "protoc-gen-go"

		// 移除 .exe 后缀进行比较 (如果有)
		if runtime.GOOS == "windows" && strings.HasSuffix(entryName, ".exe") {
			entryName = strings.TrimSuffix(entryName, ".exe")
		}

		if entryName == pluginName {
			// 获取插件版本
			version, err := GetProtoGoPluginVersion()
			if err != nil {
				fmt.Printf("GetProtoGoPluginVersion failed: %v\n", err)
				return true, ""
			}
			return true, version
		}
	}

	fmt.Printf("protoc-gen-go not found in %s\n", binPath)
	return false, version
}

func GetProtoGoPluginVersion() (string, error) {
	// 跨平台兼容: 在 Windows 上自动添加 .exe 后缀
	pluginName := "protoc-gen-go"
	if runtime.GOOS == "windows" {
		pluginName += ".exe"
	}

	cmd := exec.Command(pluginName, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get protoc-gen-go version: %w", err)
	}

	// 输出格式示例: "protoc-gen-go v1.31.0"
	// 提取 v 后面的版本号
	outputStr := strings.TrimSpace(string(output))

	// 查找 'v' 的位置
	if idx := strings.Index(outputStr, "v"); idx != -1 {
		// 提取从 v 开始的部分
		version := outputStr[idx:]
		// 取第一个空格之前的内容 (如果有的话)
		if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
			version = version[:spaceIdx]
		}
		return strings.TrimSpace(version), nil
	}

	// 如果没有找到 v, 返回原始输出
	return outputStr, nil
}

// RemovePlugin 删除已安装的插件
func RemovePlugin(pluginName string) error {
	binPath, err := GetGoBinPath()
	if err != nil {
		return err
	}

	pluginPath := filepath.Join(binPath, pluginName)
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil // 文件不存在,无需删除
	}

	err = os.Remove(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to remove plugin %s: %w", pluginName, err)
	}

	fmt.Printf("Removed plugin: %s\n", pluginName)
	return nil
}

// InstallPlugin 安装指定版本的插件
func InstallPlugin(module string, version string) error {
	var installPath string
	if version != "" && version != "latest" {
		installPath = fmt.Sprintf("%s@%s", module, version)
	} else {
		installPath = module + "@latest"
	}

	fmt.Printf("Installing %s...\n", installPath)
	cmd := exec.Command("go", "install", installPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", installPath, err)
	}

	fmt.Printf("Successfully installed %s\n", installPath)
	return nil
}

func uninstallPlugin(pluginName string) error {
	return RemovePlugin(pluginName)
}
