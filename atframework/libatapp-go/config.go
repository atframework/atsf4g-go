package libatapp

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	atframe_protocol "github.com/atframework/libatapp-go/protocol/atframe"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

// skipSpace 跳过字符串中的空白字符
func skipSpace(str string) string {
	return strings.TrimSpace(str)
}

// pickNumber 解析数字并处理负数和十六进制、八进制等
func pickNumber(str string, ignoreNegative bool) (int64, string, error) {
	// 处理负号
	negative := false
	if len(str) > 0 && str[0] == '-' {
		negative = true
		str = str[1:]
	}

	var val int64
	index := 0

	for ; index < len(str); index++ {
		// 验证字符是否为数字
		if !unicode.IsDigit(rune(str[index])) {
			break
		}
		// 将字符转换为数字，并构建整数值
		val = val*10 + int64(str[index]-'0')
	}
	str = str[index:]

	if negative && !ignoreNegative {
		val = -val
	}
	return val, str, nil
}

// pickDuration 解析字符串并填充到 protobuf 的 Duration 结构
func pickDuration(value string) (*durationpb.Duration, error) {
	// 去除空格
	value = skipSpace(value)

	// 解析数字
	var tmVal int64
	tmVal, value, err := pickNumber(value, false)
	if err != nil {
		return nil, err
	}

	// 去除空格
	value = skipSpace(value)

	// 解析单位
	unit := strings.ToLower(value)

	duration := durationpb.Duration{}

	switch {
	case unit == "s" || unit == "sec" || unit == "second" || unit == "seconds":
		duration.Seconds = tmVal
	case unit == "ms" || unit == "millisecond" || unit == "milliseconds":
		duration.Seconds = tmVal / 1000
		duration.Nanos = int32((tmVal % 1000) * 1000000)
	case unit == "us" || unit == "microsecond" || unit == "microseconds":
		duration.Seconds = tmVal / 1000000
		duration.Nanos = int32((tmVal % 1000000) * 1000)
	case unit == "ns" || unit == "nanosecond" || unit == "nanoseconds":
		duration.Seconds = tmVal / 1000000000
		duration.Nanos = int32(tmVal % 1000000000)
	case unit == "m" || unit == "minute" || unit == "minutes":
		duration.Seconds = tmVal * 60
	case unit == "h" || unit == "hour" || unit == "hours":
		duration.Seconds = tmVal * 3600
		duration.Nanos = int32(tmVal % 1000000000)
	case unit == "d" || unit == "day" || unit == "days":
		duration.Seconds = tmVal * 3600 * 24
	case unit == "w" || unit == "week" || unit == "weeks":
		duration.Seconds = tmVal * 3600 * 24 * 7
	default:
		return nil, fmt.Errorf("pickDuration unsupported unit: %s", unit)
	}

	return &duration, nil
}

// pickDuration 解析字符串并填充到 protobuf 的 Duration 结构
func pickSize(value string) (uint64, error) {
	// 去除空格
	value = skipSpace(value)

	// 解析数字
	var baseVal int64
	baseVal, value, err := pickNumber(value, true)
	if err != nil {
		return 0, err
	}

	// 去除空格
	value = skipSpace(value)

	// 解析单位
	unit := strings.ToLower(value)

	var val uint64

	switch {
	case unit == "b":
		val = uint64(baseVal)
	case unit == "kb":
		val = uint64(baseVal) * 1024
	case unit == "mb":
		val = uint64(baseVal) * 1024 * 1024
	case unit == "gb":
		val = uint64(baseVal) * 1024 * 1024 * 1024
	case unit == "pb":
		val = uint64(baseVal) * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("pickSize unsupported unit: %s", unit)
	}

	return val, nil
}

// pickTimestamp 解析时间字符串并填充到 protobuf 的 Timestamp 结构
func pickTimestamp(value string) (*timestamppb.Timestamp, error) {
	// 去除空格
	value = skipSpace(value)

	// 解析日期时间
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	// 转换为 protobuf Timestamp
	return timestamppb.New(t), nil
}

