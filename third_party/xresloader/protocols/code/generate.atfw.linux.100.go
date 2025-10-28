package libatapp

//go:generate bash -c "protoc --go_out=. --mutable_out=. --go_opt=paths=source_relative --mutable_opt=paths=source_relative --proto_path=. protocol/extension/xrescode_extensions_v3.proto"
