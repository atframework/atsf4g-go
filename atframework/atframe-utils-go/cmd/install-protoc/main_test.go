package main

import (
	"testing"
)

// TestGetGoBinPath 测试获取 Go bin 路径
func TestGetGoBinPath(t *testing.T) {
	binPath, err := GetGoBinPath()
	if err != nil {
		t.Fatalf("GetGoBinPath failed: %v", err)
	}

	if binPath == "" {
		t.Fatal("GetGoBinPath returned empty path")
	}

	t.Logf("Go bin path: %s", binPath)
}

func TestGetGoProtocPlugins(t *testing.T) {
	GetGoProtocPlugins()
}

// TestInstallProtoc 测试 protoc 安装流程
// 模拟命令: go run . --protoc-version 32.1 --go-plugin-version v1.36.9 --tools-dir G:\project\public\server\project/build/tools --settings-file G:\project\public\server\project/build/build-settings.json
// func TestInstallProtoc(t *testing.T) {
// 	cfg := Config{
// 		ProtocVersion:   "32.1",
// 		GoPluginVersion: "v1.36.9",
// 		ToolsDir:        "G:\\project\\public\\server\\project\\build\\tools",
// 		SettingsFile:    "G:\\project\\public\\server\\project\\build\\build-settings.json",
// 	}

// 	err := installProtoc(cfg)
// 	if err != nil {
// 		t.Fatalf("installProtoc failed: %v", err)
// 	}

// 	t.Log("✅ protoc installation test passed")
// }
