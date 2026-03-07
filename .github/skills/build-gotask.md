# Build with go-task (`task`)

This repo uses [go-task](https://taskfile.dev/) (`task`) as the primary build orchestrator (see `Taskfile.yml`).

## Prerequisites

- Install `task` (go-task). On Windows, use one of:
  - `choco install go-task`
  - `scoop install task`
  - `go install github.com/go-task/task/v3/cmd/task@latest` (ensure `$GOPATH/bin` is on `PATH`)
- System requirements (used by `task` bootstrap steps):
  - Go toolchain (this repo is multi-module)
  - Git (for submodules)
  - Java `>= 17` (checked by `.taskfiles/Taskfile.build.prepare.yml`)
  - Python (Windows: `python`, others: `python3`) with `venv` support

## Python venv location

Bootstrap tasks create/use a Python virtual environment:

- Default path:
  - Windows: `./protject_venv`
  - Linux/macOS: `${HOME}/protject_venv`
- Override: set env var `PYTHON_VENV_PATH` to point to a custom venv directory.

## Submodules

This repo includes git submodules under `third_party/xresloader/*`.

- Initialize/update all submodules:
  - `git submodule update --init --recursive`
- Update submodules to upstream commits:
  - `git submodule update --remote --recursive`

## Common tasks (repo root)

- List all tasks: `task -l`
- Full build (includes cleanup + tool/bootstrap + codegen): `task build`
- Fast build (skips some expensive steps): `task fast-build`
- Build only one module:
  - `task build:lobbysvr` / `task fast-build:lobbysvr`
  - `task build:robot` / `task fast-build:robot`
- Lint: `task lint` (delegates to `task ci:lint`, which installs `golangci-lint@v2.6.0` if needed)

## Outputs and directories

- Build outputs: `build/install/**`
- Generated files: `build/_gen/**`
- Tool downloads/cache: `build/tools/**`
- Tool versions/env: `.taskfiles/build-tools.env`

## VS Code tasks

If you use VS Code, `/.vscode/tasks.json` contains wrappers such as:

- `Build-build` / `Build-fast-build` (runs `task build` / `task fast-build`)
- `Deploy-Update Dependencies` / `Deploy-GenerateServiceConfig` (runs scripts under `build/install/`)
