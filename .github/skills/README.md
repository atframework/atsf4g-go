# Skills (Legacy Playbooks / Compatibility Index)

> **权威入口迁移说明**：本仓库跨工具标准 Agent Skills 已迁至 [/.agents/skills/](../../.agents/skills/README.md)，共享规则权威源为根目录 [AGENTS.md](../../AGENTS.md)。本目录文件保留以确保历史链接与子模块 `atframework/libatbus-go/CLAUDE.md`（引用 `testing.md`）兼容。

`/.github/copilot-instructions.md` stays short and links to `AGENTS.md` and `/.agents/skills/`.

## Index

- `build-gotask.md` - Build, bootstrap, submodules, and common `task` commands.
- `codegen.md` - Proto/config/RPC-related generation flow and where templates/rules live.
- `architecture.md` - High-level service/module/data-layer structure and common patterns.
- `implementation-details.md` - Caching/dirty-tracking/protobuf mutability conventions.
- `rpc-handler.md` - Standard RPC task action implementation checklist.
- `config-expression.md` - Configuration loading and expression expansion.
- `testing.md` - Unit test conventions and how to run tests in this repo.
