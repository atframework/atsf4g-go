---
name: go-testing-and-lint
description: "Design, write, and run Go unit tests, regression tests, golangci-lint, proto formatting, and Markdown/diagram quality checks for atsf4g-go. Do not use for running full go-task code generation builds or prose-only style rewrites."
license: MIT
metadata:
  owner: "atsf4g-go-maintainers"
---

# 工作结果

为 `atsf4g-go` 的 Go 模块与文档变更提供与风险匹配的自动化测试、回归测试、静态分析（`golangci-lint` / `go vet` / `goimports`）及文档质量验证，并产出可追溯的执行证据（目录、命令、环境前提、退出码、用例数与通过/失败/未执行明细）。

## 适用范围与输入

- **适用场景**：编写或更新 `*_test.go`、为缺陷补充可复现的回归测试、按模块运行 `go test` / `-race` / Benchmark、运行 Go lint 与预提交 hook 检查、执行文档/图表及 Markdown 格式与链接核查。
- **不适用场景**：全量工具准备与 proto/xresloader 代码生成（改用 `build-and-codegen`）、纯文案文风润色（改用 `docs-and-comments-writing`）。
- **输入前提**：确认当前改动属于哪一 Go 模块（主仓库：`src/component`、`src/lobbysvr`、`tools/*`；独立子模块：`atframework/atframe-utils-go`、`atframework/libatapp-go`、`atframework/libatbus-go`），并确认改动是否依赖已生成的 `*.pb.go` 或 `src/component/config/generate_config`。

## 工作流

### 1. 遵循本仓库 Go 单元测试规范

- 完整按类别编码模板（工具函数、含外部依赖的结构体方法、接口实现、并发测试、Benchmark）以 [.github/instructions/gotest.instructions.md](../../../.github/instructions/gotest.instructions.md) 为权威源，核心要求：
  - 命名：文件 `xxx_test.go`；函数 `Test<Target><Scenario>`；基准 `Benchmark<Target>`。
  - 框架与结构：原生 `testing` + 可选 `github.com/stretchr/testify/assert`；严格采用 Arrange-Act-Assert（AAA），用例独立无状态。
  - 覆盖原则：覆盖核心分支、边界值（0/空/nil/极值）、错误返回（优先 `errors.Is`/`errors.As`，避免脆弱字符串匹配）与 `recover` panic 场景；不为凑数量重复复制实现。
  - 缺陷修复尽量先补能复现原问题的回归测试；确实无法稳定自动化时，写明原因、复现步骤与替代验证。
  - **Mock 与真实依赖边界**：单元测试用接口 mock（优先匿名/轻量结构体，参见 `src/lobbysvr/logic/mail/impl/user_mail_manager_instance_test.go`）隔离外部依赖；但 mock 不能替代真实协议、Redis/DB 迁移或进程生命周期验证，涉及此类风险时需补集成测试或明确记录验收范围。

### 2. 按模块执行测试（明确前置条件）

- 主仓库现有 `*_test.go` 入口（2026-11-25 核验）：
  - `src/component/logical_time/datetime_cache_test.go`（纯 Go，**无需先跑 codegen**，可在 `src/component` 目录直接执行 `go test -v ./logical_time/...`）。
  - `src/component/dynamiclib_caller/example_test.go`。
  - `src/lobbysvr/logic/global_mail/impl/global_mail_manager_impl_test.go` 与 `src/lobbysvr/logic/mail/impl/user_mail_manager_instance_test.go`（依赖生成的 `*.pb.go`，**必须先运行 `task build:lobbysvr` 或 `task fast-build:lobbysvr` 生成代码**后，再在 `src/lobbysvr` 目录执行 `go test ./...`）。
- 独立子模块如需单独验证，进入对应子模块目录运行 `go test ./...`（例如 `atframework/libatapp-go`、`atframework/libatbus-go`）。
- 参考 `Jenkinsfile` 合同：单包测试超时 `-timeout=5m`，外层超时 `10m`。并发改动增加 `-race` 检测（Windows 需可用的 CGO/gcc 环境；若缺少 CGO 明确报告原因，不静默跳过）。
- 注意：仓库中不存在 `src/robot/` 目录，不要对其运行测试。

### 3. 执行 Go Lint、Proto 格式与 Git Hooks 检查

