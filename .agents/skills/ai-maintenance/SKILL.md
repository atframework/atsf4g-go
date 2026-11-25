---
name: ai-maintenance
description: "Maintain atsf4g-go AI rules, Agent Skills, compatibility layers, source index, initialization coverage matrix, Skill trigger/quality evaluation, and pre-completion audits. Do not use for routine Go feature coding or bug fixes."
license: MIT
metadata:
  owner: "atsf4g-go-maintainers"
---

# 工作结果

持续维护 `atsf4g-go` 的低冗余、可验证 AI 工程层（`AGENTS.md`、`CLAUDE.md`、`.github/copilot-instructions.md`、`.agents/skills/`、`docs/ai/`），确保每项适用要求都有真实内容、可达落点、核验来源与诚实的验证状态，防止规则膨胀、遗漏与跨工具漂移。

## 适用范围与输入

- **适用场景**：全面初始化或重构项目 AI 规则、新增或修改 `.agents/skills/`、更新跨客户端兼容矩阵、刷新 `docs/ai/source-index.md`、将反复发生的纠错根因转化为规则/检查、评估 Skill 触发与输出质量，以及交付前覆盖验收。
- **不适用场景**：普通业务功能开发、单次代码缺陷修复（普通改动不要求每次重跑整套 AI 初始化）。
- **输入前提**：
  - 完整读取当前维护提示词/变更需求、`AGENTS.md`、`docs/ai/initialization-coverage.md`（唯一权威覆盖表）、`docs/ai/source-index.md`、`docs/ai/client-compatibility.md` 与 `docs/ai/engineering-governance.md`。
  - 如工具输出或输入发生截断，必须分段补读原文，不得凭摘要猜测剩余要求。

## 工作流

### 1. 维护与防遗漏流程

1. **先列要求，再分层精简**：从非代码围栏正文展开全部要求、列表、表格、条件分支与模板项，同步登记到 `docs/ai/initialization-coverage.md`（保留稳定 ID，新增条目使用新 ID，严禁删除已有 ID 缩小待办；不新建平行覆盖表）。
2. **核查现状与官方依据**：先检查 `git status` 保护未提交改动与独立子模块；对易变外部结论查官方一手页面/发布版本，在 `docs/ai/source-index.md` 维护 6 字段记录（`claim / scope`、`source_url / source_version`、`verified_at / method`、`installed_version / status`、`review_cadence / update_trigger`、`impact / owner`），删除过时结论但保留复核日期与必要证据。
3. **按决策表分配落点**：
   - 高频硬约束与按需导航留根 `AGENTS.md`；
   - 专业流程放 `.agents/skills/<name>/SKILL.md`，长参考与全量工具清单放 `references/`（直接从 `SKILL.md` 一层指向，不制造失效路径）；
   - 若外部本地笔记或聊天附件路径不在仓库内，必须将必要规则内化写入已授权的仓库产物；
   - 保留完整现代 CLI 候选清单及“尽可能优先使用”的可执行提示，不可压缩成泛化的一句话。
4. **回读双向验收**：从覆盖表查文件章节，再从修改后的文件反查要求覆盖，核对正文实质、相对资源、加载入口与实测结果。

### 2. Agent Skill 编写、脚本合同与供应链边界

- **规范字段**：`SKILL.md` 必填 `name`（1–64 位小写 ASCII 字母、数字、连字符，与目录同名，无首尾或连续连字符）与 `description`（1–1024 字符，触发点前置，明确结果、适用场景与最近排除边界，不写“任何任务必调”）；按需填 `license`、`compatibility`（<=500 字符）、`metadata`（字符串键值）与实验性 `allowed-tools`。建议正文 `<500` 行，长资料放 `references/`。
- **必备内容**：每个 Skill 须覆盖工作结果、适用范围与输入、工作流、异常与恢复、验证与交付及按需资源（无资源时可省），不含示例填空占位文本。
- **脚本合同**（若某 Skill 添加 `scripts/`）：明确运行依赖、输入输出、工作目录、退出码、超时和副作用；提供 `--help`，无人值守不等待交互输入；可变参数显式传入，优先结构化输出；写操作按需支持 `--dry-run`、幂等或回滚。
- **调用策略区分**：人工触发、自动选择与执行授权三者分离。Claude 的 `disable-model-invocation` 属客户端扩展；Codex 隐式调用通过 `agents/openai.yaml` 的 `policy.allow_implicit_invocation: false` 控制；这些设置均**不替代**运行层 sandbox 或发布授权。
- **供应链审计**：引入或更新外部 Skill 须审计完整 bundle、frontmatter、动态 shell（如 Kilo 支持内嵌 shell）、hooks、依赖与出站网络，固定版本并复核 diff；规范格式校验通过不代表安全，也不证明所有客户端均能发现。

### 3. Skill 触发与任务质量评估

