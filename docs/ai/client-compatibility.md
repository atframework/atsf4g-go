# 跨工具兼容、文件承载决策与自定义 Agent 规则

本文件作为 `atsf4g-go` 的跨 AI 客户端兼容参考、文件位置决策依据与自定义 Agent 治理说明。规则主入口为根目录 [AGENTS.md](../../AGENTS.md)。

## 1. 文件选择决策表

| 需求类型 | 首选承载位置 | 使用边界与约束 |
| --- | --- | --- |
| 大多数任务必需的项目规则 | 根 [AGENTS.md](../../AGENTS.md) | 保持精简：仅放已核验命令、硬约束、关键路径和按需读取指引 |
| 模块或文件类型专属规则 | 嵌套 `AGENTS.md` 或客户端路径规则（如 `.github/instructions/gotest.instructions.md`） | 启用前确认目标客户端的嵌套发现开关、glob 语法与加载时机；不修改独立子模块规则 |
| 可复用专业工作流 | `.agents/skills/<name>/SKILL.md` | 承载构建、代码生成、领域开发、测试、CLI 约定、写作规范与 AI 维护流程 |
| Claude Code 兼容层 | 根 [CLAUDE.md](../../CLAUDE.md) | 通过 `@AGENTS.md` 引用共享规则，仅补充 Claude 专属差异，不复制全文 |
| VS Code Copilot 兼容层 | `.github/copilot-instructions.md`、`.github/instructions/*.instructions.md` | 指向 `AGENTS.md` 与 `.agents/skills/`；保留被子模块引用的兼容文件路径 |
| 独立 Agent 角色 | 目标客户端专属 agent 配置（当前仓库暂未创建） | 仅在确需工具限制、只读隔离、上下文隔离或 handoff 时创建，不为一次性分工建常驻角色 |
| 手动重复任务 | Agent Skill（`.agents/skills/`）或目标客户端工作流 | VS Code Agent Host 已弃用且不加载 `.github/prompts/`（Local agent 暂可用），不要将 `.github/prompts/` 当作通用入口 |
| 可审阅行为变更 | 现有设计文档（`doc/`）、Issue/PR、ADR；若日后采用 OpenSpec 则用 change | 同一变更只维护一份权威规范，其余位置链接引用 |
| 外部实时数据与操作 | MCP 或已有官方 API/CLI | 先核验来源、认证、权限与数据范围，当前仓库未接入项目级 MCP |
| 强制安全与质量约束 | sandbox、权限配置、Git hooks（`script/hooks/`）、CI（`Jenkinsfile`） | 提示词不能替代运行层权限和门禁限制 |
| 长期工程经验 | 自动化测试、`doc/`、规则或对应 Skill | 仅记录已核验、可复用的结论，避免规则无限膨胀 |

## 2. 跨客户端兼容矩阵

以下逐项记录模板覆盖的全部 13 个客户端在 `atsf4g-go` 中的当前使用情况、规则/Skills 发现入口、覆盖边界与验证状态。**切勿将某个客户端的覆盖顺序、YAML 扩展字段或 `@import` 语法推广到其他客户端。**

