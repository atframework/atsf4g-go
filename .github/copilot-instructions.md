# Copilot Instructions for atsf4g-go

本仓库是 Go 游戏服务器框架（`src/lobbysvr` / `src/robot`），包含较多代码生成与多模块构建。

## 优先阅读（Skills/Playbooks）

将可复用的工作流说明集中在 `/.github/skills/`，本文件保持“短、准、只放关键规则”。

- 构建/子模块/常用 gotask：`/.github/skills/build-gotask.md`
- Proto/Config/RPC 代码生成：`/.github/skills/codegen.md`
- 架构与常见模式：`/.github/skills/architecture.md`
- 实现细节（dirty/cache/proto mutability）：`/.github/skills/implementation-details.md`
- RPC Handler 任务动作规范：`/.github/skills/rpc-handler.md`
- 配置加载与表达式展开：`/.github/skills/config-expression.md`
- 测试：`/.github/skills/testing.md` + `/.github/instructions/gotest.instructions.md`

### 子组件文档

- libatapp-go：`/atframework/libatapp-go/.github/copilot-instructions.md`

## Repo 关键规则（高优先级）

- 模块管理器：必须用 `data.UserGetModuleManager[T](user)` 获取，并做 nil-check。
- 配置访问：配置 getter 可能返回 nil；不要跨 RPC 缓存 config struct 指针（配置可刷新）。
- 配置表达式：proto 字段 `enable_expression: true` 的字段支持 `$VAR`/`${VAR:-default}` 环境变量展开（详见 `/.github/skills/config-expression.md`）。
- 构建/生成：优先使用 `task`（`Taskfile.yml` 为准），不要手写脚本绕过流程。
