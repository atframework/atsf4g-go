# Build Settings CLI 使用示例

## 初始化配置
```bash
# 删除旧文件并生成新的 build-settings.json
go run ./cmd/build-setting init
```

## 获取工具路径
```bash
# 获取 protoc 的完整路径
PROTOC_PATH=$(go run ./cmd/build-setting get protoc)
echo "protoc: $PROTOC_PATH"

# 获取其他工具路径
PROTOC_GEN_GO=$(go run ./cmd/build-setting get protoc-gen-go)
PROTOC_GEN_MUTABLE=$(go run ./cmd/build-setting get protoc-gen-mutable)
```

## 设置工具路径
```bash
# 设置 protoc 路径
go run ./cmd/build-setting set protoc -path "/usr/bin/protoc" -version "32.1"

# 尝试再设置一次（会报错，防止覆盖）
go run ./cmd/build-setting set protoc -path "/another/path" -version "32.2"
# 错误：❌ Error: tool 'protoc' already configured at: /usr/bin/protoc
#        Cannot set the same tool twice. Use a different tool name or reinitialize.

# 如果需要修改，先重置
go run ./cmd/build-setting reset protoc
go run ./cmd/build-setting set protoc -path "/new/path" -version "32.2"
```

## 列出所有工具
```bash
# 显示所有工具的状态
go run ./cmd/build-setting list
```

## 重置工具配置
```bash
# 重置某个工具（清空其配置）
go run ./cmd/build-setting reset protoc
```

## 在 Taskfile 中使用

```yaml
tasks:
  build-code-generate:
    desc: "Generate code from protocol buffers"
    cmds:
      # 初始化配置
      - go run ./atframework/atframe-utils-go/cmd/build-setting init
      
      # 获取工具路径（赋值到变量）
      - |
        PROTOC_BIN=$(go run ./atframework/atframe-utils-go/cmd/build-setting get protoc)
        PROTOC_GEN_GO=$(go run ./atframework/atframe-utils-go/cmd/build-setting get protoc-gen-go)
        PROTOC_GEN_MUTABLE=$(go run ./atframework/atframe-utils-go/cmd/build-setting get protoc-gen-mutable)
        
        echo "Using protoc: $PROTOC_BIN"
        echo "Using protoc-gen-go: $PROTOC_GEN_GO"
        echo "Using protoc-gen-mutable: $PROTOC_GEN_MUTABLE"
```

## build-settings.json 文件格式

生成的 JSON 文件示例：

```json
{
  "version": "1.0",
  "timestamp": "2025-11-10T10:30:00Z",
  "platform": "windows",
  "arch": "amd64",
  "tools": {
    "protoc": {
      "name": "Protocol Buffers Compiler",
      "version": "32.1",
      "installed": true,
      "path": "C:\\tools\\protoc\\32.1\\windows-amd64\\protoc.exe",
      "verified": true,
      "lastVerified": "2025-11-10T10:30:00Z"
    },
    "protoc-gen-go": {
      "name": "Go Protocol Buffers Plugin",
      "version": "1.36.9",
      "installed": false,
      "path": "",
      "verified": false,
      "lastVerified": "0001-01-01T00:00:00Z"
    },
    "protoc-gen-mutable": {
      "name": "Custom Mutable Plugin",
      "version": "",
      "installed": false,
      "path": "",
      "verified": false,
      "lastVerified": "0001-01-01T00:00:00Z"
    },
    "python": {
      "name": "Python Runtime",
      "version": "",
      "installed": false,
      "path": "",
      "verified": false,
      "lastVerified": "0001-01-01T00:00:00Z"
    }
  },
  "configurations": {
    "configOutputDir": "src/component/config",
    "excelDir": "build/install/resource/ExcelTables",
    "protoDirs": [
      "src/component/protocol/public",
      "src/component/protocol/private",
      "src/lobbysvr/protocol"
    ]
  }
}
```

## 关键特性

- ✅ **init**: 删除老文件，生成新的 build-settings.json
- ✅ **get**: 获取工具的完整路径（只输出路径，便于变量赋值）
- ✅ **set**: 设置工具路径，防止重复写入（违反则 os.Exit(1) 并打印错误）
- ✅ **reset**: 重置工具配置（允许重新设置）
- ✅ **list**: 列出所有工具状态
- ✅ **文件位置**: build-settings.json 在执行命令的同级目录

