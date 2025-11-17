package libatframe_utils_proto_utility

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

var CASKeyField = "CAS_VERSION"

func PBMapToRedis(msg proto.Message, CASVersion *uint64) []string {
	m := msg.ProtoReflect().Descriptor()
	var ret []string
	if CASVersion != nil {
		ret = make([]string, 0, m.Fields().Len()*2+2)
		ret = append(ret, CASKeyField, fmt.Sprintf("%d", *CASVersion))
	} else {
		ret = make([]string, 0, m.Fields().Len()*2)
	}

	for i := 0; i < m.Fields().Len(); i++ {
		fd := m.Fields().Get(i)
		v := msg.ProtoReflect().Get(fd)

		name := string(fd.TextName())
		if fd.IsList() || fd.IsMap() {
			continue
		}
		switch fd.Kind() {
		case protoreflect.StringKind:
			ret = append(ret, name, "&"+v.String())
		case protoreflect.BytesKind:
			ret = append(ret, name, lu.BytestoString(append([]byte("&"), v.Bytes()...)))
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind:
			ret = append(ret, name, fmt.Sprintf("&%d", v.Int()))
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			ret = append(ret, name, fmt.Sprintf("&%d", v.Uint()))
		case protoreflect.BoolKind:
			ret = append(ret, name, fmt.Sprintf("&%t", v.Bool()))
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			ret = append(ret, name, fmt.Sprintf("&%f", v.Float()))
		case protoreflect.MessageKind:
			b, err := proto.MarshalOptions{}.MarshalAppend([]byte("&"), v.Message().Interface())
			if err != nil {
				continue
			}
			ret = append(ret, name, lu.BytestoString(b))
		default:
			continue
		}
		continue
	}
	return ret
}

func RedisMapToPB(data map[string]string, msg proto.Message) (uint64, error) {
	m := msg.ProtoReflect()
	var casVersion uint64 = 0

	for key, val := range data {
		fd := m.Descriptor().Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			if key == CASKeyField {
				version, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("parse cas version failed:%s", val)
				}
				casVersion = version
				continue
			}
			return 0, fmt.Errorf("field not found:%s", key)
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
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfInt32(int32(i)))
		case protoreflect.Uint32Kind:
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfUint32(uint32(i)))
		case protoreflect.Int64Kind, protoreflect.Sint64Kind:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfInt64(i))
		case protoreflect.Uint64Kind:
			i, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfUint64(uint64(i)))
		case protoreflect.BoolKind:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfBool(b))
		case protoreflect.FloatKind:
			f, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfFloat32(float32(f)))
		case protoreflect.DoubleKind:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfFloat64(f))
		case protoreflect.MessageKind:
			// 嵌套结构从bytes反序列化
			subMsg := m.NewField(fd)
			err := proto.Unmarshal(lu.StringtoBytes(val), subMsg.Message().Interface())
			if err != nil {
				return 0, fmt.Errorf("unmarshal failed")
			}
			m.Set(fd, subMsg)
		default:
			return 0, fmt.Errorf("not support type: %d", fd.Kind())
		}
	}
	return casVersion, nil
}

func RedisSliceMapToPB(field []string, data []interface{}, msg proto.Message) (uint64, error) {
	if len(field) != len(data) {
		return 0, fmt.Errorf("redis req rsp len not match")
	}

	m := msg.ProtoReflect()
	var casVersion uint64 = 0

	for i := range data {
		key := field[i]
		val, ok := data[i].(string)
		if !ok {
			return 0, fmt.Errorf("data not string %s", data[i])
		}

		fd := m.Descriptor().Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			if key == CASKeyField {
				version, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("parse cas version failed:%s", val)
				}
				casVersion = version
				continue
			}
			return 0, fmt.Errorf("field not found:%s", key)
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
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfInt32(int32(i)))
		case protoreflect.Uint32Kind:
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfUint32(uint32(i)))
		case protoreflect.Int64Kind, protoreflect.Sint64Kind:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfInt64(i))
		case protoreflect.Uint64Kind:
			i, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfUint64(uint64(i)))
		case protoreflect.BoolKind:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfBool(b))
		case protoreflect.FloatKind:
			f, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfFloat32(float32(f)))
		case protoreflect.DoubleKind:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return 0, err
			}
			m.Set(fd, protoreflect.ValueOfFloat64(f))
		case protoreflect.MessageKind:
			// 嵌套结构从bytes反序列化
			subMsg := m.NewField(fd)
			sm := subMsg.Message()
			err := proto.Unmarshal(lu.StringtoBytes(val), sm.Interface())
			if err != nil {
				return 0, fmt.Errorf("unmarshal failed")
			}
			m.Set(fd, subMsg)
		default:
			return 0, fmt.Errorf("not support type: %d", fd.Kind())
		}
	}
	return casVersion, nil
}
