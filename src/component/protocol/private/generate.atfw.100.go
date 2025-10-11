package component_protocol_private

import (
	_ "github.com/atframework/atframe-utils-go"
)

//go:generate protoc --go_out=extension --go_opt=paths=source_relative --proto_path=extension --proto_path=../public/extension extension/protocol/extension/*.proto
//go:generate protoc --go_out=common --go_opt=paths=source_relative --proto_path=common --proto_path=extension --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../public/common --proto_path=../public/extension common/protocol/common/*.proto
//go:generate protoc --go_out=config --go_opt=paths=source_relative --proto_path=config --proto_path=common --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=extension --proto_path=../public/config --proto_path=../public/common --proto_path=../public/extension --proto_path=../../../../atframework/libatapp-go/protocol config/protocol/config/*.proto
//go:generate protoc --go_out=pbdesc --go_opt=paths=source_relative --proto_path=pbdesc --proto_path=common --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=extension --proto_path=../public/pbdesc --proto_path=../public/common --proto_path=../public/extension --proto_path=../../../../atframework/libatapp-go/protocol pbdesc/protocol/pbdesc/*.proto
