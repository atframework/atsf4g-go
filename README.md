# atsf4g-go

Service framework for game server using `libatbus-go`, `libatapp-go`, and `atframe-utils-go`.

To work with <https://github.com/atframework/atsf4g-co>.

## Quick Start

### Toolchain Setup (Recommended: `mise` + `uv`)

```powershell
# Recommended: provision pinned Go 1.27.4, JDK, uv, protoc, golangci-lint, helm via mise
mise install
```

Or install prerequisites manually on Windows:

```powershell
scoop install go uv helm
scoop bucket add java
scoop install microsoft-lts-jdk
```

### Build (Pure Go CLI: `go tool dev`)

`atsf4g-go` uses a single root Go 1.27.4 module (`github.com/atframework/atsf4g-go`) with `go.work` linking the three standalone `atframework/*` submodules, driven by the unified cross-platform Go CLI `go tool dev` (backed by `atframe-utils-go/buildkit`).

```powershell
# Inspect local toolchain and auto-discovered business services
go tool dev doctor

# Incremental build for all discovered services (SHA-256 cached proto/Excel/Mako codegen + Go build)
go tool dev build

# Build a specific service (e.g. lobbysvr) or skip codegen for fast inner-loop Go compilation
go tool dev build -service lobbysvr
go tool dev build -service lobbysvr -skip-codegen

# Force full regeneration of proto, Excel configs, and Mako templates
go tool dev build -force-codegen

# Run granular stages independently
go tool dev proto
go tool dev config
go tool dev codegen
go tool dev docs
```

### Testing, Standalone Verification & Linting

```powershell
# Run tests across atsf4g-go (supports -affected origin/main and GOCACHEPROG remote cache)
go tool dev test

# Verify all three atframework submodules compile & pass tests standalone (GOWORK=off)
go tool dev verify-standalone -test

# Run golangci-lint (strict exit codes + depguard architecture boundary rules)
go tool dev lint

# Clean generated code / caches / full build artifacts
go tool dev clean -pb
go tool dev clean -all
```

### Project Configuration

```powershell
cd build/install/
.\update_dependency.bat
.\generate_config.bat
```
