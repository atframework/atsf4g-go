package main

//go:generate go run .
//go:generate protoc --go_out=extension --go_opt=paths=source_relative --proto_path=extension extension/protocol/extension/*.proto
//go:generate protoc --go_out=common --go_opt=paths=source_relative --proto_path=common --proto_path=extension --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code common/protocol/common/*.proto
//go:generate protoc --go_out=config --go_opt=paths=source_relative --proto_path=config --proto_path=common --proto_path=extension --proto_path=../../../../atframework/libatapp-go/protocol --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code config/protocol/config/*.proto
//go:generate protoc --go_out=pbdesc --go_opt=paths=source_relative --proto_path=pbdesc --proto_path=common --proto_path=extension --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code pbdesc/protocol/pbdesc/*.proto

//go:generate protoc --proto_path=config --proto_path=common --proto_path=extension --proto_path=../../../../atframework/libatapp-go/protocol --proto_path=../../../../third_party/xresloader/protocols/core --proto_path=../../../../third_party/xresloader/protocols/code -o $BuildPbdescPath/config.pb extension/protocol/extension/*.proto common/protocol/common/*.proto config/protocol/config/*.proto ../../../../third_party/xresloader/protocols/core/protocol/extension/v3/xresloader.proto ../../../../third_party/xresloader/protocols/core/protocol/extension/v3/xresloader_ue.proto ../../../../third_party/xresloader/protocols/core/protocol/config/pb_header_v3.proto ../../../../third_party/xresloader/protocols/code/protocol/extension/xrescode_extensions_v3.proto

//go:generate $PythonExecutable $XresloaderPath/xresconv-cli/xresconv-cli.py --java-path $javaExecutable $ExcelGenBytePath/xresconv.xml
