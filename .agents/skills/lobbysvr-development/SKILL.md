---
name: lobbysvr-development
description: "Implement or fix features in atsf4g-go src/lobbysvr and src/component, including domain managers, RPC task actions, DB persistence, dirty tracking, downstream push APIs, and config/expression loading. Do not use for running go-task build pipelines, standalone Go test architecture, or prose/comment polishing."
license: MIT
metadata:
  owner: "atsf4g-go-maintainers"
---

# 工作结果

在 `src/lobbysvr` 与 `src/component` 中交付符合本仓库模块分层、玩家数据持久化、脏数据同步、RPC 任务动作与配置加载契约的 Go 代码修改，避免空指针、配置重载悬垂引用及生成代码冲突。

## 适用范围与输入

- **适用场景**：新增或修改 `src/lobbysvr/logic/<domain>/` 业务逻辑、玩家模块管理器、RPC `TaskActionCSBase` 处理器、脏数据同步回调、配置读取及 `enable_expression` 表达式配置使用，以及 `src/component/` 公共组件逻辑。
- **不适用场景**：构建/代码生成（改用 `build-and-codegen`）、单测规范与 lint 门禁（改用 `go-testing-and-lint`）、文档与注释润色（改用 `docs-and-comments-writing`）、修改独立子模块 `atframework/*` 源码（须单独按子模块规则处理）。
- **输入前提**：明确受影响的模块目录（`src/lobbysvr/logic/` 当前包含 `activate`、`async_jobs`、`condition`、`global_mail`、`inventory`、`mail`、`mall`、`module_unlock`、`open_platform`、`quest`、`random_pool`、`timer`、`trigger`、`unlock`、`user`；`src/component/` 包含 `async_jobs`、`config`、`db`、`dispatcher`、`dynamiclib_caller`、`logical_time`、`mail`、`open_platform`、`operation_support_system`、`protocol`、`router`、`service_shared_collection`、`user_controller`、`uuid`）。修改前先读对应领域接口、实现与调用方源码。

## 工作流

1. **遵循模块组织与注册边界**
   - `src/lobbysvr/logic/<domain>/` 放置对外接口定义，`impl/` 放置内部实现，`action/` 放置该领域的 RPC 任务动作，`data/` 按需放置内部数据结构。
   - 新增玩家模块管理器实现时，检查并在 `src/lobbysvr/app/register_modules.go` 注册 blank import（该文件是 `src/lobbysvr/app/` 下唯一手工维护的 Go 文件）。
   - 获取玩家模块管理器必须通过泛型入口 `data.UserGetModuleManager[T](user)`，且**每次调用后必须做 nil 检查**。
   - 玩家持久化统一实现 `InitFromDB(cd.RpcContext, *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult` 与 `DumpToDB(cd.RpcContext, *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult`。

2. **遵循配置访问与表达式展开约束**
   - 读取配置表与区段配置统一通过 `config.GetConfigManager().GetCurrentConfigGroup()` 的类型安全访问器；所有配置 getter 均可能返回 `nil`，使用前必须判空。
   - **禁止跨 RPC 或跨重载缓存配置结构体指针**（运行时支持配置热重载）。
   - 配置加载底层位于 `atframework/libatapp-go/config.go`（详述见 [atframework/libatapp-go/.github/skills/config-expression.md](../../../atframework/libatapp-go/.github/skills/config-expression.md)），加载优先级为：**环境变量（先）→ YAML（后覆盖）→ Proto 默认值（最后填充）**。
   - 仅标注 `enable_expression: true` 的 proto 字段会展开表达式（map key/value 由父 map 字段控制，最大嵌套深度 32 层），支持 `$VAR`（POSIX 字符集）、`${VAR}`（支持 `.`/`-`/`/` 等 k8s label 字符）、`${VAR:-default}`、`${VAR:+word}`、`\$` 转义及嵌套展开。注意 YAML 路径与环境变量路径为独立代码路径，且 `ParsePlainMessage` 不在 `LoadConfig` 主链路中。