func parseDefaultToYamlData(defaultValue string, fd protoreflect.FieldDescriptor, sizeMode bool, logger *slog.Logger) (interface{}, error) {
	if sizeMode {
		return defaultValue, nil
	}
	switch fd.Kind() {
	case protoreflect.MessageKind:
		return defaultValue, nil
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(defaultValue)
		if err == nil {
			return v, nil
		}
		return nil, fmt.Errorf("expected bool, got %s err %s", defaultValue, err)
	case protoreflect.Int32Kind:
		v, err := strconv.ParseInt(defaultValue, 10, 32)
		if err == nil {
			return int32(v), nil
		}
		return nil, fmt.Errorf("expected int32, got %s err %s", defaultValue, err)
	case protoreflect.Sint32Kind:
		v, err := strconv.ParseInt(defaultValue, 10, 32)
		if err == nil {
			return int32(v), nil
		}
		return nil, fmt.Errorf("expected int32, got %s err %s", defaultValue, err)
	case protoreflect.Int64Kind:
		v, err := strconv.ParseInt(defaultValue, 10, 64)
		if err == nil {
			return int64(v), nil
		}
		return nil, fmt.Errorf("expected int64, got %s err %s", defaultValue, err)
	case protoreflect.Sint64Kind:
		v, err := strconv.ParseInt(defaultValue, 10, 64)
		if err == nil {
			return int64(v), nil
		}
		return nil, fmt.Errorf("expected int64, got %s err %s", defaultValue, err)
	case protoreflect.Uint32Kind:
		v, err := strconv.ParseUint(defaultValue, 10, 32)
		if err == nil {
			return uint32(v), nil
		}
		return nil, fmt.Errorf("expected uint32, got %s err %s", defaultValue, err)
	case protoreflect.Uint64Kind:
		v, err := strconv.ParseUint(defaultValue, 10, 64)
		if err == nil {
			return uint64(v), nil
		}
		return nil, fmt.Errorf("expected uint64, got %s err %s", defaultValue, err)
	case protoreflect.StringKind:
		return defaultValue, nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(defaultValue, 32)
		if err == nil {
			return float32(v), nil
		}
		return nil, fmt.Errorf("expected float32, got %s err %s", defaultValue, err)
	}
	return nil, fmt.Errorf("parseDefaultToYamlData unsupported field type: %v", fd.Kind())
}

