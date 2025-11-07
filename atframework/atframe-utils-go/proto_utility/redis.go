package libatframe_utils_proto_utility

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func PBMapToRedis(msg proto.Message) map[string]interface{} {
	m := msg.ProtoReflect().Descriptor()
	ret := make(map[string]interface{}, m.Fields().Len())
	for i := 0; i < m.Fields().Len(); i++ {
		fd := m.Fields().Get(i)
		v := msg.ProtoReflect().Get(fd)

		name := string(fd.TextName())
		if fd.IsList() || fd.IsMap() {
			continue
		}
		switch fd.Kind() {
		case protoreflect.StringKind:
			ret[name] = "&" + v.String()
		case protoreflect.BytesKind:
			ret[name] = append([]byte("&"), v.Bytes()...)
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind:
			ret[name] = fmt.Sprintf("&%d", v.Int())
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			ret[name] = fmt.Sprintf("&%d", v.Uint())
		case protoreflect.BoolKind:
			ret[name] = fmt.Sprintf("&%t", v.Bool())
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			ret[name] = fmt.Sprintf("&%f", v.Float())
		case protoreflect.MessageKind:
			b, err := proto.Marshal(v.Message().Interface())
			if err != nil {
				continue
			}
			ret[name] = append([]byte("&"), b...)
		default:
			continue
		}
		continue
	}
	return ret
}

func RedisMapToPB(data map[string]string, msg proto.Message) error {
	m := msg.ProtoReflect()

	for key, val := range data {
		fd := m.Descriptor().Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			return fmt.Errorf("field not found:%s", key)
		}
		if val == "" || len(val) <= 1 {
			continue
		}
		val = val[1:]

		switch fd.Kind() {
		case protoreflect.StringKind:
			m.Set(fd, protoreflect.ValueOfString(val))
		case protoreflect.BytesKind:
			m.Set(fd, protoreflect.ValueOfBytes([]byte(val)))
		case protoreflect.Int32Kind, protoreflect.Sint32Kind:
			i, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfInt32(int32(i)))
		case protoreflect.Uint32Kind:
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfUint32(uint32(i)))
		case protoreflect.Int64Kind, protoreflect.Sint64Kind:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfInt64(i))
		case protoreflect.Uint64Kind:
			i, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfUint64(uint64(i)))
		case protoreflect.BoolKind:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfBool(b))
		case protoreflect.FloatKind:
			f, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfFloat32(float32(f)))
		case protoreflect.DoubleKind:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfFloat64(f))
		case protoreflect.MessageKind:
			// 嵌套结构从bytes反序列化
			subMsg := m.NewField(fd)
			sm := subMsg.Message()
			err := proto.Unmarshal([]byte(val), sm.Interface())
			if err != nil {
				return fmt.Errorf("unmarshal failed")
			}
			m.Set(fd, subMsg)
		default:
			return fmt.Errorf("not support type: %d", fd.Kind())
		}
	}
	return nil
}

func RedisSliceMapToPB(field []string, data []interface{}, msg proto.Message) error {
	if len(field) != len(data) {
		return fmt.Errorf("redis req rsp len not match")
	}

	m := msg.ProtoReflect()

	for i := range data {
		key := field[i]
		val, ok := data[i].(string)
		if !ok {
			return fmt.Errorf("data not string %s", data[i])
		}

		fd := m.Descriptor().Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			return fmt.Errorf("field not found:%s", key)
		}
		if val == "" {
			continue
		}

		switch fd.Kind() {
		case protoreflect.StringKind:
			m.Set(fd, protoreflect.ValueOfString(val))
		case protoreflect.BytesKind:
			m.Set(fd, protoreflect.ValueOfBytes([]byte(val)))
		case protoreflect.Int32Kind, protoreflect.Sint32Kind:
			i, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfInt32(int32(i)))
		case protoreflect.Uint32Kind:
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfUint32(uint32(i)))
		case protoreflect.Int64Kind, protoreflect.Sint64Kind:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfInt64(i))
		case protoreflect.Uint64Kind:
			i, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfUint64(uint64(i)))
		case protoreflect.BoolKind:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfBool(b))
		case protoreflect.FloatKind:
			f, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfFloat32(float32(f)))
		case protoreflect.DoubleKind:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			m.Set(fd, protoreflect.ValueOfFloat64(f))
		case protoreflect.MessageKind:
			// 嵌套结构从bytes反序列化
			subMsg := m.NewField(fd)
			sm := subMsg.Message()
			err := proto.Unmarshal([]byte(val), sm.Interface())
			if err != nil {
				return fmt.Errorf("unmarshal failed")
			}
			m.Set(fd, subMsg)
		default:
			return fmt.Errorf("not support type: %d", fd.Kind())
		}
	}
	return nil
}
