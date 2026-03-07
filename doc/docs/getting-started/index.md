# 快速开始

欢迎使用 ATSF4G-GO！本指南将帮助您快速运行示例服务。

## 前置要求

- **Go**: 1.21 或更高版本
- **Task**: 任务运行工具 ([安装指南](https://taskfile.dev/installation/))
- **Protoc**: Protocol Buffers 编译器
- **Python**: 3.8+ (用于代码生成)
- **Git**: 版本管理

### Windows 快速安装

```powershell
# 使用 scoop 安装依赖
scoop install go task python git
scoop bucket add java
scoop install microsoft-lts-jdk

# 或使用 winget
winget install GoLang.Go Task.Task Python.Python.3 Git.Git Microsoft.OpenJDK.25
```

### Linux/macOS

```bash
# 安装 Go (参考 https://go.dev/doc/install)
# 安装 Task (参考 https://taskfile.dev/installation/)
# 安装 Python 3
```

## 克隆项目

```bash
git clone https://github.com/atframework/atsf4g-go.git
cd atsf4g-go
```

## 构建服务

```bash
# 构建所有服务
task build

# 或单独构建
task build:lobbysvr    # 构建大厅服务
task build:robot       # 构建机器人服务
```

## 生成协议和配置

```bash
# 生成协议描述符
task generate:protocol:public
task generate:protocol:private

# 生成 Excel 配置
task generate:excel

# 生成 RPC 处理器代码
task generate:rpc
```

## 运行服务

```bash
# 启动 lobbysvr
cd build/install/lobbysvr/bin
./start_1.1.11.1.bat  # Windows
# 或
./start_1.1.11.1.sh   # Linux/macOS
```

## 项目结构

```
atsf4g-go/
├── src/
│   ├── lobbysvr/          # 大厅服务
│   ├── battlesvr/         # 战斗服务
│   ├── robot/             # 机器人服务
│   ├── component/         # 共享组件
│   └── template/          # 代码生成模板
├── build/
│   ├── install/           # 构建输出
│   └── _gen/              # 生成配置
├── resource/
│   └── ExcelTables/       # Excel 配置源文件
└── doc/                   # 文档
```

## 配置说明

服务配置文件位于 `build/install/<服务名>/conf/`：

- `atapp.yaml`: atapp 框架配置
- `server.yaml`: 业务配置

## 常见问题

### Q: 构建失败，提示 protoc not found

**A:** 确保安装了 protoc 并添加到 PATH，或在 `build/build-settings.json` 中配置 protoc 路径。

### Q: 生成协议时报错

**A:** 检查 Excel 配置文件格式是否正确，运行 `task clean` 后重新生成。

### Q: 服务启动后无法连接

**A:** 检查配置文件中的监听端口和地址，确保没有端口冲突。

## 下一步

- 查看 [架构设计](../architecture/index.md) 了解系统设计
- 阅读 [协议文档](../protocols/index.md) 了解接口定义
- 参考 Copilot 指令 `.github/copilot-instructions.md` 了解开发规范
