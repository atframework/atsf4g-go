package component_protocol_public

//go:generate protoc --go_out=extension --go_opt=paths=source_relative --proto_path=extension extension/protocol/extension/*.proto
//go:generate protoc --go_out=common --go_opt=paths=source_relative --proto_path=common --proto_path=extension --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code common/protocol/common/*.proto
//go:generate protoc --go_out=config --go_opt=paths=source_relative --proto_path=config --proto_path=common --proto_path=extension --proto_path=../../../../atframework/libatapp-go/protocol --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code config/protocol/config/*.proto
//go:generate protoc --go_out=pbdesc --go_opt=paths=source_relative --proto_path=pbdesc --proto_path=common --proto_path=extension --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code pbdesc/protocol/pbdesc/*.proto