仅修改排版/错别字时不机械跑完整模型评测；对 Skill 描述或核心流程做实质修改时，同时评估**“选得对”**与**“做得好”**：

1. **触发集合**：覆盖应触发、不应触发、近似边界、口语化与隐式意图；官方建议约 20 条查询、多次运行（3 次重复、0.5 判定线、60/40 训练/留出划分仅作为实验起点，并非所有 Skill 的统一硬门槛）。
2. **真实路由证据**：在目标客户端的实际发现与调用链上记录路由轨迹、误触发和漏触发；**不得用“模型口头声称它会调用”替代真实路由证据**。
3. **输出质量对照**：选取 2–3 个代表性任务（含边界案例），在相同输入、模型、工具、版本和预算下比较“启用 Skill”与“未启用/旧版”的产物正确性、耗时与 token 消耗；每次在独立环境运行，优先确定性断言，必要时辅以人工盲评，失败样本按根因归类。
4. **留出复核与安全样本**：仅在训练集改写描述，在留出查询上复核；关键流程补权限拒绝、恶意/注入输入与副作用边界样本。
5. **诚实记录**：写明样本数、重复次数、客户端/模型版本、基线与未测项；当前无法调用真实 harness 跑评测时，明确标为**“静态检查通过，运行评估未执行”**。

### 4. 自我改进与模板回归核对

- 遵循 `docs/ai/engineering-governance.md` 第 5 节：先查根因，优先补测试/确定性检查，再改 Skill/局部规则；新规则写清触发、范围与复核条件；模型/harness 升级后对照复核旧补丁。
- 压缩或重构提示词/规则前后按独立要求逐项比对，不只看标题或行数；回放曾出现过遗漏的历史初始化场景，检查是否漏掉整章、同章子项、既有文件要求、外部资源迁移落点或缺失可选工具的回退流程。

## 异常与恢复

- **运行验收因缺客户端或工具受阻**：在 `docs/ai/initialization-coverage.md` 中将规则落地与运行验收分项登记，无法运行的验收子项保留为 `阻塞` 并注明缺失条件（父项随子项状态汇总，有待处理/阻塞子项时父项不可标 `已覆盖`）；完成其余不依赖阻塞的工作，不无限重试。
- **发现规范与本地客户端实测行为矛盾**：以本地实测版本为准记录限制，同时保留官方页面 URL 与版本差异，不强行二选一掩盖冲突。

## 验证与交付

### 任务完成前检查清单（仅勾选实际验证项；不适用注明依据）

- [ ] 已读取完整输入至结束标记；所有源章节与新增要求均已登记于 `docs/ai/initialization-coverage.md`。
- [ ] 章内列表、表格、条件分支与模板要求已逐项核对，无未映射要求。
- [ ] 每项适用要求均有真实内容、文件与章节落点；已回读全文并检查相对资源与加载入口，无空壳、失效链接或未替换示例。
- [ ] 不适用项有事实依据；待处理、阻塞与未执行验收如实保留，未靠删行凑完成。
- [ ] 范围、验收标准、实际能力与授权边界明确；外部事实带核验日期与版本范围。
- [ ] 现有修改与独立子模块未受破坏；入口、Skills 与兼容层可达，未大量复制常驻内容。
- [ ] 实质 Skill 变更已完成触发/质量评估，或如实标明未执行原因。
- [ ] 行为变更有运行验证证据；文档格式、引用、模板与路径已核查。
- [ ] 已优先使用可用且适用的现代 CLI，保留完整工具清单与平台回退规则。
- [ ] 超时、重试、凭据、子进程与临时产物符合约束；受影响文档、计划与来源索引已同步；区分本地验证与生产验收。

### 最终交付格式

1. **覆盖结果**：给出 `docs/ai/initialization-coverage.md` 路径；分别统计章节总数/已核对数与独立子项的 `已覆盖` / `不适用` / `待处理` / `阻塞` 数量（不重复计数）；存在未完成项时结论写“部分完成”。
2. **产物**：列出新增、修改与复用的关键文件及章节链接，说明客户端差异与 Skill 路由。
3. **验证**：写明实际执行的命令、目录、退出码与结果，单列未执行的运行验收。
4. **例外与下一步**：说明不适用依据、未完成/阻塞 ID、缺失条件及后续动作。

## 按需资源

- [docs/ai/initialization-coverage.md](../../../docs/ai/initialization-coverage.md)：仓库唯一权威初始化与维护覆盖表。
- [docs/ai/source-index.md](../../../docs/ai/source-index.md)：易变外部来源索引。
- [docs/ai/client-compatibility.md](../../../docs/ai/client-compatibility.md)：13 客户端兼容矩阵、文件选择决策表与自定义 Agent 规则。
- [docs/ai/engineering-governance.md](../../../docs/ai/engineering-governance.md)：任务分流、MCP、凭据/部署、文档体系与自我改进治理规则。
