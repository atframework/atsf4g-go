# ATSF4G-GO 项目文档

欢迎来到 ATSF4G-GO 游戏服务器框架示例文档！

## 项目简介

ATSF4G-GO 是一个基于 libatapp-go 和 libatbus-go 的游戏服务器框架示例项目，展示如何使用 [atframework](https://github.com/atframework) 构建游戏服务。

配合 [atsf4g-co](https://github.com/atframework/atsf4g-co) 使用。

## 主要特性

- **基于 atframework**: 使用成熟的 atapp/atbus 框架
- **协议生成**: Protocol Buffers + 自动代码生成
- **配置管理**: Excel → Protobuf 配置（xresloader）
- **RPC 处理器**: 基于模板自动生成 RPC handler
- **完整示例**: 包含大厅、战斗、机器人等服务示例

## 快速开始

```bash
# 克隆项目
git clone https://github.com/atframework/atsf4g-go.git
cd atsf4g-go

# 构建
task build

# 生成协议和配置
task generate:protocol:public
task generate:excel
```

## 核心服务

### lobbysvr - 大厅服务
玩家交互的核心服务，包含角色、库存、建筑、菜单、任务、商城、冒险等系统。

### battlesvr - 战斗服务
管理游戏战斗房间，处理实时战斗逻辑。

### robot - 机器人服务
用于压测和自动化测试的机器人服务。

## 技术栈

- **语言**: Go 1.21+
- **框架**: libatapp-go, libatbus-go
- **协议**: Protocol Buffers
- **配置**: xresloader (Excel → Protobuf)
- **代码生成**: Mako 模板

## 文档导航

- [快速开始](getting-started/index.md) - 快速上手指南
- [架构设计](architecture/index.md) - 系统架构说明
- [协议文档](protocols/index.md) - 自动生成的协议文档
- [可观测性](observability/index.md) - 日志和调试
- [文档编写指南](writing-guide/index.md) - 如何编写和维护文档

## 相关项目

- [atframework](https://github.com/atframework) - 底层框架
- [atsf4g-co](https://github.com/atframework/atsf4g-co) - C++ 版本
- [xresloader](https://github.com/xresloader) - 配置转换工具

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](https://github.com/atframework/atsf4g-go/blob/main/LICENSE) 文件。
