# Agent Skills 索引（`atsf4g-go`）

本目录按 [Agent Skills 开放标准](https://agentskills.io/specification) 组织可复用专业工作流，供支持 `.agents/skills/` 的跨工具 AI 客户端按需渐进加载。

> 说明：本 `README.md` 仅作人类与维护 Agent 的导航索引，不是 Agent Skills 规范要求的发现文件。规则主入口位于根目录 [AGENTS.md](../../AGENTS.md)。

## 技能目录与触发路由

| Skill 名称 | 入口文件 | 适用场景 | 最近排除边界 | 按需参考资源 |
| --- | --- | --- | --- | --- |
| `build-and-codegen` | [SKILL.md](build-and-codegen/SKILL.md) | `go-task` 构建、工具与子模块准备、`protoc-gen-mutable` / xresloader / Mako 代码生成 | 不用于日常 Go 业务逻辑编写、单测设计或文案润色 | 无（直接读取 `Taskfile.yml` 与 `.taskfiles/`） |
| `lobbysvr-development` | [SKILL.md](lobbysvr-development/SKILL.md) | `src/lobbysvr` 与 `src/component` 业务模块、RPC `TaskActionCSBase`、DB 持久化、脏数据同步与配置表达式 | 不用于全量构建流水线或纯文风修改 | 子模块 `atframework/libatapp-go/.github/skills/config-expression.md` |
| `go-testing-and-lint` | [SKILL.md](go-testing-and-lint/SKILL.md) | Go 单元测试/回归测试、按模块 `go test`、`golangci-lint`、`.proto` 格式与 Markdown/图表门禁 | 不用于代码生成或纯散文/文档文风改写 | `.github/instructions/gotest.instructions.md`、`.golangci.yaml` |
| `docs-and-comments-writing` | [SKILL.md](docs-and-comments-writing/SKILL.md) | Go 注释、文档注释、README、PR/变更说明、技术方案与 MkDocs 文档写作与修订 | 不用于机械代码格式化、代码生成或修改 API 行为 | [writing-rules-detail.md](docs-and-comments-writing/references/writing-rules-detail.md) |
| `cli-and-shell-conventions` | [SKILL.md](cli-and-shell-conventions/SKILL.md) | 现代 CLI 选型（含完整 30 项候选与回退）、Windows PowerShell 7+ 规则、退出码、超时/重试/幂等与跨环境 Git 传输 | 不用于编写 Go 业务代码或文档正文润色 | [modern-cli-catalog.md](cli-and-shell-conventions/references/modern-cli-catalog.md) |
| `ai-maintenance` | [SKILL.md](ai-maintenance/SKILL.md) | 维护 `AGENTS.md`、兼容层、Skills、覆盖表、来源索引、Skill 触发/质量评估与自我改进 | 不用于普通业务功能开发或缺陷修复 | `docs/ai/{initialization-coverage,source-index,client-compatibility,engineering-governance}.md` |

## 兼容说明

- **支持直接发现 `.agents/skills/` 的客户端**：Antigravity、VS Code Copilot、Kilo Code、Pi、Oh My Pi、Codex、OpenCode、Command Code、OpenClaw、Hermes Agent、Devin Desktop（具体开关、信任边界与覆盖顺序见 [docs/ai/client-compatibility.md](../../docs/ai/client-compatibility.md)）。
- **旧路径保护**：仓库保留 `.github/skills/*.md` 与 `.github/instructions/gotest.instructions.md`，避免破坏既有工作流及子模块 `atframework/libatbus-go/CLAUDE.md` 的相对路径导入。
