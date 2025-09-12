# atsf4g-go

Service framework for game server using libatbus-go, libatapp-go and etc.

To work with <https://github.com/atframework/atsf4g-co>.

## Quick Start

### Prepare

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install "github.com/bufbuild/buf/cmd/buf@latest"

cd tools/generate
go run .
cd -

# go run <project>/tools/generate/main.go <sub dir>

# go generate ./...
```
