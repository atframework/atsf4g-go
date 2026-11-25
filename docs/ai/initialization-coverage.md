# `atsf4g-go` AI 工程初始化与维护覆盖表

本文件是 `atsf4g-go` 仓库唯一的权威 AI 工程维护覆盖记录（更新日期：`2026-11-25`）。它用于核对初始化提示词（至 `AI-INIT-PROMPT-END`）、用户补充要求及仓库既有约束的落实与验收情况，不写入常驻 `AGENTS.md`。

- **输入核对**：已完整读取初始化提示词全文至结束标记 `AI-INIT-PROMPT-END`，以及用户补充要求与本地参考文档（原路径不在目标仓库内，已完整内化至 `.agents/skills/docs-and-comments-writing/`，无失效外链）。
- **状态定义**（仅使用以下 4 种）：`已覆盖`、`不适用`、`待处理`、`阻塞`。
- **汇总原则**：规则落地与运行验收分项记录；任一父章下存在 `待处理` 或 `阻塞` 子项时，该父章不标为 `已覆盖`。

## 覆盖统计汇总

- **交付结论**：**部分完成**（全部规则、Skills、参考文档、静态结构检查与 `kilo` Skills 发现验收已完成；受限于本机未安装 `claude`/`skills-ref`/`markdownlint-cli2`/`golangci-lint` 及交互式客户端/模型运行评测环境，部分运行验收子项保持 `阻塞`）。
- **章节核对总览（24 个模板核对章节 `C00`–`C23` + 2 个扩展项 `U01`、`R01`，合计 26 个父项）**：
  - 模板基线章节（`C00`–`C23`，共 24 章）：已核对 `24/24`；其中父项状态为 `已覆盖` 17 章（`C01`–`C05`、`C07`、`C11`–`C14`、`C16`–`C20`、`C22`–`C23`）、`阻塞` 7 章（`C00`、`C06`、`C08`、`C09`、`C10`、`C15`、`C21`）、`待处理` 0 章、`不适用` 0 章。
  - 扩展父项（`U01`、`R01`，共 2 项）：已核对 `2/2`；均为 `已覆盖`。
- **独立可验收项统计（不重复计数父章，合计 51 项：第 2 节 49 项子项 + 第 3 节 2 项扩展项）**：
  - `已覆盖`：40 项（第 2 节 38 项 + 第 3 节 `U01`、`R01` 2 项）
  - `不适用`：4 项（`C06.4`、`C11.2`、`C12.3`、`C13.2`）
  - `待处理`：0 项
  - `阻塞`：7 项（`C00.2`、`C06.3`、`C08.2`、`C09.3`、`C10.2`、`C15.3`、`C21.2`）
  - 未映射要求：`0`

## 1. 模板核对目录父项汇总（`C00`–`C23`）

