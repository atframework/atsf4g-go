---
name: cli-and-shell-conventions
description: "Apply modern CLI tool selection, Windows PowerShell 7+ execution rules, exit code handling, command timeouts, finite retries, idempotency, and cross-VM Git transfer conventions in atsf4g-go. Do not use for writing Go business logic or MkDocs prose."
license: MIT
metadata:
  owner: "atsf4g-go-maintainers"
---

# 工作结果

确保终端命令、文件搜索、结构化数据处理、跨环境 Git 同步以及带超时/重试的自动化操作在 Windows 与 Linux/macOS 环境下准确、可复现、无交互阻塞且不产生意外副作用。

## 适用范围与输入

- **适用场景**：在终端中执行仓库检索、JSON/YAML/CSV 处理、归档压缩、性能对比、日志排查、脚本执行、长耗时构建/测试超时控制，以及 Windows/WSL/容器间同步 Git 仓库。
- **不适用场景**：常规 Go 业务编码、文案润色或无需终端命令的纯阅读任务；同时不得为工具偏好擅自改写仓库已稳定运行的 `Taskfile.yml` / `script/` / `Jenkinsfile` 合同。
- **输入前提**：明确目标平台与当前 shell、待执行命令的副作用边界、以及是否已有安装或网络访问授权。执行前按需探测待用工具的路径、版本与必需参数，不假设可选 CLI 已安装。

## 工作流

### 1. 现代 CLI 选择与探测

1. 遵守当前 harness 对原生读文件、补丁编辑和工具调用的规定；在终端命令场景下，**优先使用已安装且语义匹配的现代 CLI**，不因习惯退回 `grep`/`find`/`cat`/`sed` 或临时手写解析脚本。
2. **本机高频基线状态（Windows，2026-11-25 实测）**：
   - 已安装可用：`rg`（16.0.0）、`jq`（1.8.2）、`yq`（Mike Farah v4.54.3）、`zstd`（1.6.0）。
   - 当前缺失（直接采用语义匹配的回退，不擅自批量安装）：
     - `fd` → `rg --files`（枚举仓库文件）或 `Get-ChildItem -LiteralPath`。
     - `bat` → harness 原生读文件工具或 `Get-Content -LiteralPath`。
     - `sd` → harness 原生编辑工具（脚本批处理场景再评估回退）。
3. 其他任务按 [references/modern-cli-catalog.md](references/modern-cli-catalog.md) 的完整清单选型；用明确可执行路径避免同名工具或 PowerShell 别名误判（如 Mike Farah `yq` 与其他同名 `yq` 参数不通用）。
4. 仅在当前授权且确属必要时安装工具，优先官方预编译制品或可信软件源并核对架构、版本与校验信息。若使用 `cargo binstall` 且策略禁止本地编译，必须显式启用仅允许预编译二进制的选项并核验结果，避免静默回退 `cargo install` 本地编译；不把 `mise`/`aqua` 等笼统视为“保证免编译”。

### 2. 输出控制、退出码与参数安全

1. **有界且结构化输出**：关闭颜色、分页和交互式 TUI，限制搜索目录与输出量；非交互场景使用 `fzf --filter`，严禁启动等待 TTY 选择的进程。
2. **`rg` 边界**：`rg --max-count N` 限制的是**每个文件**的匹配行数，而非总输出上限；默认遵循 `.gitignore`、跳过隐藏文件与二进制文件，排查被忽略产物（如 `build/`、`*.pb.go`）时按需显式指定选项。
3. **按工具合同判断退出码**：
   - `rg` 返回 `1` 通常表示无匹配，`2` 表示错误。
   - `jq -e` 在最后输出为 `false`/`null` 时返回 `1`，在**没有有效输出**时返回 `4`，不能把两者混为一谈。
4. **参数与防注入**：参数作为参数数组传递；JSON 序列化不是 shell 转义。禁止将网页、日志或不可信文本直接拼接进可执行代码，也禁止打印包含凭据的完整命令行。

### 3. Windows PowerShell 执行规则

