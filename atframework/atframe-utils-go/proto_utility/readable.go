package libatframe_utils_proto_utility

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func MessageReadableText(msg proto.Message) string {
	marshaler := prototext.MarshalOptions{
		Multiline: true, // 多行显示
		Indent:    "  ", // 缩进
	}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return ""
	}
	return string(data)
}
