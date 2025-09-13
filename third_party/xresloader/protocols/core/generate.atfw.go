package libatapp

//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=. protocol/config/pb_header_v3.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative --proto_path=. protocol/extension/v3/xresloader.proto protocol/extension/v3/xresloader_ue.proto
