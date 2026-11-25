---
name: build-and-codegen
description: "Build atsf4g-go Go services, bootstrap build tools and submodules, and generate protobuf, xresloader Excel config, and RPC/DB Mako template code via go-task (Taskfile.yml). Do not use for writing domain business logic, unit test design, or prose documentation."
license: MIT
metadata:
  owner: "atsf4g-go-maintainers"
---

# 工作结果

使用仓库既定的 `go-task`（`Taskfile.yml`）流水线完成子模块初始化、构建工具准备、Protobuf（含 `protoc-gen-mutable`）、xresloader 配置表与 Mako RPC/DB 模板代码生成，以及 `lobbysvr` 服务二进制构建，并保持生成产物与源码边界清晰。

## 适用范围与输入

- **适用场景**：初始化构建环境、拉取/更新子模块、修改 `.proto` / Excel 配置表 / `generate-for-pb.atfw.yaml.tpl` / `src/template/*.mako` 后重新生成代码、编译 `lobbysvr` 或构建 `doc/` 站点。
- **不适用场景**：在未改动 proto/模板时手改生成文件、编写 Go 业务逻辑（改用 `lobbysvr-development`）、编写或运行 Go 单元测试/lint（改用 `go-testing-and-lint`）。
- **必要前提（以 `Taskfile.yml` 与 `.taskfiles/build-tools.env` 为准）**：
  - `task`（go-task）、Go（模块声明 `go 1.25.1`）、Git、Java `>= 17`（`.taskfiles/build-tools.env`：`ENV_JAVA_VERSON = "17"`）、Python 3（带 `venv` 模块；Windows 用 `python`，Linux/macOS 用 `python3`）。
  - Helm（仅更新 `install/cloud-native/charts/*` 依赖和生成部署配置时需要）。
  - Python 虚拟环境默认路径：Windows 为 `./protject_venv`，Linux/macOS 为 `${HOME}/protject_venv`，可通过环境变量 `PYTHON_VENV_PATH` 覆盖。
- **仓库事实边界**：`Taskfile.yml`、`README.md` 与 `.vscode/tasks.json` 中存在历史 `robot` 任务或注释引用，但当前工作树**不存在 `src/robot/` 目录**，且 `Taskfile.yml` 仅定义了 `inner_build-module:lobbysvr`。不要假设 `task build:robot` 可执行。

## 工作流

1. **确认子模块与工作区状态**
   - 检查工作区是否有未提交修改；确认全部 5 个 Git 子模块状态：
     - `atframework/atframe-utils-go`、`atframework/libatapp-go`、`atframework/libatbus-go`
     - `third_party/xresloader/xres-code-generator`、`third_party/xresloader/xresconv-cli`
   - 初始化子模块（在仓库根目录）：`git submodule update --init --recursive`。未获明确授权不要擅自执行 `--remote` 升级子模块或修改子模块源码。

2. **选择最小必要 `task` 入口（仓库根目录执行）**
   - 查看所有可用任务：`task -l` 或 `task help`。
   - 首次全量构建（清理 `build/` 与 venv、准备 `protoc`/`atdtool`/`xresloader`/Java/Python 并生成全部代码、编译 `lobbysvr`）：`task build` 或 `task build:lobbysvr`。
   - 增量构建（要求 `build/build-settings.json` 与工具已存在，仍会执行代码生成、`go mod tidy` 与编译）：`task fast-build`、`task fast-build:lobbysvr` 或 `task "fast-[lobby]-build"`。
   - 仅生成协议与配置：`task "fast-[proto]-gen"`（执行 `component:generate-protocol`，内部联动 `generate-config`）或 `task "fast-[config]-gen"`。
   - 仅构建文档站点：`task "fast-[doc]-build"`（创建 `doc/pyvenv`、生成协议文档并运行 `mkdocs build`）。
   - 清理生成 Go/pb 文件：`task tools:clean-generate-pb`。

