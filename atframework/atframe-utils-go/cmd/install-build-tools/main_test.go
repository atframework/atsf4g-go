package main

// import (
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"strings"
// 	"testing"
// )

// // TestInstallAtdtools 测试：atdtools 完整安装流程
// func TestInstallAtdtools(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}

// 	testDir := filepath.Join(os.TempDir(), "test-install-atdtools")
// 	defer os.RemoveAll(testDir)

// 	cfg := InstallToolConfig{
// 		AppName:      "atdtools",
// 		AppVersion:   "1.0.2",
// 		DownloadURL:  "https://github.com/atframework/atdtool/releases/download/v1.0.2/atdtool-windows-amd64.zip",
// 		DownloadPath: filepath.Join(testDir, "download"),
// 		InstallPath:  filepath.Join(testDir, "install"),
// 	}

// 	// 只需调用一个函数，所有业务逻辑都在里面
// 	installedPaths, err := installTool(cfg)
// 	if err != nil {
// 		t.Fatalf("installTool failed: %v", err)
// 	}

// 	if len(installedPaths) == 0 {
// 		t.Fatal("No tools installed")
// 	}

// 	for _, path := range installedPaths {
// 		if _, err := os.Stat(path); err != nil {
// 			t.Fatalf("Installed file not found: %s", path)
// 		}

// 		baseName := filepath.Base(path)
// 		if strings.Contains(baseName, cfg.AppVersion) {
// 			t.Errorf("Version number not removed: %s", baseName)
// 		}

// 		fmt.Printf("✅ %s installed: %s\n", cfg.AppName, baseName)
// 	}
// }

// // TestInstallXresloader 测试：xresloader 完整安装流程
// func TestInstallXresloader(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}

// 	testDir := filepath.Join(os.TempDir(), "test-install-xresloader")
// 	defer os.RemoveAll(testDir)

// 	cfg := InstallToolConfig{
// 		AppName:      "xresloader",
// 		AppVersion:   "2.21.0",
// 		DownloadURL:  "https://github.com/owent/xresloader/releases/download/v2.21.0/xresloader-2.21.0.jar",
// 		DownloadPath: filepath.Join(testDir, "download"),
// 		InstallPath:  filepath.Join(testDir, "install"),
// 	}

// 	// 只需调用一个函数，所有业务逻辑都在里面
// 	installedPaths, err := installTool(cfg)
// 	if err != nil {
// 		t.Fatalf("installTool failed: %v", err)
// 	}

// 	if len(installedPaths) == 0 {
// 		t.Fatal("No tools installed")
// 	}

// 	for _, path := range installedPaths {
// 		if _, err := os.Stat(path); err != nil {
// 			t.Fatalf("Installed file not found: %s", path)
// 		}

// 		baseName := filepath.Base(path)
// 		if baseName != "xresloader.jar" {
// 			t.Errorf("Expected xresloader.jar, got %s", baseName)
// 		}

// 		fmt.Printf("✅ %s installed: %s\n", cfg.AppName, baseName)
// 	}
// }

// // TestRemoveVersionFromFileName 单元测试
// func TestRemoveVersionFromFileName(t *testing.T) {
// 	tests := []struct {
// 		fileName   string
// 		appVersion string
// 		expected   string
// 	}{
// 		{"xresloader-2.21.0.jar", "2.21.0", "xresloader.jar"},
// 		{"tool-1.0.0.exe", "1.0.0", "tool.exe"},
// 	}

// 	for _, tt := range tests {
// 		result := removeVersionFromFileName(tt.fileName, "", tt.appVersion)
// 		if result != tt.expected {
// 			t.Errorf("%s: expected %s, got %s", tt.fileName, tt.expected, result)
// 		}
// 	}
// }
