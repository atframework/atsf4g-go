package libatframe_utils_proto_utility

import (
	reflect "reflect"
	sync "sync"
	"testing"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type TestMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Field_1       string                 `protobuf:"bytes,1,opt,name=field_1,json=field1,proto3" json:"field_1,omitempty"`
	Field_2       uint32                 `protobuf:"varint,2,opt,name=field_2,json=field2,proto3" json:"field_2,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TestMessage) Reset() {
	*x = TestMessage{}
	mi := &file_protocol_pbdesc_sample_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TestMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMessage) ProtoMessage() {}

func (x *TestMessage) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_pbdesc_sample_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMessage.ProtoReflect.Descriptor instead.
func (*TestMessage) Descriptor() ([]byte, []int) {
	return file_protocol_pbdesc_sample_proto_rawDescGZIP(), []int{0}
}

func (x *TestMessage) GetField_1() string {
	if x != nil {
		return x.Field_1
	}
	return ""
}

func (x *TestMessage) GetField_2() uint32 {
	if x != nil {
		return x.Field_2
	}
	return 0
}

type Test struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Field_1       string                 `protobuf:"bytes,11,opt,name=field_1,json=field1,proto3" json:"field_1,omitempty"`
	Field_2       []byte                 `protobuf:"bytes,12,opt,name=field_2,json=field2,proto3" json:"field_2,omitempty"`
	Field_3       int32                  `protobuf:"varint,13,opt,name=field_3,json=field3,proto3" json:"field_3,omitempty"`
	Field_4       int32                  `protobuf:"zigzag32,14,opt,name=field_4,json=field4,proto3" json:"field_4,omitempty"`
	Field_5       uint32                 `protobuf:"varint,15,opt,name=field_5,json=field5,proto3" json:"field_5,omitempty"`
	Field_6       int64                  `protobuf:"varint,16,opt,name=field_6,json=field6,proto3" json:"field_6,omitempty"`
	Field_7       int64                  `protobuf:"zigzag64,17,opt,name=field_7,json=field7,proto3" json:"field_7,omitempty"`
	Field_8       uint64                 `protobuf:"varint,18,opt,name=field_8,json=field8,proto3" json:"field_8,omitempty"`
	Field_9       bool                   `protobuf:"varint,19,opt,name=field_9,json=field9,proto3" json:"field_9,omitempty"`
	Field_10      float32                `protobuf:"fixed32,20,opt,name=field_10,json=field10,proto3" json:"field_10,omitempty"`
	Field_11      float64                `protobuf:"fixed64,21,opt,name=field_11,json=field11,proto3" json:"field_11,omitempty"`
	Field_12      *TestMessage           `protobuf:"bytes,22,opt,name=field_12,json=field12,proto3" json:"field_12,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Test) Reset() {
	*x = Test{}
	mi := &file_protocol_pbdesc_sample_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Test) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Test) ProtoMessage() {}

func (x *Test) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_pbdesc_sample_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Test.ProtoReflect.Descriptor instead.
func (*Test) Descriptor() ([]byte, []int) {
	return file_protocol_pbdesc_sample_proto_rawDescGZIP(), []int{1}
}

func (x *Test) GetField_1() string {
	if x != nil {
		return x.Field_1
	}
	return ""
}

func (x *Test) GetField_2() []byte {
	if x != nil {
		return x.Field_2
	}
	return nil
}

func (x *Test) GetField_3() int32 {
	if x != nil {
		return x.Field_3
	}
	return 0
}

func (x *Test) GetField_4() int32 {
	if x != nil {
		return x.Field_4
	}
	return 0
}

func (x *Test) GetField_5() uint32 {
	if x != nil {
		return x.Field_5
	}
	return 0
}

func (x *Test) GetField_6() int64 {
	if x != nil {
		return x.Field_6
	}
	return 0
}

func (x *Test) GetField_7() int64 {
	if x != nil {
		return x.Field_7
	}
	return 0
}

func (x *Test) GetField_8() uint64 {
	if x != nil {
		return x.Field_8
	}
	return 0
}

func (x *Test) GetField_9() bool {
	if x != nil {
		return x.Field_9
	}
	return false
}

func (x *Test) GetField_10() float32 {
	if x != nil {
		return x.Field_10
	}
	return 0
}

func (x *Test) GetField_11() float64 {
	if x != nil {
		return x.Field_11
	}
	return 0
}

func (x *Test) GetField_12() *TestMessage {
	if x != nil {
		return x.Field_12
	}
	return nil
}

var File_protocol_pbdesc_sample_proto protoreflect.FileDescriptor

