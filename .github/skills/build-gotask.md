# Build with Unified Go CLI (`go tool dev` & `atbuild`)

This repo has migrated from `go-task` to a pure Go 1.27.4 build and codegen engine (`tools/dev`, backed by `atframework/atframe-utils-go/buildkit`), with `mise.toml` for toolchain version pinning and `pyproject.toml` (`uv`) for Python tooling.

## Prerequisites

- Go `>= 1.27.4`
- Git (for submodules)
- Java `>= 17` (for `xresloader.jar`)
- `uv` (recommended; automatically manages `.venv` from `pyproject.toml`) or Python 3 with `venv`
- Optional: `mise install` to provision pinned Go, JDK, `uv`, `protoc`, `golangci-lint`, and `helm`

## Common commands (repo root)

- Inspect toolchain & discovered services: `go tool dev doctor`
- Incremental build (SHA-256 cached codegen + parallel Go build): `go tool dev build`
- Build single service / fast Go-only recompile:
  - `go tool dev build -service lobbysvr`
  - `go tool dev build -service lobbysvr -skip-codegen`
- Force full regeneration: `go tool dev build -force-codegen`
- Granular stages: `go tool dev proto`, `go tool dev config`, `go tool dev codegen`, `go tool dev docs`
- Tests (supports affected pruning & `GOCACHEPROG`): `go tool dev test` / `go tool dev test -affected origin/main`
- Verify standalone builds of the 3 `atframework/*` submodules (`GOWORK=off`): `go tool dev verify-standalone -test`
- Lint (strict exit code + `depguard` rules): `go tool dev lint`
- Clean generated code / full build: `go tool dev clean -pb` / `go tool dev clean -all`
