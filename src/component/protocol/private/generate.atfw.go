package libatapp

//go:generate protoc --go_out=extension --go_opt=paths=source_relative --proto_path=extension --proto_path=../public/extension extension/protocol/extension/*.proto
//go:generate protoc --go_out=common --go_opt=paths=source_relative --proto_path=common --proto_path=extension --proto_path=../public/common --proto_path=../public/extension common/protocol/common/*.proto
//go:generate protoc --go_out=config --go_opt=paths=source_relative --proto_path=config --proto_path=common --proto_path=extension --proto_path=../public/config --proto_path=../public/common --proto_path=../public/extension --proto_path=../../../../atframework/libatapp-go/protocol config/protocol/config/*.proto