3. **实现 RPC Task Action 标准流程**
   - 客户端请求处理器通常嵌入 `user_controller.TaskActionCSBase[Req, Rsp]`（参考 `src/lobbysvr/logic/activate/action/task_action_user_activate.go`）：
     1. 从上下文取出用户/会话（如 `user, ok := t.GetUser().(*data.User)`）并判空，失败设置错误码并返回 error。
     2. 读取请求体 `t.GetRequestBody()`，校验参数与配置，非法输入设置对应 `public_protocol_pbdesc.EnErrorCode_*`。
     3. 通过 `data.UserGetModuleManager[T](user)` 获取所需领域管理器并判空。
     4. 调用业务方法获取 `cd.RpcResult`（业务层成功返回 `cd.CreateRpcResultOk()`，错误返回 `cd.CreateRpcResultError(err, code)`）。
     5. 若 `!result.IsOK()`：执行 `t.SetResponseCode(result.GetResponseCode())` 并返回 `result.GetStandardError()`（或上下文错误）。
     6. 成功时按需通过 `t.MutableResponseBody()` 填充响应并返回 `nil`。

4. **遵循脏数据同步、Protobuf 可变性与下行推送约定**
   - **脏数据批量同步**：参照 `src/lobbysvr/logic/inventory/impl/user_inventory_manager_instance.go`，状态变更时记录脏键集合，并通过 `m.GetOwner().InsertDirtyHandleIfNotExists(m, dumpFn, clearFn)` 注册一次性的导出与清理回调；导出时写入 `dirty.MutableNormalDirtyChangeMessage()` 并对 proto 消息使用 `Clone()`，防止共享可变对象。
   - **Protobuf 访问**：读取用 `Get*()`，修改优先用 `protoc-gen-mutable` 生成的 `Mutable*()` 与 oneof 枚举常量，避免手写裸字段赋值或硬编码字段编号。
   - **服务端主动下行推送**：对请求类型为 `google.protobuf.Empty` 的 stream RPC，由 `src/template/session_downstream_api_for_cs.go.mako` 生成 `src/lobbysvr/rpc/<service>/session_downstream_api.go` 中的 `Send<RpcName>(session, body, responseCode)`（底层调用 `uc.CreateCSMessage` + `session.SendMessage`），不要手改生成文件。

5. **控制并发、时间与错误处理**
   - 游戏逻辑时间通过 `src/component/logical_time` 访问，不要在依赖逻辑偏移或日/周刷新的业务中直接硬编码 `time.Now()`。
   - 错误处理兼容 `.golangci.yaml` 要求（检查 `errcheck`、`errorlint`、`wrapcheck`、`nilerr`、`nilnil`、`exhaustive`、`durationcheck`、圈复杂度 `<= 15`、函数 `<= 80` 行 / `50` 语句）。

## 异常与恢复

- **缺少 `*.pb.go` 或 `generate_config` 导致包无法解析**：先按 `build-and-codegen` 运行 `task build:lobbysvr` 或 `task fast-build:lobbysvr` 生成依赖代码，不手工伪造生成文件。
- **业务变更涉及跨模块接口或数据库结构**：先形成可审阅的设计说明与协议改动方案，确认兼容与持久化迁移策略后再修改代码。
- **发现 `atframework/*` 子模块问题**：明确区分问题归属；未经单独授权不在解决主仓库任务时顺手修改子模块。

## 验证与交付

- **静态与编译检查**：修改后在受影响模块目录运行 `go build ./...` / `go test ./...`（涉及生成代码须先完成 codegen）及 `go vet ./...`，有 `golangci-lint` 时同步检查 lint。
- **交付说明**：明确列出修改的领域文件、受影响的 proto/配置/持久化契约、实际执行的构建/测试命令与退出码，以及未能在当前环境运行的检查项。