- **配置权威源**：根 `.golangci.yaml`（`version: "2"`，要求 `golangci-lint` `v2.6.0`，超时 `5m`；启用 `errcheck`、`errorlint`、`wrapcheck`、`nilerr`、`nilnil`、`staticcheck`、`govet`/`shadow`、`revive`、`mnd`、`prealloc`、`goconst`、`gosec`、`cyclop<=15`、`funlen<=80/50`、`gocognit<=20`、`godot`、`misspell`、`lll<=150`、`durationcheck`、`exhaustive` 等，测试文件放宽 `goconst`/`funlen`/`cyclop`/`gocognit`/`gosec`，生成文件 `*_gen.go`/`*.pb.go` 放宽 `revive`/`gosec`/`goconst`）。
- **已核验的 `task lint` 行为注意点（切勿误判）**：
  1. 根目录 `task lint` 调用 `tools/lint-runner`，报告输出至 `build/lint_report/`。
  2. `tools/lint-runner/main.go` 忽略单个 `golangci-lint` 退出码（脚本整体仍可能退出 0），且仅扫描包含 `go.mod` 的目录；当前 `.taskfiles/Taskfile.ci.yml` 的 `LintDirs` 列出了 `src/component/db` 等不含独立 `go.mod` 的子目录，会被 `lint-runner` 跳过。
  3. 因此要获得可信的门禁结论，应直接在受影响模块目录（`src/lobbysvr`、`src/component`）运行 `golangci-lint run ./...` 并检查退出码与完整报告，或核查 `build/lint_report/` 内容。
- **可选预提交钩子**：`script/hooks/` 提供 `goimports` + `golangci-lint` pre-commit hook（需本地执行 `script/hooks/install.{bat,sh}` 配置 `core.hooksPath = script/hooks`；当前本机未启用，日志写入 `script/hooks/pre-commit.log`）。
- **`.proto` 格式**：遵循 `.clang-format`（`Language: Proto`、`Google`、`ColumnLimit: 120`）。

### 4. Markdown、引用与图表质量检查

- **Markdown Lint**：若项目或环境提供 `markdownlint-cli2` 配置，按配置运行，**严禁为通过检查全局关闭规则**；当前仓库尚未配置 `.markdownlint*` 且本机未安装 `markdownlint-cli2`，变更 Markdown 时使用确定性检查核验标题层级、相对链接有效性、代码围栏闭合与语言标注、YAML frontmatter 规范、换行符（`.gitattributes`：`*.md eol=crlf`）及模板资源存在性，并在交付中如实标注 `markdownlint-cli2` 未运行。
- **图表与文档站点**：架构与流程图按 `doc/docs/writing-guide/index.md` 选用 Mermaid、Draw.io、Excalidraw 或 SVG（数据图按需用 Chart.js 等），检查语法、展示尺寸可读性及相对路径引用；不把某一绘图工具当统一硬门槛。

## 异常与恢复

- **缺少 `golangci-lint` / `goimports` / `markdownlint-cli2`**：在报告中明确标为“工具未安装，检查未执行”，不通过伪造通过状态或擅自批量安装绕过限制；必要且已获授权时再按 `README.md` / `.taskfiles/Taskfile.ci.yml` 固定版本安装。
- **区分既有失败与本次引入失败**：记录基线中已存在的失败，不删除断言、不无故 `t.Skip`、不靠盲目拉长超时掩盖死锁或竞态。
- **停止条件**：覆盖本次变更风险的必要检查全部通过后即停止，不无依据反复重跑已通过测试。

## 验证与交付

- 交付中**分别报告**以下三类状态，不得混为一谈：
  1. **发现测试 / 静态核查**（找到哪些测试、静态/链接检查结果）
  2. **编译通过**（`go build` / `go test -c` 结果）
  3. **实际执行通过**（具体目录、命令、退出码、通过/失败/跳过用例数及耗时）
- 任何因缺生成代码、缺外部服务或缺工具而未执行的检查，须单列为“未执行项”并注明缺失前提。

## 按需资源

- [.github/instructions/gotest.instructions.md](../../../.github/instructions/gotest.instructions.md)：Go 单元测试分类编码模板（独立工具函数、外部依赖 Mock、接口实现、并发与 Benchmark）。编写新测试前按需读取。
- [.golangci.yaml](../../../.golangci.yaml)：项目完整 linter 列表、阈值及排除规则。排查 lint 告警时按需读取。
