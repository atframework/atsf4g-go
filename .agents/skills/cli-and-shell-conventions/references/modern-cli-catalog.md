# 现代化命令行工具完整候选清单

本清单完整保留项目初始化模板提供的全部现代 CLI 候选条目、优先使用场景、传统对照与回退边界，并附 `2026-11-25` 在当前 Windows 工作站的实际可用性探测结果。

> 核实说明：初始化模板正文提及“31 项候选”，但模板内实际列出 30 行工具条目。本表完整保留全部 30 项，未删减也未私自编造第 31 项。各链接仅对应上游工具用途与文档，不代表全平台实测或授权批量安装；实际使用前请核对当前平台、实现版本与参数兼容性。

## 高频基线摘要（当前 Windows 主机，2026-11-25 实测）

- **已安装可用（4 项）**：
  - `rg`：`ripgrep 16.0.0`（`C:\Users\owt50\scoop\shims\rg.exe`）
  - `jq`：`jq-1.8.2`（`C:\Users\owt50\scoop\shims\jq.exe`）
  - `yq`：Mike Farah `v4.54.3`（`C:\Users\owt50\scoop\shims\yq.exe`）
  - `zstd`：`v1.6.0`（`C:\Users\owt50\scoop\shims\zstd.exe`）
- **基线缺失回退**：`fd` → `rg --files` / `Get-ChildItem -LiteralPath`；`bat` → harness 原生读取 / `Get-Content -LiteralPath`；`sd` → harness 原生编辑工具。

## 完整候选清单（全部 30 项）

