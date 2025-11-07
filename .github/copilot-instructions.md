# Copilot Instructions for atsf4g-go

## Architecture

- Entry points live under `src/lobbysvr` and `src/robot`; each constructs a service via `component-service_shared_collection` and mounts modules from `component-user_controller` and `logic/*`.
- Game logic is split into `service-lobbysvr/logic/<domain>` modules; each module exposes interfaces (e.g. `logic/character.UserCharacter`) plus `data` layer helpers in `service-lobbysvr/data` for shared state.
- The game player's data layer sits in `src/lobbysvr/data`; it defines user-centric managers and the inventory-style abstractions used to manipulate resources and items.
- Generated protobufs sit in `src/component/protocol/{public,private}`; runtime code references these via the `pbdesc` and `common` packages—always use the generated accessors/mutators instead of hand-editing nested structs.
- Excel-driven configuration is loaded through `config.GetConfigManager().GetCurrentConfigGroup()`; each group exposes typed accessors such as `GetExcelSkillBySkillId`.
- Generated configs may return nil—always guard against nil pointers when dereferencing results and prefer the provided getters over direct field access.

## Configuration & Data Flow

- Character/skill updates must resolve config via getters (never cache struct pointers): fetch skill configs with `GetExcelSkillBySkillId`, upgrade sequences with `GetExcelSkillUpgradeById`, character levels with `ExcelCharacterExpUpgrade`, and star upgrades with `ExcelCharacterStarUpgrade`.
- Conditions are validated through `logic_condition.UserConditionManager`; build runtimes with `logic_condition.CreateRuleCheckerRuntime` plus `logic_condition.CreateRuntimePair` for the active character instance and short-circuit on non-OK `cd.RpcResult` values.
- Shared managers are fetched via the generic helper `data.UserGetModuleManager`; request the concrete manager type you need, check for nil, and log failures through the provided `ctx.LogError` helpers.
- Update character state through `itemData.MutableCharacter()` and mirror changes into caches (`needRefreshAttributes`, `needRefreshItemInstanceData`) so subsequent reads trigger refresh.
- Player resource grant/check/deduct flows always go through the itemized interfaces in `service-lobbysvr/data`; when a deduction references Excel configs, call `CheckCostItem` with the expected costs before performing `Sub`, and use `MergeCostItem` to consolidate multiple config sources when needed.

## Workflows

- Build via `task build`; lint with `task lint` (requires `golangci-lint` v2.6.0 installed as noted in the root README).
- Regenerate configs/protos with the `tools/generate` suite. VS Code tasks map to `go run` wrappers such as **Generate-Protocol:Public** and **Build-lobbysvr**; prefer invoking these tasks over ad-hoc commands.
- Configuration assets under `build/install` are refreshed by `update_dependency.{bat,sh}` and `generate_config.{bat,sh}`—run both after pulling upstream schema changes.
- When adding Proto or Excel definitions, update `build/_gen/*.yaml` and rerun the associated generator tasks to keep `pbdesc` packages in sync.

## Conventions

- Use `public_protocol_pbdesc.EnErrorCode_*` when returning RPC errors; `cd.CreateRpcResultError(nil, code)` reserves the first argument for internal errors (keep `nil` unless you have an internal failure) and the second for the client-facing code.
- Resource deductions must follow the `Check` then `Sub` pattern on the player's inventory abstraction; skip direct mutations on protobuf structs to avoid desync with caches.
- Many generated structs embed pointer fields—check for nil before dereferencing and rely on methods like `GetFoo()` rather than touching members directly to stay compatible with hot-reloadable config data.
- Successful paths should return `cd.CreateRpcResultOk()`; only populate error codes when business validation fails.

## Reference Paths

- Character lifecycle examples: `service-lobbysvr/logic/character/impl/user_charater_instance.go` (skills initialization, level/star upgrades).
- Dispatcher/task abstractions: `component-dispatcher` package and `service-lobbysvr/app` wiring.
- Resource generation templates live under `src/template` and `tools/generate-for-pb`; follow these when extending config-driven features.
