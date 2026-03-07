# Architecture & Key Patterns

## Service entry points

- `src/lobbysvr` and `src/robot` are the main service modules.

## Domain modules

- Domain logic is organized under `src/lobbysvr/logic/<domain>/`.
- Public interfaces live in the domain folder; internal implementations commonly live in `impl/`.
- Domain-specific RPC task actions are typically in `action/` subdirectories.

## Data layer conventions

- Manager lookup: always fetch managers via `data.UserGetModuleManager[T](user)` and nil-check before use.
- Persistence pattern: player data persists via `DumpToDB()` / `InitFromDB()` implemented by managers.

## Config usage rules

- Config getters can return nil; always nil-check.
- Prefer typed accessors on `config.GetConfigManager().GetCurrentConfigGroup()...`.
- Do not cache config struct pointers across RPC calls (treat configs as refreshable).