| ID / 来源章节 | 适用性与依据 | 实际落点（文件 + 章节） | 状态 | 核验证据或缺口与下一步 |
| --- | --- | --- | --- | --- |
| `C00` / 完整覆盖与交付约定 | 适用；本次为仓库首次全面 AI 工程初始化 | `docs/ai/initialization-coverage.md`（全文）；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §1、异常与恢复、验证与交付） | 阻塞 | `C00.1` 已覆盖；因存在 6 项运行验收阻塞（见 `C00.2`），按完成条件汇总为 `阻塞` 并报告部分完成 |
| `C01` / 目标 | 适用；建立共享规则、开放标准 Skills、兼容层与可追溯维护 | `AGENTS.md`（全文）；`CLAUDE.md`；`.github/copilot-instructions.md`；`.agents/skills/README.md`；`docs/ai/*` | 已覆盖 | `C01.1`–`C01.2` 均已覆盖并回读验证 |
| `C02` / 不可违反的原则 | 适用；全部 10 条原则均适用于本仓库 | `AGENTS.md`（边界与变更流程、工具与执行约定、完成与同步检查）；`docs/ai/engineering-governance.md`；`.agents/skills/cli-and-shell-conventions/SKILL.md` | 已覆盖 | `C02.1`–`C02.2` 均已逐条映射并回读核验 |
| `C03` / 调研和实时更新流程 | 适用；需记录核验流程与 6 字段外部来源索引 | `docs/ai/source-index.md`（全文）；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §1） | 已覆盖 | `C03.1`–`C03.2` 均已覆盖，6 字段齐备且含 2026-11-25 复核记录 |
| `C04` / 推荐产物结构 | 适用；建立最小必要入口与 `.agents/skills/`，不建空占位目录 | `AGENTS.md`；`CLAUDE.md`；`.agents/skills/README.md`；`docs/ai/client-compatibility.md`（§3 未创建专属工具目录的依据） | 已覆盖 | `C04.1`–`C04.2` 均已覆盖，未创建多余工具占位目录 |
| `C05` / 文件选择决策表 | 适用；明确各类需求承载位置与 VS Code prompt files 边界 | `docs/ai/client-compatibility.md`（§1 文件选择决策表）；`.github/copilot-instructions.md`（按需读取） | 已覆盖 | `C05.1` 已覆盖并回读核验 |
| `C06` / 跨工具兼容规则 | 适用；需逐一登记 13 客户端状态、边界并执行可用验收 | `docs/ai/client-compatibility.md`（§2 跨客户端兼容矩阵、兼容验收最低检查项）；`docs/ai/source-index.md` | 阻塞 | `C06.1`–`C06.2` 已覆盖，`C06.4` 不适用；`C06.3` 因其余已装/团队客户端需交互式会话实测而处于 `阻塞` |
| `C07` / `AGENTS.md` 编写规则 | 适用；建立根目录共享规则主入口与 6 节骨架 | `AGENTS.md`（项目与范围、规则入口与按需读取、开发构建与验证、工具与执行约定、边界与变更流程、完成与同步检查） | 已覆盖 | `C07.1`–`C07.2` 已覆盖，文件大小 12,440 字节（远低于 Codex 32 KiB 上限） |
| `C08` / `CLAUDE.md` 兼容规则 | 适用；保留最小兼容入口供团队及与子模块习惯一致 | `CLAUDE.md`（全文）；`docs/ai/client-compatibility.md`（§4 `CLAUDE.md` 兼容层维护约定） | 阻塞 | `C08.1` 已覆盖；`C08.2` 因本机未安装 `claude` CLI 无法运行 `/context` 验收而处于 `阻塞` |
| `C09` / Agent Skills 编写规则 | 适用；按开放标准建立 6 个项目 Skills 及按需参考 | `.agents/skills/README.md`；6 个 `.agents/skills/*/SKILL.md` 及 2 个 `references/*.md`；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §2） | 阻塞 | `C09.1`–`C09.2` 已覆盖（`yq` + Python 全量通过）；`C09.3` 因本机未安装 `skills-ref` 处于 `阻塞` |
| `C10` / Skill 触发质量评估 | 适用；本次新建 6 个标准 Skills，须落规则并诚实记录评估状态 | `.agents/skills/ai-maintenance/SKILL.md`（工作流 §3 Skill 触发与任务质量评估） | 阻塞 | `C10.1` 已覆盖；`C10.2` 因未在真实模型 harness 跑触发与对照评测处于 `阻塞`（标明“静态检查通过，运行评估未执行”） |
| `C11` / 自定义 Agent 规则 | 适用；评估不创建永久角色并记录 schema/权限/隔离规则 | `docs/ai/client-compatibility.md`（§5 自定义 Agent 治理规则）；`AGENTS.md`（边界与变更流程） | 已覆盖 | `C11.1` 已覆盖；`C11.2`（实际创建自定义 Agent 文件）标为 `不适用` |
| `C12` / 任务分流与新增功能工作流 | 适用；建立分级流程，确认未采用 OpenSpec/Superpowers 及替代流程 | `AGENTS.md`（边界与变更流程）；`docs/ai/engineering-governance.md`（§1 任务分流与新增功能工作流） | 已覆盖 | `C12.1`–`C12.2` 已覆盖；`C12.3`（初始化 `openspec/`）标为 `不适用` |
| `C13` / MCP 接入规则 | 适用；确认未配置 MCP 并记录 `2026-07-28` 协议与安全要求 | `docs/ai/engineering-governance.md`（§2 MCP 接入与安全规则）；`docs/ai/source-index.md` | 已覆盖 | `C13.1` 已覆盖；`C13.2`（实际接入项目 MCP）标为 `不适用` |
| `C14` / 本地调试、部署和秘密管理 | 适用；约定 `build/<task-name>/`、凭据、CI/CD、Helm 部署与清理边界 | `docs/ai/engineering-governance.md`（§3 本地调试、凭据管理与部署准备）；`AGENTS.md`（工具与执行约定、边界与变更流程） | 已覆盖 | `C14.1`–`C14.2` 已覆盖，并核对 `Jenkinsfile` 与 `install/cloud-native/` 现状 |
| `C15` / 测试、lint 和质量门禁 | 适用；沉淀 Go 测试/回归/lint/Markdown/图表门禁并执行可用验证 | `.agents/skills/go-testing-and-lint/SKILL.md`；`AGENTS.md`（开发构建与验证、完成与同步检查）；`.github/instructions/gotest.instructions.md` | 阻塞 | `C15.1`–`C15.2` 已覆盖（`logical_time` 单测与本地 Markdown 校验通过）；`C15.3` 因缺 `markdownlint-cli2`/`golangci-lint` 及未执行 codegen 处于 `阻塞` |
| `C16` / 命令超时和重试 | 适用；区分等待/超时/总截止，落实有限重试、幂等与非交互约束 | `.agents/skills/cli-and-shell-conventions/SKILL.md`（工作流 §4）；`AGENTS.md`（工具与执行约定） | 已覆盖 | `C16.1` 已覆盖并参考 `Jenkinsfile`/`.golangci.yaml` 实际超时预算 |
| `C17` / 终端环境与命令行工具 | 适用；保留优先提示、完整 30 项输入清单、本机探测、跨环境 Git 与 Windows 规则 | `AGENTS.md`（工具与执行约定）；`.agents/skills/cli-and-shell-conventions/SKILL.md`；`.agents/skills/cli-and-shell-conventions/references/modern-cli-catalog.md` | 已覆盖 | `C17.1`–`C17.4` 全部已覆盖；已实测 30 项工具在 Windows 本机的可用性（4 项已装、26 项缺失回退） |
| `C18` / 文档、路线图和执行计划 | 适用；理清 `doc/` 与 `docs/ai/` 分层、ADR/计划分工及长任务交接 | `docs/ai/engineering-governance.md`（§4 文档、路线图、ADR 与执行计划协同）；`AGENTS.md`（项目与范围、完成与同步检查） | 已覆盖 | `C18.1` 已覆盖，保护现有 `doc/` 结构且不另造平行计划系统 |
| `C19` / 自我改进机制 | 适用；建立根因分析沉淀规则与 ClawHub 公开 API/审计边界 | `docs/ai/engineering-governance.md`（§5 自我改进与第三方 Skill 审计机制）；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §4） | 已覆盖 | `C19.1`–`C19.2` 已覆盖，未盲目采纳或安装第三方 Skill |
| `C20` / 任务完成前检查清单 | 适用；固化完成前检查表与 4 段交付格式并在本次回复执行 | `.agents/skills/ai-maintenance/SKILL.md`（验证与交付）；本次最终交付报告 | 已覆盖 | `C20.1` 已覆盖，完成前逐项核对真实结果 |
| `C21` / 首次初始化建议 | 适用；按 6 步顺序执行审查、核验、落地、不越权集成、验证与交付 | 全部本次初始化产物与本覆盖表 | 阻塞 | `C21.1` 已覆盖；`C21.2` 受本机缺工具与交互会话限制处于 `阻塞`，按规则报告部分完成 |
| `C22` / 参考来源 | 适用；为实际采用结论保留核验日期、版本与适用范围 | `docs/ai/source-index.md`（全文）；`.agents/skills/docs-and-comments-writing/references/writing-rules-detail.md`（§6.2） | 已覆盖 | `C22.1` 已覆盖，仅保留实际采用与兼容判断相关来源 |
| `C23` / 维护提示 | 适用；保留工具清单、稳定 ID、单份覆盖表及防遗漏回归要求 | `.agents/skills/ai-maintenance/SKILL.md`（工作流 §1、§4） | 已覆盖 | `C23.1` 已覆盖并回读确认 |

