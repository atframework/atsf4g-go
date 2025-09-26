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
```

```bash
cd tools/build
go run .
```
