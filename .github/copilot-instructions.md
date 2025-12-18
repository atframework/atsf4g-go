# Copilot Instructions for atsf4g-go

A Go-based game server framework for lobby and robot services using libatapp-go and libatbus-go, with code generation for RPC handlers, database schemas, and Excel configs.

## Architecture

**Service Entry Points**: `src/lobbysvr` and `src/robot` create applications via `component-service_shared_collection`, wire CS/Redis dispatchers, and register logic modules from `logic/<domain>`.

**Domain Modules**: `src/lobbysvr/logic/{character,inventory,user,building,customer,menu}/` expose public interfaces (e.g., `UserCharacter`, `UserCharacterManager`) with internal implementations under `impl/`. Domain-specific RPC actions live in `action/` subdirs.

**Data Layer**: `src/lobbysvr/data/` provides base classes (`UserModuleManagerImpl`, `UserItemManagerImpl`) and manager discovery via `data.UserGetModuleManager[T](user)`. Player data is persisted through `DumpToDB()` / `InitFromDB()`.

**Proto & Config**: Generated protobufs in `src/component/protocol/{public,private}/pbdesc/` with mutable helper methods. Excel configs load via `config.GetConfigManager().GetCurrentConfigGroup()` getters (never cache config pointers—reread on each access).

## Data Flow & Patterns

**Manager Access**: Always fetch module managers via `data.UserGetModuleManager[T](user)` with nil-check before use. Error handling example:
```go
characterMgr := data.UserGetModuleManager[logic_character.UserCharacterManager](user)
if characterMgr == nil {
  return cd.CreateRpcResultError(fmt.Errorf("..."), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
}
```

**Character State Updates**: Modify state via `itemData.MutableCharacter()` (e.g., `.Level`, `.Star`), then set dirty flags (`needRefreshItemInstanceData = true`) so subsequent reads refresh caches. State changes via `character.SetState()` mark `needRefreshItemInstanceData = true` automatically.

**Config Resolution**: Config getters return nil—always guard with nil checks. Use typed accessors: `config.GetConfigManager().GetCurrentConfigGroup().GetExcelCharacterByCharacterId(typeId)`, `GetExcelSkillBySkillId()`, `GetExcelSkillUpgradeById()`. Never cache config struct pointers across RPC calls.

**Item/Resource Deductions**: Use `CheckCostItem` → `Sub` pattern on inventory managers (in `service-lobbysvr/data`); skip direct protobuf mutations to maintain cache sync. Consolidate multiple cost sources with `MergeCostItem`.

**Condition Checking**: Validate rules via `logic_condition.UserConditionManager` with `logic_condition.AddRuleChecker` registrations (see `character/impl` for examples). Build runtimes with `logic_condition.CreateRuleCheckerRuntime` and pass character instance as runtime parameter.

## Build & Generation Workflows

**Build Tasks** (run via VS Code task panel or `task` CLI):
- `task build` — builds all binaries via `tools/build`
- `task lint` — runs `golangci-lint` v2.6.0 (must be pre-installed)
- `Generate-Protocol:{Public,Private}` — regenerate proto descriptors
- `Generate-Excel and configure` — sync Excel configs from `resource/ExcelTables`
- `Build-{lobbysvr,robot}` — compile individual services

**Code Generation**: RPC handlers auto-generate from `src/template/{handle_cs_rpc,task_action_cs_rpc}.mako` rules in `build/_gen/generate-for-pb.yaml`. Add new RPC via proto definition + template rule, then run **Generate-RPC and task codes** task.

**Config Refresh**: After pulling upstream schema changes, run `build/install/update_dependency.{bat,sh}` then `generate_config.{bat,sh}` to refresh `build/install/resource/pbdesc/`.

**Database Schema**: DB interface templates generate from `src/template/db_interface.go.mako` driven by proto `svr.local.table.proto`; DumpToDB/InitFromDB patterns in managers handle persistence.

## RPC Handler Pattern

Auto-generated RPC task actions inherit from a generated base and must:
1. Extract user via context: `user := <session logic>` with nil-check
2. Fetch domain manager via `data.UserGetModuleManager[T](user)` with nil-check
3. Validate request params; return error codes on invalid input
4. Call business logic method (e.g., `character.UpgradeLevel()`); capture `cd.RpcResult`
5. On error: `t.SetResponseCode(result.GetResponseCode()); return result.GetStandardError()`
6. On success: populate response body if needed, return `nil`

Example from `task_action_character_upgrade_level.go`:
- Get manager → validate params → call `character.UpgradeLevel()` → check `result.IsError()` → set response code or return nil.

RPC methods return `cd.RpcResult` (with `IsError()`, `GetResponseCode()`, `GetStandardError()` helpers). Success paths return `cd.CreateRpcResultOk()`; errors use `cd.CreateRpcResultError(nil, errorCode)` for client-facing codes.

## Key Implementation Details

**Dirty Tracking**: Domain managers use dirty flag maps (e.g., `dirtyCharacterIds map[int32]struct{}`) to batch updates. Mark dirty via manager's `markItemDirty()` hook; dirty flush callbacks serialize to proto messages via `DumpDirtyItemData()` and relay via session API.

**Character Cache Mechanics**: 
- `itemInstanceCache` holds the serialized proto snapshot for client delivery
- `needRefreshItemInstanceData` flag forces a refresh on next read, triggering `refreshItemInstanceDataCache()` → `itemInstanceCache.MutableItemData().MarshalFrom(&itemData)`
- Skill instances stored separately: `skillInstance map[int32]*UserSkill` + `skillInterface map[int32]UserSkill` (interface wrapper for injection)

**Session API**: RPC responses sent via auto-generated `session_downstream_api.go` with `SendDownstreamRpcResponse(rpcType, body)` — determines route based on RPC meta type, wraps in proto envelope, and sends to session.

**Protobuf Mutability**: Generated structs use immutable getters (`Get*()`) and mutable accessors (`Mutable*()`). Prefer getters for reads; use mutators when modifying (avoids direct field assignment, staying compatible with code generation and mutable plugins).

## Go Testing Requirements

When writing tests, follow the standards in `.github/instructions/gotest.instructions.md`:
- Use Arrange-Act-Assert pattern
- Test file naming: xxx_test.go
- Test function naming: Test[Function][Scenario]
- Use testify/assert for assertions
- Mock external dependencies
- Cover all branches and boundary cases
