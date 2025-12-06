# atsf4g-go

Service framework for game server using libatbus-go, libatapp-go and etc.

To work with <https://github.com/atframework/atsf4g-co>.

## Quick Start

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

scoop install python task
scoop bucket add java
scoop install microsoft-lts-jdk
scoop install helm
```

// 之后退出全部终端 刷新环境变量

#### Alternatively, you can use winget to install
winget install Task.Task Microsoft.OpenJDK.25 Helm.Helm

### Build
```powershell
# Build all modules
task build

# Or build specific modules
task build:lobbysvr    # Build lobbysvr
task build:robot       # Build robot
```

#### Fast Build (without tool downloads)

```powershell
# Fast build all modules
task fast-build

# Or fast build specific modules
task fast-build:lobbysvr
task fast-build:robot
```

**Build process includes:**
- Preparing compilation tools (protoc, Java, Python, etc.)
- Downloading/installing dependency tools (atdtool, xresloader)
- Generating Protocol Buffer code
- Running `go mod tidy` to organize dependencies
- Compiling each service module

### Project Configuration

```powershell

// config
cd build/install/

.\update_dependency.bat
.\generate_config.bat
```
### Tools and Commands

#### List all available tasks

```powershell
task help
# 或
task -l
```

#### Code Analysis

```powershell
# Install linter (one time only)
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0

# Run linting
task lint
```

