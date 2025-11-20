package main

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	anypb "google.golang.org/protobuf/types/known/anypb"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// GoTypeString 返回该字段在生成的 Go 代码中的类型名（字符串形式）
func GoTypeString(g *protogen.GeneratedFile, f *protogen.Field, withoutPointer bool) string {
	// map
	if f.Desc.IsMap() {
		keyField := f.Message.Fields[0]
		valField := f.Message.Fields[1]
		keyType := scalarGoType(keyField)
		valType := elementGoType(g, valField, false)
		return fmt.Sprintf("map[%s]%s", keyType, valType)
	}

	// repeated
	if f.Desc.IsList() {
		elemType := elementGoType(g, f, false)
		return "[]" + elemType
	}

	// 单值
	return elementGoType(g, f, withoutPointer)
}

// elementGoType 返回字段元素（非 repeated）的 Go 类型
func elementGoType(g *protogen.GeneratedFile, f *protogen.Field, withoutPointer bool) string {
	switch f.Desc.Kind() {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	case protoreflect.FloatKind:
		return "float32"
	case protoreflect.DoubleKind:
		return "float64"
	case protoreflect.BytesKind:
		return "[]byte"
	case protoreflect.EnumKind:
		// Enum 类型直接返回枚举名
		return g.QualifiedGoIdent(f.Enum.GoIdent)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		// Message 类型生成的 Go 字段通常是指针
		if withoutPointer {
			return g.QualifiedGoIdent(f.Message.GoIdent)
		}
		return "*" + g.QualifiedGoIdent(f.Message.GoIdent)
	default:
		return "interface{}"
	}
}

// scalarGoType 返回 map key 的 Go 基础类型
func scalarGoType(f *protogen.Field) string {
	switch f.Desc.Kind() {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	default:
		return "interface{}"
	}
}

func readonlyFieldVarName(f *protogen.Field) string {
	return fmt.Sprintf("field%s", f.GoName)
}

func readonlyOneofFieldVarName(oneof *protogen.Oneof) string {
	return fmt.Sprintf("field%s", oneof.GoName)
}

func readonlyOneofInterfaceName(msg *protogen.Message, oneof *protogen.Oneof) string {
	return fmt.Sprintf("getReadonly%s_%s", msg.GoIdent.GoName, oneof.GoName)
}

func readonlyOneofStructName(field *protogen.Field) string {
	return fmt.Sprintf("Readonly_%s", field.GoIdent.GoName)
}

func hasReadonlyWrapper(parent *protogen.Message, target *protogen.Message) bool {
	if parent == nil || target == nil {
		return false
	}

	if target.Desc.FullName() == (&durationpb.Duration{}).ProtoReflect().Descriptor().FullName() ||
		target.Desc.FullName() == (&timestamppb.Timestamp{}).ProtoReflect().Descriptor().FullName() ||
		target.Desc.FullName() == (&anypb.Any{}).ProtoReflect().Descriptor().FullName() ||
		target.Desc.FullName() == (&emptypb.Empty{}).ProtoReflect().Descriptor().FullName() {
		return false
	}

	return true
}

func qualifiedReadonlyGoIdent(gf *protogen.GeneratedFile, gi protogen.GoIdent) string {
	name := gf.QualifiedGoIdent(gi)
	index := strings.Index(name, ".")
	if index == -1 {
		return "Readonly_" + name
	}

	// 将字符串分割并插入
	return name[:index+1] + "Readonly_" + name[index+1:]
}

func readonlyElementGoType(g *protogen.GeneratedFile, parent *protogen.Message, f *protogen.Field) string {
	switch f.Desc.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if hasReadonlyWrapper(parent, f.Message) {
			return "*" + qualifiedReadonlyGoIdent(g, f.Message.GoIdent)
		}
		return "*" + g.QualifiedGoIdent(f.Message.GoIdent)
	default:
		return elementGoType(g, f, false)
	}
}

