# BuildSetting 包 API 文档

## 概述

`buildsetting` 包提供了一个 Go 模块化接口，用于管理构建工具的配置持久化。核心是 `Manager` 结构体，它处理所有的读写操作。

## 使用示例

### 基础用法

```go
package main

import (
	"fmt"
	"github.com/atframework/atframe-utils-go/buildsetting"
)

func main() {
	// 1. 创建 Manager
	manager := buildsetting.NewManagerInDir(".taskfiles")
	
	// 2. 初始化配置文件
	if err := manager.Init(); err != nil {
		panic(err)
	}
	
	// 3. 设置工具路径
	if err := manager.SetTool("protoc", "32.1", "/usr/bin/protoc"); err != nil {
		panic(err)
	}
	
	// 4. 读取工具路径
	path, err := manager.GetToolPath("protoc")
	if err != nil {
		panic(err)
	}
	fmt.Println("protoc path:", path)
}
```

## API 文档

### 创建 Manager

#### NewManager(settingsFilePath string) *Manager
创建一个 Manager，指定完整的配置文件路径。

```go
manager := buildsetting.NewManager("/path/to/build-settings.json")
```

#### NewManagerInDir(dir string) *Manager
在指定目录创建 Manager，自动使用 `build-settings.json` 作为文件名。

```go
manager := buildsetting.NewManagerInDir(".taskfiles")
// 等价于 NewManager(".taskfiles/build-settings.json")
```

#### NewManagerInCwd() (*Manager, error)
在当前工作目录创建 Manager。

```go
manager, err := buildsetting.NewManagerInCwd()
if err != nil {
	panic(err)
}
```

### 配置文件操作

#### (m *Manager) Init() error
初始化配置文件（删除旧文件并生成新的默认配置）。

```go
if err := manager.Init(); err != nil {
	fmt.Printf("初始化失败: %v\n", err)
}
```

#### (m *Manager) Read() (*BuildSettings, error)
读取配置文件。返回 nil 如果文件不存在。

```go
settings, err := manager.Read()
if err != nil {
	panic(err)
}
if settings == nil {
	fmt.Println("配置文件不存在")
}
```

#### (m *Manager) Write(settings *BuildSettings) error
写入配置文件。

```go
settings := buildsetting.GetDefaultSettings()
if err := manager.Write(settings); err != nil {
	panic(err)
}
```

### 工具管理

#### (m *Manager) SetTool(toolName, version, path string) error
设置工具路径。防止重复写入（同一工具不允许设置两次）。

```go
if err := manager.SetTool("protoc", "32.1", "/usr/bin/protoc"); err != nil {
	fmt.Printf("设置失败: %v\n", err)
	// 输出可能是: tool 'protoc' already configured at: /usr/bin/protoc
}
```

**错误情况：**
- 工具名为空
- 路径为空
- 配置文件不存在
- 工具已存在于配置中（已设置过）
- 指定的路径不存在
- 文件系统错误

#### (m *Manager) GetToolPath(toolName string) (string, error)
获取工具的完整路径。

```go
path, err := manager.GetToolPath("protoc")
if err != nil {
	fmt.Printf("获取失败: %v\n", err)
}
```

#### (m *Manager) GetTool(toolName string) (*Tool, error)
获取完整的工具信息对象。

```go
tool, err := manager.GetTool("protoc")
if err != nil {
	panic(err)
}
fmt.Printf("Name: %s, Version: %s, Path: %s\n", tool.Name, tool.Version, tool.Path)
```

#### (m *Manager) ResetTool(toolName string) error
重置工具配置（清空路径和版本）。

```go
if err := manager.ResetTool("protoc"); err != nil {
	panic(err)
}
// 现在可以重新调用 SetTool 为同一工具设置新路径
```

#### (m *Manager) ListTools() (map[string]Tool, error)
列出所有工具。

```go
tools, err := manager.ListTools()
if err != nil {
	panic(err)
}
for name, tool := range tools {
	if tool.Installed {
		fmt.Printf("%s: %s (v%s)\n", name, tool.Path, tool.Version)
	}
}
```

#### (m *Manager) VerifyTool(toolName string) (bool, error)
验证工具是否存在。

```go
ok, err := manager.VerifyTool("protoc")
if err != nil {
	panic(err)
}
if ok {
	fmt.Println("工具已验证")
} else {
	fmt.Println("工具不存在")
}
```

### 其他操作

#### (m *Manager) GetSettingsFile() string
获取配置文件的完整路径。

```go
filePath := manager.GetSettingsFile()
fmt.Println("配置文件:", filePath)
```

#### GetDefaultSettings() *BuildSettings
获取默认配置对象（静态方法）。

```go
defaults := buildsetting.GetDefaultSettings()
fmt.Printf("支持的工具: %v\n", len(defaults.Tools))
```

## 数据结构

### Tool
```go
type Tool struct {
	Name         string    `json:"name"`         // 工具名称
	Version      string    `json:"version"`      // 版本号
	Installed    bool      `json:"installed"`    // 是否已安装
	Path         string    `json:"path"`         // 工具路径
	Verified     bool      `json:"verified"`     // 是否已验证存在
	LastVerified time.Time `json:"lastVerified"` // 最后验证时间
}
```

### BuildSettings
```go
type BuildSettings struct {
	Version        string                 `json:"version"`        // 配置版本
	Timestamp      time.Time              `json:"timestamp"`      // 更新时间
	Platform       string                 `json:"platform"`       // 操作系统
	Arch           string                 `json:"arch"`           // 架构
	Tools          map[string]Tool        `json:"tools"`          // 工具映射
	Configurations map[string]interface{} `json:"configurations"` // 其他配置
}
```

## 在其他地方导入使用

### 在 install-protoc 中使用

```go
package main

import (
	"fmt"
	"log"
	"github.com/atframework/atframe-utils-go/buildsetting"
)

func main() {
	// 创建 manager（假设配置文件在 .taskfiles 目录）
	manager := buildsetting.NewManagerInDir(".taskfiles")
	
	// 安装 protoc 后写入配置
	protocPath := "/path/to/installed/protoc"
	if err := manager.SetTool("protoc", "32.1", protocPath); err != nil {
		log.Fatalf("Failed to register protoc: %v", err)
	}
	
	fmt.Println("✅ protoc registered successfully")
}
```

### 在其他命令中使用

```go
package main

import (
	"fmt"
	"log"
	"github.com/atframework/atframe-utils-go/buildsetting"
)

func main() {
	manager := buildsetting.NewManagerInDir(".")
	
	// 获取所有已安装的工具
	tools, err := manager.ListTools()
	if err != nil {
		log.Fatal(err)
	}
	
	for name, tool := range tools {
		if tool.Installed {
			fmt.Printf("%s: %s\n", name, tool.Path)
		}
	}
}
```

## 错误处理

所有可能返回错误的方法都遵循 Go 的错误处理惯例：

```go
if err := manager.SetTool("protoc", "32.1", "/path"); err != nil {
	// 处理错误
	switch err.Error() {
	case "tool 'protoc' already configured at: ...":
		// 工具已配置，需要 Reset
	case "path does not exist: ...":
		// 路径不存在，检查安装
	default:
		log.Printf("Unexpected error: %v", err)
	}
}
```

## 导入方式

在您的 `go.mod` 中（如果使用本地模块）：

```go
require github.com/atframework/atframe-utils-go v0.0.0-00010101000000-000000000000

replace github.com/atframework/atframe-utils-go => ../atframework/atframe-utils-go
```

或直接导入：

```go
import "github.com/atframework/atframe-utils-go/buildsetting"
```