## 2. 展开后的独立可验收子项明细（共 49 项）

| ID / 具体要求 | 适用性与依据 | 实际落点（文件 + 章节） | 状态 | 核验证据或缺口与下一步 |
| --- | --- | --- | --- | --- |
| `C00.1` / 完整读取输入、建立单份权威覆盖表、全章展开子项、状态与分层约束、回读验收 | 适用；初始化验收合同核心要求 | `docs/ai/initialization-coverage.md`（全文）；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §1） | 已覆盖 | 已读至 `AI-INIT-PROMPT-END` 及附加要求；所有新建/修改产物均已回读与脚本核验 |
| `C00.2` / 满足全部运行验收条件以声明“初始化完成” | 适用；须区分规则落地与运行验收 | `docs/ai/initialization-coverage.md`（覆盖统计汇总） | 阻塞 | 缺失条件：`C06.3`、`C08.2`、`C09.3`、`C10.2`、`C15.3`、`C21.2` 尚未在目标客户端/缺失 CLI 上完成运行验收；下一步见各阻塞子项 |
| `C01.1` / 建立以 `AGENTS.md` 与开放标准 `.agents/skills/` 为核心的规则和技能体系 | 适用；目标核心产物 | `AGENTS.md`；`.agents/skills/README.md`；6 个 `.agents/skills/*/SKILL.md` | 已覆盖 | 全部文件已创建并通过 `yq` 与结构校验 |
| `C01.2` / 按实际使用情况兼容目标客户端并保持决策与验证可追溯 | 适用；仓库已有 `.github/` 且本机装有 `kilo`/`pi`/`omp`/Antigravity | `CLAUDE.md`；`.github/copilot-instructions.md`；`docs/ai/client-compatibility.md`；`docs/ai/source-index.md` | 已覆盖 | 未为未用客户端建占位目录；核验日期与实测结果均已落盘 |
| `C02.1` / 保留证据先行、授权边界、保护现状与子模块、单一事实源、精简上下文 5 项原则 | 适用；核心工程约束 | `AGENTS.md`（项目与范围、规则入口与按需读取、边界与变更流程）；`docs/ai/client-compatibility.md`（§1–§3） | 已覆盖 | 回读确认；`git status` 与 `git submodule status` 证实零破坏现有修改或子模块 |
| `C02.2` / 保留能力发现先行、验证与风险匹配、数据与指令隔离、可追溯性、现代工具优先 5 项原则 | 适用；安全与执行约束 | `AGENTS.md`（工具与执行约定、边界与变更流程、完成与同步检查）；`.agents/skills/cli-and-shell-conventions/SKILL.md` | 已覆盖 | 回读确认；本次未编造任何工具名或跳过核验 |
| `C03.1` / 固化现状读取、官方一手核验、方案与同步更新流程 | 适用；持续维护流程 | `.agents/skills/ai-maintenance/SKILL.md`（工作流 §1） | 已覆盖 | 回读确认，已在本轮按该流程先核源码再写文档 |
| `C03.2` / 建立含完整 6 字段（`claim/scope`、`source_url/version`、`verified_at/method`、`installed_version/status`、`review_cadence/trigger`、`impact/owner`）的来源索引 | 适用；易变外部资料追踪 | `docs/ai/source-index.md`（全文表格） | 已覆盖 | 已核对 17 条来源记录均含全部 6 字段，标注 `rolling`/版本及 `2026-11-25` 复核日期 |
| `C04.1` / 采用 `AGENTS.md`、条件 `CLAUDE.md`、`.agents/skills/README.md` 与 `<skill>/SKILL.md` + 按需 `references/` 推荐布局 | 适用；本仓库需管理 6 个 Skills 与 Claude 兼容 | `AGENTS.md`；`CLAUDE.md`；`.agents/skills/README.md`；`.agents/skills/*/` | 已覆盖 | 实际目录与文件结构已核验，无空 Skill 或空目录 |
| `C04.2` / 记录不创建 `.codex/`、`.claude/`、`.omp/`、`.pi/`、`.roo/`、`.kilo/`、`.opencode/`、`.commandcode/`、`.devin/`、`.hermes/`、`openspec/`、`roadmap/`、`development/` 的依据 | 适用；防止占位目录污染与 `.omp/` 遮蔽 | `docs/ai/client-compatibility.md`（§3 未创建专属工具目录的依据） | 已覆盖 | 已核查仓库根目录不含上述未授权目录 |
| `C05.1` / 建立文件选择决策表并明示 VS Code Agent Host 弃用 `.github/prompts/`（Local agent 暂可用）与执行层限制边界 | 适用；指导后续规则归位 | `docs/ai/client-compatibility.md`（§1 文件选择决策表）；`.github/copilot-instructions.md`（按需读取） | 已覆盖 | 回读确认覆盖全部 11 类需求场景及边界 |
| `C06.1` / 完整登记全部 13 个客户端的规则/Skills 入口、覆盖边界、本仓库使用情况与兼容验收步骤 | 适用；跨工具兼容基线 | `docs/ai/client-compatibility.md`（§2 跨客户端兼容矩阵、兼容验收最低检查项） | 已覆盖 | Python 校验确认 13/13 客户端齐全，边界准确 |
| `C06.2` / 已安装 `kilo 7.4.21` 从仓库根与 `src/lobbysvr` 执行无副作用 Skills 发现验收 | 适用；本机已装 `kilo` CLI 且提供只读 `kilo debug skill` | `docs/ai/client-compatibility.md`（§2 Kilo Code 行）；`docs/ai/source-index.md` | 已覆盖 | 2026-11-25 实测：仓库根与 `src/lobbysvr` 运行 `kilo debug skill` 均 exit=0，均发现 6/6 `.agents/skills/*/SKILL.md` |
| `C06.3` / Antigravity 新会话、VS Code Copilot、Pi `1.6.0`、Oh My Pi `19.1.1`、团队 Claude Code 的实时规则/Skills 加载与禁用/权限验收 | 适用；这些客户端为本仓库实际使用/已装或团队兼容目标 | `docs/ai/client-compatibility.md`（§2 对应客户端行） | 阻塞 | 缺失条件：本机无 `claude` CLI；Antigravity 新会话、VS Code Copilot UI、`pi`/`omp` 交互式会话无法在当前无交互命令行完成真实加载验收。下一步：在各客户端从仓库根与 `src/lobbysvr` 启动会话并用无副作用请求核验 |
| `C06.4` / 未启用客户端（Codex、OpenCode、Command Code、Zoo Code、OpenClaw、Hermes Agent、Devin Desktop）的实机运行验收 | 不适用；本机均未安装且仓库未启用其专属配置 | `docs/ai/client-compatibility.md`（§2 对应 7 行） | 不适用 | 依据：本机 CLI 探测均为 MISSING，仓库无对应工具目录；待团队启用任一客户端前按 §2 检查项补验 |
| `C07.1` / 按 6 节骨架编写精简根 `AGENTS.md`，包含已核验项目范围、命令/前提、边界、按需读取与完成检查 | 适用；共享规则主入口 | `AGENTS.md`（全部 6 节） | 已覆盖 | 已回读全文；所有路径、Go 模块、Taskfile 命令与符号均对照源码/配置核实 |
| `C07.2` / 避免在 `AGENTS.md` 复制完整目录树或全文参考，控制常驻体积 | 适用；上下文精简与 Codex 32 KiB 限制 | `AGENTS.md`（全文） | 已覆盖 | 实测 `AGENTS.md` 为 84 行 / 12,440 字节，长清单与治理细则已分层至 Skills 与 `docs/ai/` |
| `C08.1` / 创建最小 `CLAUDE.md`（`@AGENTS.md` 导入），写明移除条件、4 跳上限、`.claude/rules/` `paths` 与 `/context` 核查方式 | 适用；团队兼容且与子模块 `libatbus-go/CLAUDE.md` 一致 | `CLAUDE.md`（全文）；`docs/ai/client-compatibility.md`（§4） | 已覆盖 | 已回读确认仅 9 行，未复制 `AGENTS.md` 正文 |
| `C08.2` / 在 Claude Code 会话中使用 `/context` 实际核验 `CLAUDE.md` 与 `AGENTS.md` 加载且未被遮蔽 | 适用；`C08` 要求的加载检查 | `CLAUDE.md`；`docs/ai/client-compatibility.md`（§4） | 阻塞 | 缺失条件：本机未安装 `claude` CLI。下一步：团队成员在 Claude Code 会话执行 `/context` 并更新本表与 `source-index.md` |
| `C09.1` / 按开放标准在 `.agents/skills/` 编写 6 个 Skill（必填/可选字段、描述边界、`<500` 行、直链资源、模板 6 节）并固化脚本合同、调用策略与供应链规则 | 适用；替代旧扁平 `.github/skills/*.md` 并承接项目与写作/CLI/维护流程 | `.agents/skills/README.md`；6 个 `SKILL.md` 及 2 个 `references/*.md`；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §2） | 已覆盖 | 已回读全部 6 个 `SKILL.md`（60–89 行），无占位文本、无死链 |
| `C09.2` / 对全部 `.agents/skills/*/SKILL.md` 执行 YAML frontmatter 解析、命名/长度约束与结构校验 | 适用；静态规范检查 | 全部 6 个 `.agents/skills/*/SKILL.md` | 已覆盖 | `yq --front-matter=extract`（6/6 OK）+ Python 规范校验 script（exit=0，0 errors） |
| `C09.3` / 运行 `skills-ref validate <skill-dir>` 检查全部 6 个 Skill | 适用；模板指定的官方校验工具 | `.agents/skills/*` | 阻塞 | 缺失条件：本机未安装 `skills-ref` CLI（`Get-Command` 返回 MISSING），按授权不擅自安装。下一步：安装 `skills-ref` 后对 6 个目录逐一运行校验 |
| `C10.1` / 沉淀 Skill 触发集合、真实路由、基线输出对照、隔离/留出/安全样本与诚实记录规则 | 适用；Skill 持续维护要求 | `.agents/skills/ai-maintenance/SKILL.md`（工作流 §3） | 已覆盖 | 已回读确认包含 20 条查询/3 次/0.5/60-40 起点说明与诚实记录要求 |
| `C10.2` / 对本次新建的 6 个 Skill 执行真实 harness 触发集路由与 2–3 任务基线对照质量评估 | 适用；本次属于实质新增 Skill 描述与工作流 | `.agents/skills/*`；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §3） | 阻塞 | 缺失条件：当前无自动化多模型/多客户端评测 harness 执行 20 条查询×多次路由及对照任务；按 `C10` 标为**“静态检查通过，运行评估未执行”**。下一步：在目标客户端上针对 6 个 Skill 构造触发集与对照任务补测 |
| `C11.1` / 固化自定义 Agent 创建时机、schema 不可互换、真实只读、继承隔离、并行所有权与模型选择规则 | 适用；未来角色扩展治理 | `docs/ai/client-compatibility.md`（§5 自定义 Agent 治理规则） | 已覆盖 | 已回读确认覆盖 `C11` 全部 7 项要点 |
| `C11.2` / 创建仓库级自定义 Agent 配置文件（如 `.github/agents/`、`.codex/agents/`） | 不适用；当前任务无稳定独立角色或跨会话工具隔离需求 | `docs/ai/client-compatibility.md`（§5） | 不适用 | 依据：6 个 Skills 足以承接现有工作流，不为一次性分工建常驻角色 |
| `C12.1` / 建立任务分级表及 `atsf4g-go` 现行设计/变更替代流程 | 适用；指导日常与跨模块变更 | `AGENTS.md`（边界与变更流程）；`docs/ai/engineering-governance.md`（§1.1–§1.2） | 已覆盖 | 已回读确认覆盖全部 5 类任务分级与完成证据要求 |
| `C12.2` / 核实 OpenSpec/Superpowers 未安装与未采用状态，并记录未来获授权采用时的强制执行约定 | 适用；避免虚假宣称或越权启用 | `docs/ai/engineering-governance.md`（§1.2–§1.3）；`docs/ai/source-index.md` | 已覆盖 | 实测 `openspec` 为 MISSING 且仓库无 `openspec/`；`v1.13.1`/`v6.4.1` 关键边界与归档门禁已记录 |
| `C12.3` / 初始化 `openspec/` 目录或创建变更 change | 不适用；仓库未采用 OpenSpec 且未获授权初始化 | `docs/ai/engineering-governance.md`（§1.2）；`docs/ai/client-compatibility.md`（§3） | 不适用 | 依据：按 `C04`/`C12`/`C21` 仅在项目决定采用并授权时初始化，且不为未知功能建 change |
| `C13.1` / 记录 MCP `2026-07-28` 无状态基线、OAuth/STDIO 认证区分、传输/Origin/进程清理、输入校验与凭据隔离规则 | 适用；未来接入 MCP 的强制安全规范 | `docs/ai/engineering-governance.md`（§2 MCP 接入与安全规则）；`docs/ai/source-index.md` | 已覆盖 | 已复核 `2026-07-28` 为官网 current；回读确认禁止 token passthrough、annotations 不作授权依据等全部条款 |
| `C13.2` / 配置仓库级 MCP 服务器或 `.mcp.json` | 不适用；当前仓库无项目级 MCP 接入需求与授权 | `docs/ai/engineering-governance.md`（§2） | 不适用 | 依据：已检索仓库无现有 MCP 配置，不擅自接入外部服务 |
| `C14.1` / 约定 `build/<task-name>/` 临时产物、`development/secret/` 安全边界、凭据保护、`.env.example` 与安全清理规则 | 适用；仓库已有 `build/` 与 `.env` 忽略项 | `docs/ai/engineering-governance.md`（§3.1–§3.2）；`AGENTS.md`（工具与执行约定、边界与变更流程） | 已覆盖 | 回读确认覆盖 `jq`/`yq` 非安全通道、不粘贴秘密、递归删除前核验路径与符号链接 |
| `C14.2` / 固化 CI/CD 凭据与 GitHub Actions 完整 commit SHA 固定规则，以及 Helm/部署准备与环境验收分离边界 | 适用；仓库已有 `Jenkinsfile`、`install/cloud-native/` 与 `script/` | `docs/ai/engineering-governance.md`（§3.2–§3.3）；`AGENTS.md`（完成与同步检查） | 已覆盖 | 已核对 `Jenkinsfile` 中 `FEISHU_PROJECT_Y_URL` 及 Helm/脚本路径并写入规则 |
| `C15.1` / 沉淀 Go 测试、回归、mock/集成边界、`.golangci.yaml`、`tools/lint-runner` 差异、Markdown 与图表质量门禁 | 适用；仓库为多模块 Go 项目且含 MkDocs 图表文档 | `.agents/skills/go-testing-and-lint/SKILL.md`；`AGENTS.md`（开发构建与验证、完成与同步检查） | 已覆盖 | 已核实 `tools/lint-runner` 忽略退出码且跳过无 `go.mod` 子目录的真实行为并写入提醒 |
| `C15.2` / 运行无需 codegen 的代表性 Go 单测及全部本次文档/Skill 确定性结构校验 | 适用；核验既有测试基线与本次纯文档/配置变更质量 | `.agents/skills/go-testing-and-lint/SKILL.md`；全部本次产物 | 已覆盖 | `src/component`: `go test -v ./logical_time/...` exit=0（7 PASS, 0.364s）；Python 结构校验（含 links/fences/headings/CRLF/UTF-8/frontmatter）+ `yq` 均 exit=0 |
| `C15.3` / 运行 `markdownlint-cli2`、`golangci-lint` 及 `src/lobbysvr` 全量测试 | 适用；对应外部 lint 与需 codegen 的模块测试入口 | `.agents/skills/go-testing-and-lint/SKILL.md`；`AGENTS.md`（开发构建与验证） | 阻塞 | 缺失条件：本机未安装 `markdownlint-cli2` 与 `golangci-lint`（均为 MISSING），且本次未修改 Go 源码也未触发 `task build` 生成 `*.pb.go`。下一步：安装相应工具并在需要验证 Go 行为变更时先执行 `task build:lobbysvr` 后运行测试与 lint |
| `C16.1` / 区分等待窗口/子进程超时/总截止时间，固化项目预算、脱敏诊断、有限退避重试、幂等防重放与非交互规则 | 适用；构建、下载、CI 与部署均涉及长耗时或副作用命令 | `.agents/skills/cli-and-shell-conventions/SKILL.md`（工作流 §4）；`AGENTS.md`（工具与执行约定） | 已覆盖 | 已结合 `Jenkinsfile`（`5m`/`10m`）与 `.golangci.yaml`（`5m`）落地 |
| `C17.1` / 在常驻 `AGENTS.md` 写入“尽可能优先使用现代 CLI”可执行提示及本机已装/回退说明 | 适用；确保每次会话生效 | `AGENTS.md`（工具与执行约定） | 已覆盖 | 已回读确认包含具体优先顺序、本机已装（`rg`/`jq`/`yq`/`zstd`）、回退与完整清单链接 |
| `C17.2` / 完整保留输入清单全部现代 CLI 候选（实际表格 30 项，正文误标 31 项已如实注明）与本机探测结果 | 适用；禁止压缩为工具类别概述 | `.agents/skills/cli-and-shell-conventions/references/modern-cli-catalog.md`（全文） | 已覆盖 | Python 校验确认 30/30 工具条目及链接齐全；2026-11-25 实测 4 项已装（`rg 16.0.0`、`jq 1.8.2`、`yq v4.54.3`、`zstd 1.6.0`）、26 项缺失 |
| `C17.3` / 固化 `cargo binstall` 仅二进制策略、`rg --max-count` 语义、`rg`/`jq -e` 退出码、参数防注入与跨 VM/WSL 临时 Git 服务传输规则 | 适用；跨平台与安全边界 | `.agents/skills/cli-and-shell-conventions/SKILL.md`（工作流 §1、§2、§5）；`AGENTS.md`（工具与执行约定） | 已覆盖 | 回读确认逐条完整覆盖 |
| `C17.4` / 固化 Windows PowerShell 7+ 规则（`-NoLogo -NoProfile -NonInteractive`、别名规避、引用、`-LiteralPath`、`$PSNativeCommandArgumentPassing`/`--%`、UTF-8/EOL/5.1 编码、`-ErrorAction Stop`、`$LASTEXITCODE`、隐藏窗口与 PID 清理） | 适用；当前工作站为 Windows（已装 `pwsh 7.7.0`，默认终端 5.1） | `.agents/skills/cli-and-shell-conventions/SKILL.md`（工作流 §3）；`AGENTS.md`（工具与执行约定） | 已覆盖 | 本次验证命令已实际使用 `pwsh -NoLogo -NoProfile -NonInteractive` 并保留 `$LASTEXITCODE` 检查 |
| `C18.1` / 明确 `doc/` 与 `docs/ai/` 分层、按需 README、路线图/计划/spec 职责、复用现有 `Plan.md` 与长任务交接恢复机制 | 适用；持续文档与计划协同 | `docs/ai/engineering-governance.md`（§4）；`AGENTS.md`（项目与范围、完成与同步检查） | 已覆盖 | 已核查主仓库无 `Plan.md`（仅子模块 `libatbus-go/Plan.md`），未新建多余 README 或平行计划 |
| `C19.1` / 固化纠错根因定位、沉淀优先级（测试/检查 > Skill > 局部规则 > 根规则）、规则生效/复核条件与模型升级复评 | 适用；防止规则膨胀 | `docs/ai/engineering-governance.md`（§5.1）；`.agents/skills/ai-maintenance/SKILL.md`（工作流 §4） | 已覆盖 | 回读确认逐项覆盖 |
| `C19.2` / 固化第三方 Skill 调研与 ClawHub 公开只读 API、限速/`Retry-After`、完整安全审计边界 | 适用；供应链安全约束 | `docs/ai/engineering-governance.md`（§5.2）；`docs/ai/source-index.md` | 已覆盖 | 回读确认：明确排名/下载量/`nonSuspiciousOnly` 不构成安全保证，未安装任何第三方 Skill |
| `C20.1` / 固化任务完成前检查清单与 4 段最终交付格式，并在本次交付逐项核对执行 | 适用；交付质量门禁 | `.agents/skills/ai-maintenance/SKILL.md`（验证与交付）；本轮最终回复 | 已覆盖 | 交付前逐项对照本表与实际文件核验，未将未执行项标为通过 |
| `C21.1` / 完成首次初始化前 5 步非阻塞工作（审查现状、核验来源、分层写入规则与 Skills、不越权初始化可选生态、回读与可用校验） | 适用；本次任务主流程 | 全部新建与修改产物及本覆盖表 | 已覆盖 | 已按顺序完成并保留完整证据 |
| `C21.2` / 完成首次初始化全部客户端实测发现、官方校验工具与运行评估验收 | 适用；完整初始化结束前提 | `docs/ai/initialization-coverage.md`（阻塞项明细） | 阻塞 | 缺失条件：依赖 `C06.3`、`C08.2`、`C09.3`、`C10.2`、`C15.3` 补齐；按约定本轮诚实报告“部分完成” |
| `C22.1` / 为实际采用的外部结论记录核验日期、版本范围、限制与适用影响，不机械照搬无关来源 | 适用；来源可追溯性 | `docs/ai/source-index.md`（全文）；`.agents/skills/docs-and-comments-writing/references/writing-rules-detail.md`（§6.2） | 已覆盖 | 回读确认全部采用结论均可追溯至官方或研究来源及核验日期 |
| `C23.1` / 固化维护提示（保留完整 CLI 清单与优先提示、同步单份覆盖表与稳定 ID、压缩前后逐项对比、回放历史遗漏、区分静态校验与实际遵循） | 适用；防止后续维护回退 | `.agents/skills/ai-maintenance/SKILL.md`（工作流 §1、§4） | 已覆盖 | 回读确认覆盖全部维护要求 |

