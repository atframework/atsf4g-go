# Implementation Details (Conventions)

This file captures a few repo-specific implementation conventions that frequently affect correctness.

## Dirty tracking

- Domain managers commonly batch updates via dirty-flag maps (e.g., `map[int32]struct{}`).
- Mark dirty through the manager’s hook (e.g., `markItemDirty()`); flush callbacks serialize via `DumpDirtyItemData()` and notify via session APIs.

## Item/character cache mechanics (example pattern)

- Some domains maintain a serialized proto snapshot cache for downstream delivery (e.g., `itemInstanceCache`).
- When mutating underlying state, set a “refresh needed” flag (e.g., `needRefreshItemInstanceData = true`) so the next read rebuilds the snapshot (e.g., `refreshItemInstanceDataCache()`).

## Protobuf mutability

- Generated protobuf helpers typically expose:
  - Immutable getters: `Get*()`
  - Mutable accessors/builders: `Mutable*()`
- Prefer getters for reads; prefer `Mutable*()` for mutations (avoid direct field assignment to stay compatible with codegen/plugins).

## Session RPC response path

- RPC responses are commonly sent via an auto-generated downstream API (e.g., `session_downstream_api.go`) with a helper like `SendDownstreamRpcResponse(rpcType, body)`.