| # | 现代工具与来源 | 本机状态（2026-11-25） | 用途与优先使用场景 | 传统对照、回退与边界 |
| --- | --- | --- | --- | --- |
| 1 | [ripgrep（`rg`）](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md) | 已安装（`16.0.0`） | 首选文本搜索；`rg --files` 枚举仓库文件 | 对照 `grep`；缺失时用 `Select-String` 等；注意忽略规则、隐藏项、二进制及 `--max-count` 按单文件计数 |
| 2 | [ugrep](https://github.com/Genivia/ugrep) | 未安装 | 需要其搜索能力或 grep 兼容选项时使用 | 对照 `grep`；逐项验证实际选项，不宣称所有 grep 行为完全等价；自动化避免 TUI |
| 3 | [fd](https://github.com/sharkdp/fd) | 未安装 | 按文件名、路径、类型筛选文件 | 对照 `find`；回退 `rg --files` / `Get-ChildItem`；注意隐藏项、忽略规则及正则/字面量差异 |
| 4 | [bat](https://github.com/sharkdp/bat) | 未安装 | 查看文件、行号和指定范围 | 对照 `cat`；回退原生读取或 `Get-Content`；自动化关闭分页和颜色 |
| 5 | [sd](https://github.com/chmln/sd) | 未安装 | 适合其字符串/正则语义的文本替换 | 对照 `sed`；先核验替换范围、捕获组和文件编码；补丁类修改仍按 harness 规定执行 |
| 6 | [eza](https://github.com/eza-community/eza) | 未安装 | 目录列表和树形浏览 | 对照 `ls`/`tree`；按平台核验支持，限制递归深度，不解析图标和彩色展示作为数据 |
| 7 | [erdtree（`erd`）](https://github.com/solidiquis/erdtree) | 未安装 | 结合目录树与磁盘占用查看目录 | 对照 `tree`/`du`；限制扫描范围，区分目录展示与精确存储统计 |
| 8 | [dust](https://github.com/bootandy/dust) | 未安装 | 定位大文件、目录及磁盘占用分布 | 对照 `du`；注意逻辑大小、分配空间和权限造成的统计差异 |
| 9 | [duf](https://github.com/muesli/duf) | 未安装 | 查看文件系统容量和挂载信息 | 对照 `df`；容器、网络盘和挂载可见范围以实际环境为准 |
| 10 | [hyperfine](https://github.com/sharkdp/hyperfine) | 未安装 | 重复测量命令耗时、比较候选方案 | 对照 `time`/`Measure-Command`；控制预热与缓存；Windows 默认 shell 为 `cmd.exe`，依赖 PowerShell 语义时显式加 `--shell pwsh` |
| 11 | [tokei](https://github.com/XAMPPRocky/tokei) | 未安装 | 按语言统计代码、注释及空行 | 对照 `cloc`；记录排除目录及统计口径，不用行数替代质量评价 |
| 12 | [hexyl](https://github.com/sharkdp/hexyl) | 未安装 | 有界查看二进制的十六进制内容 | 对照 `xxd`/`hexdump`；二进制转换和编辑另核验所需工具，避免输出敏感数据 |
| 13 | [jq](https://jqlang.org/manual/) | 已安装（`1.8.2`） | JSON 查询、转换及条件判断的默认选择 | 回退项目解析库或 PowerShell JSON cmdlet；保留类型和退出码语义（`jq -e`：`false`/`null` 为 `1`，无有效输出为 `4`），不用正则代替 JSON 解析 |
| 14 | [jaq](https://github.com/01mf02/jaq) | 未安装 | 已确认过滤器兼容时的 `jq` 替代候选 | 不保证与 `jq` 完全等价；先验证项目实际过滤器、模块和错误行为 |
| 15 | [yq（Mike Farah）](https://github.com/mikefarah/yq) | 已安装（`v4.54.3`） | YAML 等结构化配置的查询与修改 | 必须核对实现与版本；与其他同名 `yq` 参数不通用；写回后检查注释、样式及数据语义 |
| 16 | [Miller（`mlr`）](https://github.com/johnkerl/miller) | 未安装 | CSV、TSV、JSON 等记录数据的筛选与转换 | 对照 `awk`/`cut` 等组合；遵循真实格式、引号和字段类型，不直接按逗号切 CSV |
| 17 | [qsv](https://github.com/dathere/qsv) | 未安装 | CSV 数据处理、校验和统计 | 对照 CSV 脚本及文本管道；按发行变体、实际子命令和资源需求核验能力，不默认全部操作恒定内存 |
| 18 | [git-delta（`delta`）](https://github.com/dandavison/delta) | 未安装 | 提升人工审阅 diff 的可读性 | 对照普通 `diff` 展示；自动化关闭分页，机器处理保留 `git diff` 原始输出 |
| 19 | [difftastic（`difft`）](https://github.com/Wilfred/difftastic) | 未安装 | 对支持的语言比较语法结构差异 | 对照逐行 diff；结构视图辅助审阅，变更范围及补丁仍核对 `git diff` |
| 20 | [tailspin（`tspin`）](https://github.com/bensadeh/tailspin) | 未安装 | 日志高亮与人工定位 | 对照 `tail`/`less` 的日志展示；先核验分页、跟随和颜色设置，机器解析优先原始日志 |
| 21 | [plocate](https://sources.debian.org/src/plocate/1.1.18-1/README) | 未安装（Linux 专用） | Linux 环境中利用已有索引查找文件名 | 对照 `locate`；依赖索引覆盖及新鲜度，命中后检查文件；无适用索引时用 `fd`/`rg --files`，不为局部查找默认扫描整盘建库。注：上游官网此前核验超时，基础用途依据 Debian 保存的 `1.1.18` README，未据此确认其最新版本 |
| 22 | [pigz](https://zlib.net/pigz/) | 未安装 | 需要 gzip 格式时进行并行压缩 | 对照 `gzip`；不把压缩的并行收益套用到所有解压阶段，核验线程与内存预算 |
| 23 | [zstd](https://github.com/facebook/zstd) | 已安装（`v1.6.0`） | 消费端支持 Zstandard 时的压缩候选 | 对照 `gzip`/`xz` 等方案；依据格式合同、压缩率、耗时和内存测量选择，不直接改制品格式 |
| 24 | [ouch](https://github.com/ouch-org/ouch) | 未安装 | 统一入口处理其支持的压缩及归档格式 | 对照 `tar`/`unzip` 等；核验具体格式和选项，解压前确认目标目录及路径安全 |
| 25 | [aria2（`aria2c`）](https://aria2.github.io/) | 未安装 | 文件下载、续传及适用的并发下载 | 对照 `wget`/`curl` 的下载用途；遵守服务限流，校验制品，避免凭据进入参数或日志 |
| 26 | [fzf](https://github.com/junegunn/fzf) | 未安装 | 候选集合的模糊筛选 | 自动化使用 `--filter`；精确匹配优先 `rg`，不启动等待 TTY 选择的流程 |
| 27 | [xh](https://github.com/ducaale/xh) | 未安装 | 适合其语义的 HTTP 请求与调试 | 对照 `curl`/HTTPie；不假定参数相同，核验请求体、认证、重定向及失败退出码 |
| 28 | [doggo](https://github.com/mr-karan/doggo) | 未安装 | DNS 查询与排障 | 对照 `dig`/`nslookup`；按任务明确解析器、记录类型及传输方式 |
| 29 | [procs](https://github.com/dalance/procs) | 未安装 | 筛选和查看进程信息 | 对照 `ps`；回退 `Get-Process` 等；平台支持和可见字段以当前版本为准 |
| 30 | [watchexec](https://github.com/watchexec/watchexec) | 未安装 | 开发期间监听文件变化并运行命令 | 对照轮询脚本；适合有意持续运行的任务，设置忽略范围并管理子进程和退出清理 |

## 性能与安装提醒

- 实现语言或外部基准不能证明在本项目数据上必然更快；对性能敏感的任务须在真实工作负载上实测（例如使用 `hyperfine`）。
- 仅在已获授权且任务确实需要时按平台安装缺失工具，优先官方预编译二进制与可信软件源并核对校验和；不要把本表当作批量安装清单。
