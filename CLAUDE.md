@AGENTS.md

## Claude Code

- 共享规则权威源为根目录 `AGENTS.md`；本文件仅作为 Claude Code 兼容入口保留，不复制共享全文（`@import` 深度上限为 4 跳，且在启动时计入上下文）。
- 仅当确认团队全部 Claude Code 会话运行在 `v2.1.277+`、原生 `AGENTS.md` 加载已启用且无 Claude 专属补充时，才可移除本文件。
- 若后续需要按路径条件加载 Claude 专属规则，使用 `.claude/rules/` 并配置 `paths`（普通 Markdown 链接不会触发 `@path` 导入）。
- 仓库跨工具 Agent Skills 位于 `.agents/skills/`；如在当前 Claude Code 版本中未自动发现该目录，按需直接读取对应 `SKILL.md` 或按 [docs/ai/client-compatibility.md](docs/ai/client-compatibility.md) 评估兼容方案。
- 在 Claude Code 会话中可使用 `/context` 核对实际加载文件，确认 `AGENTS.md` 未被本地未跟踪的 `CLAUDE.local.md` 或 `.claude/CLAUDE.md` 遮蔽。
