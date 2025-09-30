# atsf4g-go

Service framework for game server using libatbus-go, libatapp-go and etc.

To work with <https://github.com/atframework/atsf4g-co>.

## Quick Start

### Prepare

```bash
// windows PowerShell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

scoop install python
scoop bucket add java
scoop install gradle maven microsoft-lts-jdk
scoop install helm
```

```bash
// Build
cd tools/build
go run .
```

```bash
// config
cd build/install/
update_dependency.bat
generate_config.bat
```