func readonlyFieldGoType(g *protogen.GeneratedFile, parent *protogen.Message, f *protogen.Field) string {
	switch {
	case f.Desc.IsMap():
		keyField := f.Message.Fields[0]
		valField := f.Message.Fields[1]
		return fmt.Sprintf("map[%s]%s", scalarGoType(keyField), readonlyElementGoType(g, parent, valField))
	case f.Desc.IsList():
		return "[]" + readonlyElementGoType(g, parent, f)
	default:
		return readonlyElementGoType(g, parent, f)
	}
}

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, f := range plugin.Files {
			if !f.Generate {
				continue
			}
			generateFile(plugin, f)
		}
		return nil
	})
}

var initGenerate []string

func generateFile(plugin *protogen.Plugin, f *protogen.File) {
	filename := f.GeneratedFilenamePrefix + "_mutable.pb.go"
	g := plugin.NewGeneratedFile(filename, f.GoImportPath)

	g.P("// Code generated by protoc-gen-mutable. DO NOT EDIT.")
	g.P("// source: ", f.Desc.Path())
	g.P()
	g.P("package ", f.GoPackageName)
	g.P()
	if len(f.Messages) != 0 {
		g.P("import \"google.golang.org/protobuf/proto\"")
		g.P("import \"log/slog\"")
		g.P("import pu \"github.com/atframework/atframe-utils-go/proto_utility\"")
		g.P("import \"reflect\"")
		g.P("import protoreflect \"google.golang.org/protobuf/reflect/protoreflect\"")
		g.P()

		g.P(fmt.Sprintf(`type inner%sReadonlyMessage interface {`, f.GoDescriptorIdent.GoName))
		g.P("	Descriptor() protoreflect.MessageDescriptor")
		g.P("	Type() protoreflect.MessageType")
		g.P("	New() protoreflect.Message")
		g.P("	Interface() protoreflect.ProtoMessage")
		g.P("	Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool)")
		g.P("	Has(protoreflect.FieldDescriptor) bool")
		g.P("	Get(protoreflect.FieldDescriptor) protoreflect.Value")
		g.P("	WhichOneof(protoreflect.OneofDescriptor) protoreflect.FieldDescriptor")
		g.P("	GetUnknown() protoreflect.RawFields")
		g.P("	IsValid() bool")
		g.P("}")
	}

	for _, msg := range f.Messages {
		generateMutableForMessage(f, g, msg)
	}

	if len(initGenerate) != 0 {
		g.P("func init() {")
		for _, initCode := range initGenerate {
			g.P(initCode)
		}
		g.P("}")
		g.P()
	}
	initGenerate = nil
}