| 客户端 | 本仓库状态与本机版本（2026-11-25） | 规则与 Skills 入口 | 关键边界与兼容处置 | 运行验收状态 |
| --- | --- | --- | --- | --- |
| **Antigravity** | **实际使用**（当前执行环境；独立 `agy` CLI 不在 PATH） | 工作区 `AGENTS.md`、`.agents/rules/`、`.agents/skills/`；全局 `~/.gemini/GEMINI.md` | 2.0、CLI、IDE 的全局 Skills 位置不同；旧 `.agents/workflows/` 已迁移为 Skills；本规则不可直接推断独立 Gemini CLI 行为。不额外建 `.agents/rules/` 重复副本，直接复用根 `AGENTS.md` 与 `.agents/skills/` | 规则与 Skills 写入后须在新会话复核自动发现与加载 |
| **VS Code Copilot** | **实际使用**（仓库已有 `.vscode/tasks.json`、`.github/copilot-instructions.md`、`.github/instructions/gotest.instructions.md`） | 根 `AGENTS.md`、`.github/copilot-instructions.md`；Skills 支持 `.github/skills/`、`.claude/skills/`、`.agents/skills/`；自定义 Agent 为 `.github/agents/*.agent.md` | 嵌套 `AGENTS.md` 仍属实验特性（受 `chat.useNestedAgentsMdFiles` 控制）；自定义 Agent 的 `infer` 字段已弃用；区分 Local agent、Agent Host 及其他 harness。保留 `.github/copilot-instructions.md` 与 `.github/instructions/gotest.instructions.md`，并保留 `.github/skills/` 兼容路径以防破坏子模块 `libatbus-go/CLAUDE.md` 的引用 | 静态引用已检查；VS Code Copilot 实际会话发现未在本次终端环境运行验收 |
| **Kilo Code** | **本机已安装**（`kilo` `7.4.21`，`C:\Users\owt50\AppData\Local\pnpm\bin\kilo.ps1`） | `AGENTS.md`（`AGENT.md` 备选）；`.kilo/skills/` 与 `.agents/skills/` | 官方区分 VS Code 与 CLI 行为；Claude Skills 兼容受开关控制；部分可信位置 Skill 支持嵌入 shell，安全审计不能只查 `scripts/` 目录；未建 `.kilo/` 专属目录 | **已运行验证（Skills 发现）**：2026-11-25 在仓库根与 `src/lobbysvr` 执行 `kilo debug skill`（exit=0），均发现全部 6 个 `.agents/skills/*/SKILL.md`；规则交互式调用与权限禁用仍待会话实测 |
| **Pi** | **本机已安装**（`pi` `1.6.0`，`C:\Users\owt50\scoop\shims\pi.exe`） | `AGENTS.md`、同目录 `AGENTS.override.md`；`.pi/skills/`、`~/.pi/agent/skills/`、`.agents/skills/` | 项目配置与 Skills 受 project trust 控制；context 文件发现不等于工具沙箱；修改 `.pi/SYSTEM.md` 或 `APPEND_SYSTEM.md` 前须先查影响。当前不建 `.pi/` 目录 | 本机 CLI 版本已核验；交互式会话与 project trust 加载未运行验收 |
| **Oh My Pi（`omp`）** | **本机已安装**（`omp` `19.1.1`，`C:\Users\owt50\scoop\shims\omp.exe`） | 共享 `AGENTS.md` 与多来源 Skills；原生 `.omp/AGENTS.md`、`.omp/RULES.md` | 原生发现停在最近的非空 `.omp/` 目录（缺少某文件时不会继续向上查找）；外部用户级来源可能需 opt-in；`RULES.md` 常驻、`skill://` 按需读取。为避免遮蔽根 `AGENTS.md`，当前不建 `.omp/` 目录 | 本机 CLI 版本已核验；交互式会话加载未运行验收 |
| **Claude Code** | **团队兼容支持**（本机未安装 `claude` CLI；子模块 `atframework/libatbus-go` 已配置 `CLAUDE.md`） | `CLAUDE.md`；条件原生读取 `AGENTS.md`；`.claude/rules/`、`.claude/skills/`、`.claude/agents/` | 原生 `AGENTS.md` 需 v2.1.277+ 且受内置插件、feature flags 与会话配置影响；若存在 `CLAUDE.md`/`.claude/CLAUDE.md`/`CLAUDE.local.md` 默认优先采用。当前保留最小 `CLAUDE.md`（`@AGENTS.md`），暂不建 `.claude/`；若团队需 Claude 自动发现 Skills 且当前版本不读 `.agents/skills/`，再评估受控同步至 `.claude/skills/` | 静态格式已检查；本机无 `claude` 客户端，`/context` 加载验收未执行 |
| **Codex** | **标准兼容支持**（本机未安装 `codex` CLI；不建 `.codex/`） | `AGENTS.md`；仓库与用户级 `.agents/skills/`；自定义 Agent 为 `.codex/agents/*.toml` | 规则按项目根到 cwd 拼接，每目录 `AGENTS.override.md` 优先于 `AGENTS.md`，默认合并上限 32 KiB；Skills 从 cwd 向仓库根发现，同名不自动合并；隐式调用控制位于 `agents/openai.yaml` 的 `policy.allow_implicit_invocation` | 本机未安装，运行未验证 |
| **OpenCode** | **当前未使用**（本机未安装；不建 `.opencode/`） | `AGENTS.md`；`.opencode/agents/*.md`；Skills 支持 `.opencode/skills/`、`.claude/skills/`、`.agents/skills/` | 项目 Skills 向上发现至 Git worktree 边界；规则回退与权限按 OpenCode 官方文档核对 | 不适用当前启用验收（未安装） |
| **Command Code** | **当前未使用**（本机未安装；不建 `.commandcode/`） | `AGENTS.md`；`.commandcode/skills/` 与 `.agents/skills/`；`.commandcode/agents/` | Skills 同名优先级为项目专属 > 项目共享 > 用户专属 > 用户共享；该顺序不可套用到自定义 Agent 同名冲突 | 不适用当前启用验收（未安装） |
| **Zoo Code** | **当前未使用**（本机未安装；不建 `.roo/`） | 根 `AGENTS.md`（`AGENT.md` 备选）；`.roo/rules/`、`.roo/rules-<mode>/` | 当前官方文档仍使用 `.roo/` 与 `roo-cline.useAgentRules`；不要臆造 `.zoo/` 目录，也不推断共享 Skills 自动发现 | 不适用当前启用验收（未安装） |
| **OpenClaw** | **当前未使用**（本机未安装；不建专属目录） | workspace `AGENTS.md`；workspace `skills/`、`.agents/skills/` 等 | 文件型 Skills 前四级为 workspace skills、项目 agent skills、个人 agent skills、state-dir skills（个人共享目录仅默认 state 生效）；library/workshop/执行 workspace 与 allowlist 另有规则 | 不适用当前启用验收（未安装） |
| **Hermes Agent** | **当前未使用**（本机未安装；不建 `.hermes/`） | `.hermes.md`/`HERMES.md`、`AGENTS.override.md`、`AGENTS.md`、`CLAUDE.md`（first-match） | 项目 context 类型取首次命中；Skills 支持 Git 根的 `.hermes/skills/`、`.agents/skills/` 及 `~/.hermes/skills/`；项目信任与外部目录配置启用前需复核 | 不适用当前启用验收（未安装） |
| **Devin Desktop** | **当前未使用**（本机未安装；不建 `.devin/`） | `AGENTS.md`；`.devin/rules/`；首选 `.devin/skills/`，兼容 `.agents/skills/` | `.windsurf/rules/`、`.windsurf/skills/` 为兼容位置；Devin Local 与 legacy Cascade 记忆行为不同，不可泛化 | 不适用当前启用验收（未安装） |