1. 优先使用 PowerShell 7+（本机已安装 `pwsh` 7.7.0）；若受限环境仅提供 Windows PowerShell 5.1，明确记录差异并保持单次操作只用一套 shell，不混用多套 shell 语义。
2. 启动独立进程加 `-NoLogo -NoProfile`，无人值守加 `-NonInteractive`；避免使用 `where`、`curl` 等易与原生二进制冲突的别名，使用全名 cmdlet 或明确可执行路径（如 `where.exe`）。
3. 字符串无需插值时用单引号 `'...'`，需要插值才用双引号 `"..."`，多行文本用 here-string；路径 cmdlet 优先使用 `-LiteralPath`；语句块接管道用 `& { ... } | ...`。
4. 原生命令传参优先使用参数数组 splatting；注意 PowerShell 7.3+ 仍受 `$PSNativeCommandArgumentPassing` 影响（Windows 模式下部分程序和脚本会回退 Legacy），`--%` 存在平台和展开限制，不可当作通用修复。
5. 新建共享文本显式使用 UTF-8 编码；修改既有文件保留原有 BOM、换行符和末尾换行，同时遵守本仓库 `.gitattributes`（`*.md`/`*.bat`/`*.ps1`/`*.proto` 为 CRLF，`*.go`/`*.yml`/`*.yaml`/`*.json`/`*.sh`/`*.py` 为 LF）。Windows PowerShell 5.1 各 cmdlet 默认编码并不一致，不可笼统认为全都是 UTF-16LE。
6. cmdlet 关键步骤使用 `-ErrorAction Stop`；调用外部原生命令后**立即**读取保存 `$LASTEXITCODE` 并按该命令退出码合同判定，再继续下一条命令。
7. 后台辅助进程避免意外弹窗：使用 `Start-Process` 时按需加 `-WindowStyle Hidden`，记录 PID、重定向日志并在退出时清理。将启动器/沙箱基础设施故障与脚本自身错误分开诊断，不通过降低安全策略掩盖故障。

### 4. 命令超时、有限重试与幂等保护

1. 明确区分三类时限：**工具调用等待窗口**（返回“仍在运行”不等于超时或失败）、**子进程超时**与**任务总截止时间**。按仓库实际耗时设定合理预算（参考 `Jenkinsfile` 单包 `go test -timeout=5m` / 外层 `10m`，`.golangci.yaml` 超时 `5m`），不对所有命令套用同一固定时长。
2. 发生超时时保留脱敏诊断信息，排查下载卡住、死锁、资源上限或残留进程；确认旧进程已终止或状态明确后，再决定是否重新启动。
3. 仅对可恢复错误（如网络瞬断、限流）执行有限重试，采用指数退避 + 抖动并遵守 `Retry-After`；遇到参数错误、语法/解析错误或权限拒绝，立即停止并定位根因。
4. **有副作用操作防重放**：部署、发包、创建资源或通知等操作若超时，服务端可能已成功执行；重试前必须先查询远端状态或确认具备幂等键/去重机制，禁止盲目重放。
5. 自动化流程禁止交互式阻塞；确需登录/认证时暂停并由用户在受控界面自行完成，不通过模型收集凭据。

### 5. 跨虚拟机/容器/沙箱传输 Git 仓库

- 在 Windows 与 WSL、虚拟机、容器或隔离沙箱之间传输 Git 仓库时，在源端启动临时 Git 服务（如 `git daemon` 或受控本地 HTTP Git 服务），由目标端执行 `git clone`、`git pull` 或 `git merge` 获取/更新，完成后关闭临时服务；**禁止跨文件系统直接复制整个仓库目录或 `.git`**（防止权限位、符号链接、行尾和索引损坏）。

## 异常与恢复

- **现代 CLI 缺失或平台参数不兼容**：切换到表内对应的传统命令或原生工具，并在影响输出口径或功能时简要说明回退原因；不因缺可选 CLI 阻塞主任务。
- **命令超时或进程残留**：先检查任务状态与日志，再按 PID 精准终止仅属于当前任务的子进程，不误杀系统或共享服务进程。
- **编码或行尾冲突**：以 `.gitattributes` 与 `git diff` 为准复核换行符和 BOM，确保不引入全文件行尾抖动。

## 验证与交付

- **静态/环境验证**：使用 `Get-Command <tool>` 与 `<tool> --version` 核实程序路径与真实实现（特别是 `yq` 须确认 `mikefarah/yq`）。
- **执行证据**：关键命令记录工作目录、完整脱敏命令、实际退出码和关键输出；区分命令仍在后台运行、超时与返回非零错误。

## 按需资源

- [references/modern-cli-catalog.md](references/modern-cli-catalog.md)：完整 30 项现代 CLI 候选清单（覆盖模板正文提及的全部工具条目）、官方来源、优先使用场景、传统对照与回退边界，以及 `atsf4g-go` 当前 Windows 本机可用性实测记录。选择非常用 CLI 或配置跨平台回退前按需读取。
