package libatapp_go_protocol_generate

//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=./ ./atframe/*.proto