3. **遵循代码生成与 `protoc-gen-mutable` 规范**
   - **权威源文件（可编辑）**：
     - `.proto` 定义：`src/component/protocol/{public,private}/...`、`src/lobbysvr/protocol/{public,private}/...`
     - Excel 配置：`resource/ExcelTables/*.xlsx`
     - Mako 模板：`src/template/*.mako`（含 `handle_cs_rpc.go.mako`、`task_action_cs_rpc.go.mako`、`session_downstream_api_for_cs.go.mako`、`config_group.go.mako`、`config_set.go.mako`、`db_rpc_*.mako` 等）
     - 模块生成规则模板：`src/lobbysvr/generate-for-pb.atfw.yaml.tpl`、`src/component/db/generate-for-pb.atfw.yaml.tpl`、`tools/generate-for-pbtool/generate-for-pb.conf.yaml`
   - **禁止手改的生成产物（均在 `.gitignore` 中）**：
     - `*.pb.go`、`*_mutable.pb.go`、`src/component/config/generate_config/`
     - `src/component/db/global_db.go`、`src/component/db/local_db.go`
     - `src/lobbysvr/app/*`（**唯一例外**：`src/lobbysvr/app/register_modules.go` 为手工维护文件）
     - `src/lobbysvr/rpc/lobbyclientservice/*`
     - `build/_gen/generate-for-pb.yaml`、`build/install/**`、`build/tools/**`
   - **自定义插件 `protoc-gen-mutable` 能力**（生成 `*_mutable.pb.go`）：
     - `Mutable<Field>()`：消息字段空值自动创建并返回非 nil 指针；优于手写 nil 检查与初始化。
     - `Clone()` / `Merge(other)`：安全深拷贝与合并，避免跨 goroutine 共享可变 proto 指针。
     - Oneof 枚举常量与可变访问：`Get<Oneof>OneofCase()` 配合生成的 `En<Oneof>ID_*` 常量（如 `DActivate_EnEventID_QuestId`）与 `Mutable<Case>()`，不硬编码字段号。
     - `ToReadonly()` 不可变视图、`GetReflectType()` 反射辅助及 `slog.LogValuer` 结构化日志输出。

4. **新增或修改 RPC 的标准顺序**
   1. 修改对应 `.proto` 文件（遵守 `.clang-format`：`Language: Proto`、`BasedOnStyle: Google`、`ColumnLimit: 120`）。
   2. 按需更新 `src/lobbysvr/generate-for-pb.atfw.yaml.tpl` 或 `src/component/db/generate-for-pb.atfw.yaml.tpl`。
   3. 运行 `task fast-build:lobbysvr`（或首次运行 `task build:lobbysvr`）重新生成代码并编译验证。
   4. 生成 `build/install/` 后，若需本地生成服务配置，在 `build/install/` 下按平台执行 `update_dependency.{bat,sh}` 与 `generate_config.{bat,sh}`。

## 异常与恢复

- **`fast-build` 报 `build-settings.json` 或 `build_setting` 缺失**：先执行完整 `task build`（或 `task build-prepare:build-tools-prepare`）初始化 `build/tools` 与 `build/build-settings.json`。
- **工具下载超时或网络波动**：检查 `build/tools/` 半成品后有限重试；工具版本由 `.taskfiles/build-tools.env` 控制（当前 `protoc 32.1`、`protoc-gen-go v1.36.9`、`atdtool 1.0.6`、`xresloader 2.23.4`），不要随意改写版本号掩盖网络问题。
- **残留旧 `.pb.go` 引起编译冲突**：在仓库根目录运行 `task tools:clean-generate-pb` 后再执行 `task fast-build:lobbysvr`。
- **切勿破坏正常构建目录**：清理仅针对本任务临时路径或使用既有 `task` 清理任务，确认不包含未提交源码。

## 验证与交付

- **静态检查**：确认未手工改动任何 `.gitignore` 中的生成路径，`.proto` 与模板修改符合格式。
- **构建验收**：记录工作目录、执行的 `task` 命令、环境变量（如 `BUILD_MODE`）、退出码及产出路径（如 `build/install/lobbysvr/bin/lobbysvrd(.exe)`）；未实际执行构建时明确标明“仅静态核对，构建未运行”。
- **产物交付**：列出修改的源文件/模板、已触发的代码生成范围、验证结果以及任何环境阻塞项（例如缺工具或网络离线）。