// 从一个Field内读出数据 非Message 且为最底层 嵌套终点
func parseField(yamlData interface{}, fd protoreflect.FieldDescriptor, logger *slog.Logger) (protoreflect.Value, error) {
	// 首先需要预处理 yamlData
	if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
		if confMeta.DefaultValue != "" && yamlData == nil {
			logger.Info("Use Default", slog.Any("Field Name", fd.FullName()))
			var err error
			yamlData, err = parseDefaultToYamlData(confMeta.DefaultValue, fd, confMeta.SizeMode, logger)
			if err != nil {
				return protoreflect.Value{}, err
			}
		}

		if confMeta.SizeMode {
			// 需要从String 转为 Int
			v, ok := yamlData.(string)
			if !ok {
				return protoreflect.Value{}, fmt.Errorf("SizeMode true expected String, got %T", yamlData)
			}
			size, err := pickSize(v)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfUint64(size), nil
		}
	}

	if yamlData == nil {
		return protoreflect.Value{}, nil
	}

	if fd.Kind() == protoreflect.MessageKind {
		if fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) {
			v, ok := yamlData.(string)
			if !ok {
				return protoreflect.Value{}, fmt.Errorf("duration expected string, got %T", yamlData)
			}
			duration, err := pickDuration(v)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfMessage(duration.ProtoReflect()), nil
		}
		if fd.Message().FullName() == proto.MessageName(&timestamppb.Timestamp{}) {
			v, ok := yamlData.(string)
			if !ok {
				return protoreflect.Value{}, fmt.Errorf("timestamp expected string, got %T", yamlData)
			}
			timestamp, err := pickTimestamp(v)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfMessage(timestamp.ProtoReflect()), nil
		}
		return protoreflect.Value{}, fmt.Errorf("%s expected Duration or Timestamp, got %T", fd.FullName(), yamlData)
	}

	switch fd.Kind() {
	case protoreflect.BoolKind:
		if v, ok := yamlData.(bool); ok {
			return protoreflect.ValueOfBool(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected bool, got %T", yamlData)

	case protoreflect.Int32Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfInt32(int32(v)), nil
		}
		if v, ok := yamlData.(int32); ok {
			return protoreflect.ValueOfInt32(int32(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected int32, got %T", yamlData)
	case protoreflect.Sint32Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfInt32(int32(v)), nil
		}
		if v, ok := yamlData.(int32); ok {
			return protoreflect.ValueOfInt32(int32(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected int32, got %T", yamlData)
	case protoreflect.Int64Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		if v, ok := yamlData.(int32); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		if v, ok := yamlData.(int64); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected Int64, got %T", yamlData)
	case protoreflect.Sint64Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		if v, ok := yamlData.(int32); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		if v, ok := yamlData.(int64); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected Int64, got %T", yamlData)
	case protoreflect.Uint32Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfUint32(uint32(v)), nil
		}
		if v, ok := yamlData.(uint32); ok {
			return protoreflect.ValueOfUint32(uint32(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected uint32, got %T", yamlData)
	case protoreflect.Uint64Kind:
		if v, ok := yamlData.(int); ok {
			return protoreflect.ValueOfUint64(uint64(v)), nil
		}
		if v, ok := yamlData.(uint64); ok {
			return protoreflect.ValueOfUint64(uint64(v)), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected uint64, got %T", yamlData)
	case protoreflect.StringKind:
		if v, ok := yamlData.(string); ok {
			return protoreflect.ValueOfString(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected string, got %T", yamlData)

	case protoreflect.FloatKind:
		if v, ok := yamlData.(float32); ok {
			return protoreflect.ValueOfFloat32(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected float32, got %T", yamlData)
	}

	return protoreflect.Value{}, fmt.Errorf("unsupported field type: %v", fd.Kind())
}

func parseMessage(yamlData map[string]interface{}, msg proto.Message, logger *slog.Logger) error {
	len := msg.ProtoReflect().Descriptor().Fields().Len()
	for i := 0; i < len; i++ {
		fd := msg.ProtoReflect().Descriptor().Fields().Get(i)
		fieldName := fd.TextName()
		field_match := false

		if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
			if confMeta.FieldMatch != nil && confMeta.FieldMatch.FieldName != "" && confMeta.FieldMatch.FieldValue != "" {
				// 存在跳过规则
				if value, ok := yamlData[confMeta.FieldMatch.FieldName].(string); ok {
					// 存在
					if value == confMeta.FieldMatch.FieldValue {
						field_match = true
					} else {
						continue
					}
				}
			}
		}

		if fd.IsMap() {
			if yamlData == nil {
				continue
			}
			innerMap, ok := yamlData[fieldName].(map[string]interface{})
			if ok {
				// 这边需要循环Value
				for k, v := range innerMap {
					keyValue, err := parseField(k, fd.MapKey(), logger)
					if err != nil {
						return err
					}
					valueValue, err := parseField(v, fd.MapValue(), logger)
					if err != nil {
						return err
					}

					if keyValue.IsValid() && valueValue.IsValid() {
						msg.ProtoReflect().Mutable(fd).Map().Set(keyValue.MapKey(), valueValue)
					}
				}
			}
			continue
		}

		if fd.IsList() {
			if yamlData == nil {
				continue
			}
			innerList, ok := yamlData[fieldName].([]interface{})
			if ok && innerList != nil {
				for _, item := range innerList {
					if fd.Kind() == protoreflect.MessageKind {
						if fd.Message().FullName() != proto.MessageName(&durationpb.Duration{}) &&
							fd.Message().FullName() != proto.MessageName(&timestamppb.Timestamp{}) {
							// Message
							innerMap, ok := item.(map[string]interface{})
							if ok {
								if err := parseMessage(innerMap, msg.ProtoReflect().Mutable(fd).List().AppendMutable().Message().Interface(), logger); err != nil {
									return err
								}
							}
							continue
						}
					}
					// 非Message
					value, err := parseField(item, fd, logger)
					if err != nil {
						return err
					}
					if value.IsValid() {
						msg.ProtoReflect().Mutable(fd).List().Append(value)
					}
				}
			}
			continue
		}

		if fd.Kind() == protoreflect.MessageKind {
			if fd.Message().FullName() != proto.MessageName(&durationpb.Duration{}) &&
				fd.Message().FullName() != proto.MessageName(&timestamppb.Timestamp{}) {
				// 需要继续解析的字段
				if yamlData == nil {
					if err := parseMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
					continue
				}
				if field_match {
					// 在同层查找
					if err := parseMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
				} else {
				innerMap, ok := yamlData[fieldName].(map[string]interface{})
				if ok {
						if err := parseMessage(innerMap, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
				} else {
						if err := parseMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
						}
					}
				}
				continue
			}
		}

		var fieldData interface{}
		if yamlData != nil {
			fieldData = yamlData[fieldName]
		}
		value, err := parseField(fieldData, fd, logger)
		if err != nil {
			return err
		}
		if value.IsValid() {
			msg.ProtoReflect().Set(fd, value)
		}
	}

	return nil
}

func LoadConfigFromYaml(configPath string, firstPath string, configPb proto.Message, logger *slog.Logger) (err error) {
	var data []byte
	data, err = os.ReadFile(configPath)
	if err != nil {
		return
	}

	yamlData := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, yamlData)
	if err != nil {
		return
	}

	atappData := yamlData[firstPath].(map[string]interface{})
	err = parseMessage(atappData, configPb, logger)
	return
}