## 3. 用户补充要求与仓库既有约束（扩展项 `U01`、`R01`）

| ID / 具体要求 | 适用性与依据 | 实际落点（文件 + 章节） | 状态 | 核验证据或缺口与下一步 |
| --- | --- | --- | --- | --- |
| `U01` / 参考外部写作指导笔记编写项目“文档与注释写作规则”Skill，并将必要内容完整内化进仓库产物、不留失效本地路径链接 | 适用；用户本轮明确补充要求 | `.agents/skills/docs-and-comments-writing/SKILL.md`；`.agents/skills/docs-and-comments-writing/references/writing-rules-detail.md`；`AGENTS.md`（规则入口与按需读取）；`.agents/skills/README.md`；`docs/ai/source-index.md` | 已覆盖 | 已完整分段读取原文档全部 620 行，内化简版规则、文体结构、标题与入口文案、复句辨析、24 项中文易空泛词表、空泛套话、双语规则、Go/MkDocs 代码文本保护、正反示例、检查表、研究依据（2026-10-06）与后续复核流程；校验确认 0 处外部本地路径泄漏且 `kilo debug skill` 可发现 |
| `R01` / 保护并迁移仓库既有 `.github/` 规则与源码事实（兼容子模块 `libatbus-go/CLAUDE.md` 引用、修正过时 `src/robot` 误述、核实 `UserGetModuleManager`/`InsertDirtyHandleIfNotExists`/`session_downstream_api_for_cs.go.mako` 等真实代码模式、不改动任何子模块） | 适用；保护现状与基于源码核验原则 | `.github/copilot-instructions.md`；`.github/skills/README.md`；`.agents/skills/build-and-codegen/SKILL.md`；`.agents/skills/lobbysvr-development/SKILL.md`；`.agents/skills/go-testing-and-lint/SKILL.md`；`AGENTS.md` | 已覆盖 | 保留 `.github/instructions/gotest.instructions.md` 与 `.github/skills/*.md` 全部路径未删；源码 `rg` 与实读验证全部符号；`git submodule status` 确认子模块零改动 |

