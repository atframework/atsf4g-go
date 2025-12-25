package libatframe_utils_proto_utility

import (
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

// 使用atapp-go的日志接口会自动调用MessageReadableText打印Message

func MessageReadableText(msg proto.Message) string {
	if lu.IsNil(msg) {
		return "<nil_proto_message>"
	}
	marshaler := prototext.MarshalOptions{
		Multiline: true, // 多行显示
		Indent:    "  ", // 缩进
	}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return ""
	}
	return lu.BytestoString(data)
}

func MessageReadableTextIndent(msg proto.Message) string {
	if lu.IsNil(msg) {
		return "<nil_proto_message>"
	}
	marshaler := prototext.MarshalOptions{
		Multiline: true, // 多行显示
		Indent:    "  ", // 缩进
	}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return ""
	}

	str := lu.BytestoString(data)
	if str == "" {
		return ""
	}
	str = strings.TrimRight(str, "\n")
	return "  " + strings.ReplaceAll(str, "\n", "\n  ") + "\n"
}
