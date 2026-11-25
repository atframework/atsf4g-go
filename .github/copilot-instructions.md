# Copilot Instructions for atsf4g-go

本仓库共享工程规则的权威入口位于根目录 [AGENTS.md](../AGENTS.md)。本文件仅保留 VS Code Copilot 兼容导航与高频核心约束，避免复制共享全文。

## 按需读取（Agent Skills 与路径规则）

标准跨工具 Agent Skills 位于 `/.agents/skills/`（VS Code Copilot 支持发现 `.agents/skills/`，索引见 [/.agents/skills/README.md](../.agents/skills/README.md)）：

- 构建、子模块与 proto/xresloader/Mako 代码生成：[/.agents/skills/build-and-codegen/SKILL.md](../.agents/skills/build-and-codegen/SKILL.md)
- `src/lobbysvr` / `src/component` 业务开发、RPC Action 与配置表达式：[/.agents/skills/lobbysvr-development/SKILL.md](../.agents/skills/lobbysvr-development/SKILL.md)
- Go 测试与 Lint 门禁：[/.agents/skills/go-testing-and-lint/SKILL.md](../.agents/skills/go-testing-and-lint/SKILL.md) + [/.github/instructions/gotest.instructions.md](instructions/gotest.instructions.md)
- 文档与代码注释写作规范：[/.agents/skills/docs-and-comments-writing/SKILL.md](../.agents/skills/docs-and-comments-writing/SKILL.md)
- 现代 CLI、PowerShell 7+、超时与重试约定：[/.agents/skills/cli-and-shell-conventions/SKILL.md](../.agents/skills/cli-and-shell-conventions/SKILL.md)
- AI 规则与 Skills 维护：[/.agents/skills/ai-maintenance/SKILL.md](../.agents/skills/ai-maintenance/SKILL.md)

> 旧版 `/.github/skills/*.md` 保留作为历史与子模块兼容路径，不再作为新规则首选位置；VS Code Agent Host 已弃用 `.github/prompts/`，不要新建通用 prompt files。

### 子组件文档（独立子模块，只读参考）

- `libatapp-go`：`/atframework/libatapp-go/.github/copilot-instructions.md`
- `libatbus-go`：`/atframework/libatbus-go/AGENTS.md`、`/atframework/libatbus-go/.github/copilot-instructions.md`

## Repo 关键规则（高优先级）

- 当前活跃主服务模块为 `src/lobbysvr` 与公共组件 `src/component`（当前工作树不存在 `src/robot/` 目录）。
- 模块管理器：必须通过 `data.UserGetModuleManager[T](user)` 获取，并做 nil 检查。
- 配置访问：配置 getter 可能返回 `nil`；禁止跨 RPC/reload 缓存 config 结构体指针。
- 配置表达式：proto 字段 `enable_expression: true` 支持 `$VAR` / `${VAR:-default}` 等环境变量展开。
- 构建与代码生成：优先使用 `task`（以 `Taskfile.yml` 为准），不要手改 `*.pb.go` 等生成文件或手写脚本绕过流水线。
- 终端工具：优先使用已安装的现代 CLI（如 `rg`、`jq`、Mike Farah `yq`、`zstd`），详见根 `AGENTS.md` 与 `cli-and-shell-conventions` Skill。