## 4. 交接摘要（未完成与阻塞项）

- **唯一权威覆盖表**：`docs/ai/initialization-coverage.md`（本文件）
- **未完成 / 阻塞子项（共 7 项，含 2 项汇总验收）**：
  1. `C06.3`：在 Antigravity 新会话、VS Code Copilot、Pi（`1.6.0`）、Oh My Pi（`19.1.1`）与 Claude Code 中分别从仓库根和 `src/lobbysvr` 启动会话，执行无副作用规则/Skills 发现与权限限制验收。
  2. `C08.2`：在 Claude Code 会话中运行 `/context`，核实 `CLAUDE.md` → `@AGENTS.md` 正确加载且未被本地文件遮蔽。
  3. `C09.3`：安装 `skills-ref` 后对 `.agents/skills/` 下 6 个目录运行 `skills-ref validate <skill-dir>`。
  4. `C10.2`：对 6 个新建 Skill 在目标客户端/模型 harness 中运行触发查询集（~20 条×多次）与 2–3 个对照任务输出评估。
  5. `C15.3`：在安装 `markdownlint-cli2`、`golangci-lint`（`v2.6.0`）并执行 `task build:lobbysvr` 后，运行文档 lint、Go lint（直接在 `src/lobbysvr` 与 `src/component` 执行）与 `src/lobbysvr` 全量单测。
  6. `C00.2` / `C21.2`：上述 5 项运行验收全部通过后，更新对应子项与父章状态为 `已覆盖`，即可将初始化结论升级为“完成”。
- **授权边界**：后续执行 `skills-ref`/`markdownlint-cli2`/`golangci-lint` 安装、交互式客户端评测、`task build` 全量工具下载，或 `git commit`/`push`，需在明确授权范围内进行。