func generateMutableForMessage(f *protogen.File, g *protogen.GeneratedFile, msg *protogen.Message) {
	if msg.Desc.IsMapEntry() {
		return
	}

	g.P("// ===== Clone methods for ", msg.GoIdent.GoName, " ===== Message ====")
	g.P(fmt.Sprintf(`func (m *%s) Clone() *%s {`, msg.GoIdent.GoName, msg.GoIdent.GoName))
	g.P(`  if m == nil {`)
	g.P(fmt.Sprintf(`    return new(%s)`, msg.GoIdent.GoName))
	g.P(`  }`)
	g.P(fmt.Sprintf(`  return proto.Clone(m).(*%s)`, msg.GoIdent.GoName))
	g.P(`}`)
	g.P()

	g.P("// ===== Merge methods for ", msg.GoIdent.GoName, " ===== Message ====")
	g.P(fmt.Sprintf(`func (m *%s) Merge(src *%s) {`, msg.GoIdent.GoName, msg.GoIdent.GoName))
	g.P(`  if m == nil {`)
	g.P(`    return`)
	g.P(`  }`)
	g.P(`  proto.Merge(m, src)`)
	g.P(`}`)
	g.P()

	g.P("// ===== SlogValue methods for ", msg.GoIdent.GoName, " ===== Message ====")
	g.P(fmt.Sprintf(`func (m *%s) LogValue() slog.Value {`, msg.GoIdent.GoName))
	g.P(`  return slog.StringValue(pu.MessageReadableText(m))`)
	g.P(`}`)
	g.P()

	g.P("// ===== GetMessageReflectType methods for ", msg.GoIdent.GoName, " ===== Message =====")
	g.P(fmt.Sprintf(`	var ReflectType%s reflect.Type`, msg.GoIdent.GoName))
	g.P(fmt.Sprintf(`func GetReflectType%s() reflect.Type {`, msg.GoIdent.GoName))
	g.P(fmt.Sprintf(`  return ReflectType%s`, msg.GoIdent.GoName))
	g.P(`}`)
	g.P()
	initGenerate = append(initGenerate, fmt.Sprintf(`	ReflectType%s = reflect.TypeOf((*%s)(nil)).Elem()`, msg.GoIdent.GoName, msg.GoIdent.GoName))

	generateReadonlyForMessage(f, g, msg)

	for _, oneof := range msg.Oneofs {
		oneofName := oneof.GoName
		fullFieldName := fmt.Sprintf("%s_%s", msg.GoIdent.GoName, oneofName)

		g.P("// ===== Case Enum for ", msg.GoIdent.GoName, " Oneof ", oneofName, " ===== Oneof =====")
		g.P(fmt.Sprintf(`type %s_En%sID int32`, msg.GoIdent.GoName, oneofName))
		g.P("const (")
		g.P(fmt.Sprintf(`	%s_En%sID_%s %s_En%sID = 0 // none`, msg.GoIdent.GoName, oneofName, "NONE", msg.GoIdent.GoName, oneofName))
		for _, o := range oneof.Fields {
			g.P(fmt.Sprintf(`	%s_En%sID_%s %s_En%sID = %d // %s`, msg.GoIdent.GoName, oneofName, o.GoName, msg.GoIdent.GoName, oneofName, o.Desc.Number(), o.Desc.TextName()))
		}
		g.P(")")
		g.P()

		g.P("// ===== GetCase interface for ", msg.GoIdent.GoName, " Oneof ", oneofName, " ===== Oneof =====")
		g.P(fmt.Sprintf(`type get%s interface {`, fullFieldName))
		g.P(fmt.Sprintf(`	Get%s() %s_En%sID`, fullFieldName, msg.GoIdent.GoName, oneofName))
		g.P(fmt.Sprintf(`	GetReflectType%s() reflect.Type`, fullFieldName))
		g.P("}")
		g.P()

		g.P("// ===== GetCase methods for ", msg.GoIdent.GoName, " Oneof ", oneofName, " ===== Oneof =====")
		g.P(fmt.Sprintf(`func (m *%s) Get%sOneofCase() %s_En%sID {`, msg.GoIdent.GoName, oneofName, msg.GoIdent.GoName, oneofName))
		g.P(`  if m == nil {`)
		g.P(`    return 0`)
		g.P(`  }`)
		g.P(fmt.Sprintf(`	v, ok := m.%s.(get%s)`, oneofName, fullFieldName))
		g.P(`  if !ok {`)
		g.P(`		return 0`)
		g.P(`  }`)
		g.P(fmt.Sprintf(`	return v.Get%s()`, fullFieldName))
		g.P(`}`)
		g.P()

		g.P("// ===== GetOneofReflectType methods for ", msg.GoIdent.GoName, " Oneof ", oneofName, " ===== Oneof =====")
		g.P(fmt.Sprintf(`func (m *%s) Get%sReflectType() reflect.Type {`, msg.GoIdent.GoName, oneofName))
		g.P(`  if m == nil {`)
		g.P(`    return nil`)
		g.P(`  }`)
		g.P(fmt.Sprintf(`	v, ok := m.%s.(get%s)`, oneofName, fullFieldName))
		g.P(`  if !ok {`)
		g.P(`		return nil`)
		g.P(`  }`)
		g.P(fmt.Sprintf(`	return v.GetReflectType%s()`, fullFieldName))
		g.P(`}`)
		g.P()
	}

	for _, field := range msg.Fields {
		fieldName := field.GoName
		switch {
		case field.Desc.IsMap():
			g.P("// ===== Mutable methods for ", msg.GoIdent.GoName, " ===== Map =====")
			fieldType := GoTypeString(g, field, false)
			g.P(fmt.Sprintf(`func (m *%s) Mutable%s() %s {`, msg.GoIdent.GoName, fieldName, fieldType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = make(%s, 0)`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`  return m.%s`, fieldName))
			g.P(`}`)
			g.P()
		case field.Desc.IsList():
			fieldType := GoTypeString(g, field, false)
			elementType := elementGoType(g, field, false)

			g.P("// ===== Mutable methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
			g.P(fmt.Sprintf(`func (m *%s) Mutable%s() %s {`, msg.GoIdent.GoName, fieldName, fieldType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = %s{}`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`  return m.%s`, fieldName))
			g.P(`}`)
			g.P()

			g.P("// ===== ReverseIfNil methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
			g.P(fmt.Sprintf(`func (m *%s) ReverseIfNil%s(l int32) %s {`, msg.GoIdent.GoName, fieldName, fieldType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = make(%s, 0, l)`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`  return m.%s`, fieldName))
			g.P(`}`)
			g.P()

			g.P("// ===== Append methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
			g.P(fmt.Sprintf(`func (m *%s) Append%s(d %s) {`, msg.GoIdent.GoName, fieldName, elementType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = %s{}`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`    m.%s = append(m.%s, d)`, fieldName, fieldName))
			g.P(`}`)
			g.P()

			if field.Message != nil {
				// Message Add
				g.P("// ===== Add methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
				g.P(fmt.Sprintf(`func (m *%s) Add%s() %s {`, msg.GoIdent.GoName, fieldName, elementType))
				g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
				g.P(fmt.Sprintf(`    m.%s = %s{}`, fieldName, fieldType))
				g.P(`  }`)
				g.P(fmt.Sprintf(`  addValue := new(%s)`, elementGoType(g, field, true)))
				g.P(fmt.Sprintf(`    m.%s = append(m.%s, addValue)`, fieldName, fieldName))
				g.P(`	return addValue`)
				g.P(`}`)
				g.P()
			}

			g.P("// ===== Merge methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
			g.P(fmt.Sprintf(`func (m *%s) Merge%s(d %s) %s {`, msg.GoIdent.GoName, fieldName, fieldType, fieldType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = %s{}`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`    m.%s = append(m.%s, d...)`, fieldName, fieldName))
			g.P(fmt.Sprintf(`  return m.%s`, fieldName))
			g.P(`}`)
			g.P()

			g.P("// ===== RemoveLast methods for ", msg.GoIdent.GoName, " ===== Repeated =====")
			g.P(fmt.Sprintf(`func (m *%s) RemoveLast%s() {`, msg.GoIdent.GoName, fieldName))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = %s{}`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`  if len(m.%s) != 0 {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = m.%s[:len(m.%s)-1]`, fieldName, fieldName, fieldName))
			g.P(`}`)
			g.P(`}`)
			g.P()
		case field.Oneof != nil:
			g.P("// ===== Mutable methods for ", msg.GoIdent.GoName, " ===== Oneof =====")
			oneofName := field.Oneof.GoName
			fullFieldName := fmt.Sprintf("%s_%s", msg.GoIdent.GoName, fieldName)
			g.P(fmt.Sprintf(`func (m *%s) Mutable%s() *%s {`, msg.GoIdent.GoName, fieldName, fullFieldName))
			g.P(fmt.Sprintf(`  if x, ok := m.%s.(*%s); ok {`, oneofName, fullFieldName))
			g.P(`    return x`)
			g.P(`  }`)
			g.P(fmt.Sprintf(`  x := new(%s)`, fullFieldName))
			g.P(fmt.Sprintf(`  m.%s = x`, oneofName))
			g.P(`  return x`)
			g.P(`}`)
			g.P()

			g.P("// ===== Get reflect Type for ", msg.GoIdent.GoName, " Oneof ", oneofName, " ===== Oneof =====")
			g.P(fmt.Sprintf(`	var ReflectType%s reflect.Type`, fullFieldName))
			g.P(fmt.Sprintf(`func GetReflectType%s() reflect.Type {`, fullFieldName))
			g.P(fmt.Sprintf(`	return ReflectType%s`, fullFieldName))
			g.P(`}`)
			g.P()
			initGenerate = append(initGenerate, fmt.Sprintf(`	ReflectType%s = reflect.TypeOf((*%s)(nil)).Elem()`, fullFieldName, fullFieldName))

			g.P("// ===== Oneof Interface for ", msg.GoIdent.GoName, " Oneof ", fullFieldName, " ===== Oneof =====")
			g.P(fmt.Sprintf(`func (m *%s) Get%s_%s() %s_En%sID {`, fullFieldName, msg.GoIdent.GoName, oneofName, msg.GoIdent.GoName, oneofName))
			g.P(fmt.Sprintf(`  return %s_En%sID_%s`, msg.GoIdent.GoName, oneofName, fieldName))
			g.P(`}`)
			g.P(fmt.Sprintf(`func (m *%s) GetReflectType%s_%s() reflect.Type {`, fullFieldName, msg.GoIdent.GoName, oneofName))
			g.P(fmt.Sprintf(`  return ReflectType%s`, fullFieldName))
			g.P(`}`)
			g.P()
		case field.Message != nil:
			g.P("// ===== Mutable methods for ", msg.GoIdent.GoName, " ===== Message =====")
			fieldType := GoTypeString(g, field, true)
			g.P(fmt.Sprintf(`func (m *%s) Mutable%s() *%s {`, msg.GoIdent.GoName, fieldName, fieldType))
			g.P(fmt.Sprintf(`  if m.%s == nil {`, fieldName))
			g.P(fmt.Sprintf(`    m.%s = new(%s)`, fieldName, fieldType))
			g.P(`  }`)
			g.P(fmt.Sprintf(`  return m.%s`, fieldName))
			g.P(`}`)
			g.P()
		default:
			// 基本类型不生成 Mutable 方法
		}
	}

	for _, nested := range msg.Messages {
		generateMutableForMessage(f, g, nested)
	}
}

func generateReadonlyForMessage(f *protogen.File, g *protogen.GeneratedFile, msg *protogen.Message) {
	roName := "Readonly_" + msg.GoIdent.GoName
	g.P("// ===== Readonly wrapper for ", msg.GoIdent.GoName, " ===== Message ====")
	g.P(fmt.Sprintf("type %s struct {", roName))
	g.P(fmt.Sprintf("  protoData *%s", msg.GoIdent.GoName))
	for _, field := range msg.Fields {
		if field.Oneof != nil {
			continue
		}
		g.P(fmt.Sprintf("  %s %s", readonlyFieldVarName(field), readonlyFieldGoType(g, msg, field)))
	}
	for _, oneof := range msg.Oneofs {
		g.P(fmt.Sprintf("  %s %s", readonlyOneofFieldVarName(oneof), readonlyOneofInterfaceName(msg, oneof)))
	}
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func (r *%s) ReadonlyProtoReflect() inner%sReadonlyMessage {", roName, f.GoDescriptorIdent.GoName))
	g.P("  if r == nil || r.protoData == nil {")
	g.P("    return nil")
	g.P("  }")
	g.P("  return r.protoData.ProtoReflect()")
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func (m *%s) ToReadonly() *%s {", msg.GoIdent.GoName, roName))
	g.P("  if m == nil {")
	g.P("    return nil")
	g.P("  }")
	g.P("  clone := m.Clone()")
	g.P(fmt.Sprintf("  ro := &%s{protoData: clone}", roName))
	g.P("  ro.initFromProto(clone)")
	g.P("  return ro")
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func (r *%s) CloneMessage() *%s {", roName, msg.GoIdent.GoName))
	g.P("  if r == nil {")
	g.P(fmt.Sprintf("    return new(%s)", msg.GoIdent.GoName))
	g.P("  }")
	g.P("  return r.protoData.Clone()")
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func (r *%s) ToMessage() *%s {", roName, msg.GoIdent.GoName))
	g.P("  return r.CloneMessage()")
	g.P("}")
	g.P()

	g.P(fmt.Sprintf("func (r *%s) initFromProto(src *%s) {", roName, msg.GoIdent.GoName))
	g.P("  if r == nil || src == nil {")
	g.P("    return")
	g.P("  }")
	for _, field := range msg.Fields {
		if field.Oneof != nil {
			continue
		}
		generateReadonlyFieldCopy(g, msg, field)
	}
	for _, oneof := range msg.Oneofs {
		generateReadonlyOneofCopy(g, msg, oneof)
	}
	g.P("}")
	g.P()

	for _, field := range msg.Fields {
		if field.Oneof != nil {
			generateReadonlyOneofGetter(g, roName, field)
			continue
		}
		generateReadonlyGetter(g, roName, field)
	}
	for _, oneof := range msg.Oneofs {
		oneofVar := readonlyOneofFieldVarName(oneof)
		fullFieldName := fmt.Sprintf("%s_%s", msg.GoIdent.GoName, oneof.GoName)
		g.P(fmt.Sprintf("func (r *%s) Get%sOneofCase() %s_En%sID {", roName, oneof.GoName, msg.GoIdent.GoName, oneof.GoName))
		g.P("  if r == nil {")
		g.P("    return 0")
		g.P("  }")
		g.P(fmt.Sprintf("  if r.%s == nil {", oneofVar))
		g.P(fmt.Sprintf("    return %s_En%sID_NONE", msg.GoIdent.GoName, oneof.GoName))
		g.P("  }")
		g.P(fmt.Sprintf("  return r.%s.Get%s()", oneofVar, fullFieldName))
		g.P("}")
		g.P()

		g.P(fmt.Sprintf("func (r *%s) Get%sReflectType() reflect.Type {", roName, oneof.GoName))
		g.P("  if r == nil {")
		g.P("    return nil")
		g.P("  }")
		g.P(fmt.Sprintf("  if r.%s == nil {", oneofVar))
		g.P("    return nil")
		g.P("  }")
		g.P(fmt.Sprintf("  return r.%s.GetReflectType%s()", oneofVar, fullFieldName))
		g.P("}")
		g.P()
		generateReadonlyOneofInterface(g, msg, oneof)
	}
}

func generateReadonlyFieldCopy(g *protogen.GeneratedFile, msg *protogen.Message, field *protogen.Field) {
	if field.Oneof != nil {
		return
	}
	fieldVar := readonlyFieldVarName(field)
	switch {
	case field.Desc.IsMap():
		keyField := field.Message.Fields[0]
		valField := field.Message.Fields[1]
		g.P(fmt.Sprintf("  if v := src.Get%s(); len(v) != 0 {", field.GoName))
		g.P(fmt.Sprintf("    copied := make(map[%s]%s, len(v))", scalarGoType(keyField), readonlyElementGoType(g, msg, valField)))
		g.P("    for mk, mv := range v {")
		switch {
		case valField.Message != nil:
			g.P("      if mv == nil {")
			g.P("        copied[mk] = nil")
			g.P("        continue")
			g.P("      }")
			if hasReadonlyWrapper(msg, valField.Message) {
				g.P("      copied[mk] = mv.ToReadonly()")
			} else {
				g.P("      copied[mk] = proto.Clone(mv).(*", g.QualifiedGoIdent(valField.Message.GoIdent), ")")
			}
		case valField.Desc.Kind() == protoreflect.BytesKind:
			g.P("      if len(mv) != 0 {")
			g.P("        copied[mk] = append([]byte(nil), mv...)")
			g.P("      } else {")
			g.P("        copied[mk] = nil")
			g.P("      }")
		default:
			g.P("      copied[mk] = mv")
		}
		g.P("    }")
		g.P(fmt.Sprintf("    r.%s = copied", fieldVar))
		g.P("  } else {")
		g.P(fmt.Sprintf("    r.%s = nil", fieldVar))
		g.P("  }")
	case field.Desc.IsList():
		g.P(fmt.Sprintf("  if v := src.Get%s(); len(v) != 0 {", field.GoName))
		switch {
		case field.Message != nil:
			g.P(fmt.Sprintf("    out := make([]%s, 0, len(v))", readonlyElementGoType(g, msg, field)))
			g.P("    for _, item := range v {")
			g.P("      if item == nil {")
			g.P("        out = append(out, nil)")
			g.P("        continue")
			g.P("      }")
			if hasReadonlyWrapper(msg, field.Message) {
				g.P("      out = append(out, item.ToReadonly())")
			} else {
				g.P("      out = append(out, proto.Clone(item).(*", g.QualifiedGoIdent(field.Message.GoIdent), "))")
			}
			g.P("    }")
		case field.Desc.Kind() == protoreflect.BytesKind:
			g.P("    out := make([][]byte, len(v))")
			g.P("    for i := range v {")
			g.P("      if len(v[i]) != 0 {")
			g.P("        out[i] = append([]byte(nil), v[i]...)")
			g.P("      }")
			g.P("    }")
		default:
			g.P(fmt.Sprintf("    copied := make([]%s, len(v))", elementGoType(g, field, false)))
			g.P("    copy(copied, v)")
			g.P("    r.", fieldVar, " = copied")
		}
		if field.Message != nil || field.Desc.Kind() == protoreflect.BytesKind {
			g.P(fmt.Sprintf("    r.%s = out", fieldVar))
		}
		g.P("  } else {")
		g.P(fmt.Sprintf("    r.%s = nil", fieldVar))
		g.P("  }")
	case field.Message != nil:
		g.P(fmt.Sprintf("  if v := src.Get%s(); v != nil {", field.GoName))
		if hasReadonlyWrapper(msg, field.Message) {
			g.P(fmt.Sprintf("    r.%s = v.ToReadonly()", fieldVar))
		} else {
			g.P(fmt.Sprintf("    r.%s = proto.Clone(v).(*%s)", fieldVar, g.QualifiedGoIdent(field.Message.GoIdent)))
		}
		g.P("  } else {")
		g.P(fmt.Sprintf("    r.%s = nil", fieldVar))
		g.P("  }")
	case field.Desc.Kind() == protoreflect.BytesKind:
		g.P(fmt.Sprintf("  if v := src.Get%s(); len(v) != 0 {", field.GoName))
		g.P(fmt.Sprintf("    r.%s = append([]byte(nil), v...)", fieldVar))
		g.P("  } else {")
		g.P(fmt.Sprintf("    r.%s = nil", fieldVar))
		g.P("  }")
	default:
		g.P(fmt.Sprintf("  r.%s = src.Get%s()", fieldVar, field.GoName))
	}
}

func generateReadonlyGetter(g *protogen.GeneratedFile, roName string, field *protogen.Field) {
	if field.Oneof != nil {
		return
	}
	fieldType := readonlyFieldGoType(g, field.Parent, field)
	fieldVar := readonlyFieldVarName(field)
	g.P(fmt.Sprintf("func (r *%s) Get%s() %s {", roName, field.GoName, fieldType))
	g.P("  if r == nil {")
	g.P(fmt.Sprintf("    var zero %s", fieldType))
	g.P("    return zero")
	g.P("  }")
	if !field.Desc.IsMap() && !field.Desc.IsList() && field.Desc.Kind() == protoreflect.BytesKind {
		g.P(fmt.Sprintf("  if len(r.%s) == 0 {", fieldVar))
		g.P(fmt.Sprintf("    return r.%s", fieldVar))
		g.P("  }")
		g.P(fmt.Sprintf("  dup := make([]byte, len(r.%s))", fieldVar))
		g.P(fmt.Sprintf("  copy(dup, r.%s)", fieldVar))
		g.P("  return dup")
	} else {
		g.P(fmt.Sprintf("  return r.%s", fieldVar))
	}
	g.P("}")
	g.P()
}

func generateReadonlyOneofGetter(g *protogen.GeneratedFile, roName string, field *protogen.Field) {
	fieldType := readonlyElementGoType(g, field.Parent, field)
	oneofVar := readonlyOneofFieldVarName(field.Oneof)
	structName := readonlyOneofStructName(field)
	g.P(fmt.Sprintf("func (r *%s) Get%s() %s {", roName, field.GoName, fieldType))
	g.P("  if r == nil {")
	g.P(fmt.Sprintf("    var zero %s", fieldType))
	g.P("    return zero")
	g.P("  }")
	g.P(fmt.Sprintf("  if x, ok := r.%s.(*%s); ok && x != nil {", oneofVar, structName))
	g.P(fmt.Sprintf("    return x.Get%s()", field.GoName))
	g.P("  }")
	g.P(fmt.Sprintf("  var zero %s", fieldType))
	g.P("  return zero")
	g.P("}")
	g.P()
}

func generateReadonlyOneofCopy(g *protogen.GeneratedFile, msg *protogen.Message, oneof *protogen.Oneof) {
	oneofVar := readonlyOneofFieldVarName(oneof)
	g.P(fmt.Sprintf("  switch v := src.%s.(type) {", oneof.GoName))
	for _, field := range oneof.Fields {
		structName := readonlyOneofStructName(field)
		valueType := readonlyElementGoType(g, msg, field)
		g.P(fmt.Sprintf("  case *%s:", field.GoIdent.GoName))
		switch {
		case field.Message != nil:
			g.P(fmt.Sprintf("    var inner %s", valueType))
			g.P(fmt.Sprintf("    if nested := v.%s; nested != nil {", field.GoName))
			g.P("      inner = nested.ToReadonly()")
			g.P("    }")
			g.P(fmt.Sprintf("    r.%s = &%s{value: inner}", oneofVar, structName))
		case field.Desc.Kind() == protoreflect.BytesKind:
			g.P("    var copied []byte")
			g.P(fmt.Sprintf("    if data := v.%s; len(data) != 0 {", field.GoName))
			g.P("      copied = append([]byte(nil), data...)")
			g.P("    }")
			g.P(fmt.Sprintf("    r.%s = &%s{value: copied}", oneofVar, structName))
		default:
			g.P(fmt.Sprintf("    r.%s = &%s{value: v.%s}", oneofVar, structName, field.GoName))
		}
	}
	g.P("  default:")
	g.P(fmt.Sprintf("    r.%s = nil", oneofVar))
	g.P("  }")
}

func generateReadonlyOneofInterface(g *protogen.GeneratedFile, msg *protogen.Message, oneof *protogen.Oneof) {
	ifaceName := readonlyOneofInterfaceName(msg, oneof)
	fullFieldName := fmt.Sprintf("%s_%s", msg.GoIdent.GoName, oneof.GoName)
	g.P("// ===== Readonly interface for ", msg.GoIdent.GoName, " Oneof ", oneof.GoName, " ===== Oneof ====")
	g.P(fmt.Sprintf("type %s interface {", ifaceName))
	g.P(fmt.Sprintf("  Get%s() %s_En%sID", fullFieldName, msg.GoIdent.GoName, oneof.GoName))
	g.P(fmt.Sprintf("  GetReflectType%s() reflect.Type", fullFieldName))
	g.P("}")
	g.P()
	for _, field := range oneof.Fields {
		structName := readonlyOneofStructName(field)
		valueType := readonlyElementGoType(g, msg, field)
		g.P("// ===== Readonly struct for ", field.GoIdent.GoName, " ===== Oneof option ====")
		g.P(fmt.Sprintf("type %s struct {", structName))
		g.P(fmt.Sprintf("  value %s", valueType))
		g.P("}")
		g.P()
		g.P(fmt.Sprintf("func (r *%s) Get%s() %s_En%sID {", structName, fullFieldName, msg.GoIdent.GoName, oneof.GoName))
		g.P(fmt.Sprintf("  return %s_En%sID_%s", msg.GoIdent.GoName, oneof.GoName, field.GoName))
		g.P("}")
		g.P()
		g.P(fmt.Sprintf("func (r *%s) GetReflectType%s() reflect.Type {", structName, fullFieldName))
		g.P(fmt.Sprintf("  return GetReflectType%s()", field.GoIdent.GoName))
		g.P("}")
		g.P()
		g.P(fmt.Sprintf("func (r *%s) Get%s() %s {", structName, field.GoName, valueType))
		g.P("  if r == nil {")
		g.P(fmt.Sprintf("    var zero %s", valueType))
		g.P("    return zero")
		g.P("  }")
		if field.Desc.Kind() == protoreflect.BytesKind {
			g.P("  if len(r.value) == 0 {")
			g.P("    return r.value")
			g.P("  }")
			g.P("  dup := make([]byte, len(r.value))")
			g.P("  copy(dup, r.value)")
			g.P("  return dup")
		} else {
			g.P("  return r.value")
		}
		g.P("}")
		g.P()
	}
}