### 兼容验收最低检查项（启用任意客户端前）

1. 分别从仓库根目录与代表性子目录（如 `src/lobbysvr`）启动客户端。
2. 核查预期规则文件（`AGENTS.md` / 兼容层）是否实际进入上下文，未被本地未跟踪文件遮蔽。
3. 核查同名规则或同名 Skill 冲突时的实际生效顺序。
4. 核查 `.agents/skills/` 下各 Skill 是否出现在发现列表，并能用无副作用请求正常读取调用。
5. 若配置了禁用项或权限限制，实测确认执行层拒绝/禁用确实生效。仅确认文件存在于磁盘不算兼容通过。

## 3. 未创建专属工具目录的依据

本仓库当前仅维护通用入口 `AGENTS.md`、最小兼容层 `CLAUDE.md`、已有 `.github/` 兼容文件与 `.agents/skills/`。以下目录**按需不创建**：

- `.codex/`、`.claude/`、`.omp/`、`.pi/`、`.roo/`、`.kilo/`、`.opencode/`、`.commandcode/`、`.devin/`、`.hermes/`：当前无工具专属规则或自定义 Agent 需求，且避免引入目录遮蔽（如 `.omp/` 会截断向上发现）或空占位目录。
- `openspec/`、`roadmap/`、`development/`：当前仓库尚未采用 OpenSpec，也未建立对应目录；待实际立项或授权采用时再按规范初始化。

## 4. `CLAUDE.md` 兼容层维护约定

1. 当前保留根 [CLAUDE.md](../../CLAUDE.md) 仅使用 `@AGENTS.md` 导入共享规则，不复制正文。
2. 仅当确认团队全部 Claude Code 会话均运行在 `v2.1.277+`、原生 `AGENTS.md` 加载已启用且无 Claude 专属补充时，才可移除 `CLAUDE.md`。
3. 如后续需按文件路径加载 Claude 规则，使用 `.claude/rules/` 的 `paths` 配置；普通 Markdown 链接不等同于 `@path` 导入，且 `@import` 深度上限为 4 跳、在启动时直接计入上下文。
4. 在 Claude Code 中使用 `/context` 实际核查加载列表，确认共享规则未被 `CLAUDE.local.md` 或 `.claude/CLAUDE.md` 遮蔽。

## 5. 自定义 Agent（子代理）治理规则

当前 `atsf4g-go` **暂未创建**仓库级自定义 Agent 配置文件（如 `.github/agents/*.agent.md`、`.claude/agents/*.md`、`.codex/agents/*.toml`），原因是现有任务通过 `AGENTS.md` + 6 个专业 Agent Skills 即可覆盖，且未出现稳定的跨会话工具隔离/交接需求。

若后续确需引入独立 Agent 角色，必须满足以下约束：

1. **按需创建**：仅在任务需要稳定职责边界、强制工具限制、独立上下文或明确 handoff 时创建；一次性分工不固化为永久 Agent。
2. **严格核对客户端 schema**：Markdown/YAML 与 TOML 配置不可互换；`model`、`tools`、`permissions`、`mode`、`handoff`、`isolation` 在不同客户端的同名字段不一定同义，创建前必须对照对应客户端官方文档。
3. **职责精简**：角色描述只写输入、输出、完成标准、允许修改范围、失败返回与升级条件，不重复 `AGENTS.md` 中的共享工程规范。
4. **真实只读隔离**：规划与代码审查角色优先使用运行层只读工具配置；只要授予了 shell 或写型 MCP 工具，就不能仅凭提示词将其称为“只读”。
5. **验证继承与隔离边界**：逐客户端核实子代理是否继承父规则、Skills、权限和对话上下文；上下文继承不等于文件系统隔离。
6. **并行委派与所有权**：仅在存在独立子任务且当前会话授权委派时并行；明确划分文件写入所有权，隔离数据库、端口、缓存和临时目录，由主 Agent 负责整合与最终验收。
7. **模型选择纪律**：不默认硬编码最新或最贵模型；按任务复杂度、预算和已验证能力选择，不擅自覆盖用户显式指定的模型。