const file_protocol_pbdesc_sample_proto_rawDesc = "" +
	"\n" +
	"\x1cprotocol/pbdesc/sample.proto\x12\x04proy\"?\n" +
	"\vTestMessage\x12\x17\n" +
	"\afield_1\x18\x01 \x01(\tR\x06field1\x12\x17\n" +
	"\afield_2\x18\x02 \x01(\rR\x06field2\"\xcb\x02\n" +
	"\x04Test\x12\x17\n" +
	"\afield_1\x18\v \x01(\tR\x06field1\x12\x17\n" +
	"\afield_2\x18\f \x01(\fR\x06field2\x12\x17\n" +
	"\afield_3\x18\r \x01(\x05R\x06field3\x12\x17\n" +
	"\afield_4\x18\x0e \x01(\x11R\x06field4\x12\x17\n" +
	"\afield_5\x18\x0f \x01(\rR\x06field5\x12\x17\n" +
	"\afield_6\x18\x10 \x01(\x03R\x06field6\x12\x17\n" +
	"\afield_7\x18\x11 \x01(\x12R\x06field7\x12\x17\n" +
	"\afield_8\x18\x12 \x01(\x04R\x06field8\x12\x17\n" +
	"\afield_9\x18\x13 \x01(\bR\x06field9\x12\x19\n" +
	"\bfield_10\x18\x14 \x01(\x02R\afield10\x12\x19\n" +
	"\bfield_11\x18\x15 \x01(\x01R\afield11\x12,\n" +
	"\bfield_12\x18\x16 \x01(\v2\x11.proy.TestMessageR\afield12B\\H\x01ZRgithub.com/atframework/atsf4g-go/service-lobbysvr/protocol/private/protocol/pbdesc\x80\x01\x01\xf8\x01\x01b\x06proto3"

var (
	file_protocol_pbdesc_sample_proto_rawDescOnce sync.Once
	file_protocol_pbdesc_sample_proto_rawDescData []byte
)

func file_protocol_pbdesc_sample_proto_rawDescGZIP() []byte {
	file_protocol_pbdesc_sample_proto_rawDescOnce.Do(func() {
		file_protocol_pbdesc_sample_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_protocol_pbdesc_sample_proto_rawDesc), len(file_protocol_pbdesc_sample_proto_rawDesc)))
	})
	return file_protocol_pbdesc_sample_proto_rawDescData
}

var file_protocol_pbdesc_sample_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_protocol_pbdesc_sample_proto_goTypes = []any{
	(*TestMessage)(nil), // 0: proy.TestMessage
	(*Test)(nil),        // 1: proy.Test
}
var file_protocol_pbdesc_sample_proto_depIdxs = []int32{
	0, // 0: proy.Test.field_12:type_name -> proy.TestMessage
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_protocol_pbdesc_sample_proto_init() }
func file_protocol_pbdesc_sample_proto_init() {
	if File_protocol_pbdesc_sample_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_protocol_pbdesc_sample_proto_rawDesc), len(file_protocol_pbdesc_sample_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protocol_pbdesc_sample_proto_goTypes,
		DependencyIndexes: file_protocol_pbdesc_sample_proto_depIdxs,
		MessageInfos:      file_protocol_pbdesc_sample_proto_msgTypes,
	}.Build()
	File_protocol_pbdesc_sample_proto = out.File
	file_protocol_pbdesc_sample_proto_goTypes = nil
	file_protocol_pbdesc_sample_proto_depIdxs = nil
}

// 测试基本功能
func TestRedisPb(t *testing.T) {
	message := Test{
		Field_1:  "231",
		Field_2:  []byte("232"),
		Field_3:  233,
		Field_4:  234,
		Field_5:  235,
		Field_6:  236,
		Field_7:  237,
		Field_8:  238,
		Field_9:  true,
		Field_10: 230.3,
		Field_11: 231.3,
		Field_12: &TestMessage{
			Field_1: "2321",
			Field_2: 2322,
		},
	}

	var casVersion uint64 = 10
	redis := PBMapToRedis(&message, &casVersion, true)
	if len(redis) != 26 {
		t.Fatalf("len not match %d", len(redis))
	}
	data := make(map[string]string)
	for i := 0; i < len(redis); i += 2 {
		key := redis[i]
		value := redis[i+1]
		data[key] = value
	}

	newMessage := &Test{}
	retVersion, err := RedisMapToPB(data, newMessage)
	if err != nil {
		t.Fatalf("failed %s", err)
	}
	if retVersion != casVersion {
		t.Error("not match")
	}
	if newMessage.Field_1 != message.Field_1 {
		t.Error("not match")
	}
	if string(newMessage.Field_2) != string(message.Field_2) {
		t.Error("not match")
	}
	if newMessage.Field_3 != message.Field_3 {
		t.Error("not match")
	}
	if newMessage.Field_4 != message.Field_4 {
		t.Error("not match")
	}
	if newMessage.Field_5 != message.Field_5 {
		t.Error("not match")
	}
	if newMessage.Field_6 != message.Field_6 {
		t.Error("not match")
	}
	if newMessage.Field_7 != message.Field_7 {
		t.Error("not match")
	}
	if newMessage.Field_8 != message.Field_8 {
		t.Error("not match")
	}
	if newMessage.Field_9 != message.Field_9 {
		t.Error("not match")
	}
	if newMessage.Field_10 != message.Field_10 {
		t.Error("not match")
	}
	if newMessage.Field_11 != message.Field_11 {
		t.Error("not match")
	}
	if newMessage.Field_12.Field_1 != message.Field_12.Field_1 {
		t.Error("not match")
	}
	if newMessage.Field_12.Field_2 != message.Field_12.Field_2 {
		t.Error("not match")
	}
}
