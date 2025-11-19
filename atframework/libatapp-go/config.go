package libatapp

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
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
	orginValue := value
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
	case unit == "" && tmVal == 0:
		duration.Seconds = 0
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
		return nil, fmt.Errorf("pickDuration unsupported orginValue: %s", orginValue)
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
	case unit == "b" || unit == "":
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
		t, err = time.Parse(time.DateTime, value)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05Z07:00", value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse timestamp: %v", err)
			}
		}
	}

	// 转换为 protobuf Timestamp
	return timestamppb.New(t), nil
}

// 定义字符映射常量
const (
	SPLITCHAR = 1 << iota
	STRINGSYM
	TRANSLATE
	CMDSPLIT
)

// 字符映射数组，最大256个字符
var (
	mapValue   [256]int
	transValue [256]rune
)

func initCharSet() {
	// 如果已初始化则跳过
	if mapValue[' ']&SPLITCHAR != 0 {
		return
	}

	// 设置字符集
	mapValue[' '] = SPLITCHAR
	mapValue['\t'] = SPLITCHAR
	mapValue['\r'] = SPLITCHAR
	mapValue['\n'] = SPLITCHAR

	// 设置字符串开闭符
	mapValue['\''] = STRINGSYM
	mapValue['"'] = STRINGSYM

	// 设置转义字符
	mapValue['\\'] = TRANSLATE

	// 设置命令分隔符
	mapValue[' '] |= CMDSPLIT
	mapValue[','] = CMDSPLIT
	mapValue[';'] = CMDSPLIT

	// 初始化转义字符
	for i := 0; i < 256; i++ {
		transValue[i] = rune(i)
	}

	// 常见转义字符设置
	transValue['0'] = '\x00'
	transValue['a'] = '\a'
	transValue['b'] = '\b'
	transValue['f'] = '\f'
	transValue['r'] = '\r'
	transValue['n'] = '\n'
	transValue['t'] = '\t'
	transValue['v'] = '\v'
	transValue['\\'] = '\\'
	transValue['\''] = '\''
	transValue['"'] = '"'
}

// getSegment 函数：解析字符串并返回下一个段落
func getSegment(beginStr string) (string, string) {
	initCharSet()

	var val strings.Builder
	var flag rune

	// 去除分隔符前缀
	beginStr = strings.TrimLeftFunc(beginStr, func(r rune) bool {
		return (mapValue[r]&SPLITCHAR != 0)
	})

	i := 0
	for i < len(beginStr) {
		ch := rune(beginStr[i])

		if mapValue[ch]&SPLITCHAR != 0 {
			break
		}

		if mapValue[ch]&STRINGSYM != 0 {
			flag = ch
			i++

			// 处理转义字符
			for i < len(beginStr) {
				ch = rune(beginStr[i])
				if ch == flag {
					break
				}
				if mapValue[ch]&TRANSLATE != 0 && i+1 < len(beginStr) {
					i++
					ch = transValue[rune(beginStr[i])]
				}
				val.WriteRune(ch)
				i++
			}
			i++ // 跳过结束的 flag 字符
			break
		} else {
			val.WriteRune(ch)
			i++
		}
	}

	i = max(i, len(val.String()))
	// 去除分隔符后缀
	beginStr = strings.TrimLeftFunc(beginStr[i:], func(r rune) bool {
		return (mapValue[r]&SPLITCHAR != 0)
	})

	return val.String(), beginStr
}

func splitStringToArray(start string) (result []string) {
	result = make([]string, 0)
	for len(start) > 0 {
		splitedVal, next := getSegment(start)
		splitedVal = strings.TrimSpace(splitedVal)
		if splitedVal == "" {
			start = next
			continue
		}
		result = append(result, splitedVal)
		start = next
	}
	return
}

func parseStringToYamlData(defaultValue string, fd protoreflect.FieldDescriptor, sizeMode bool, logger *slog.Logger) (interface{}, error) {
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

func convertToInt64(data interface{}) (int64, error) {
	switch reflect.ValueOf(data).Kind() {
	case reflect.Int:
		return int64(reflect.ValueOf(data).Int()), nil // 转换为 int64
	case reflect.Int32:
		return int64(reflect.ValueOf(data).Int()), nil // 转换为 int64
	case reflect.Int64:
		return int64(reflect.ValueOf(data).Int()), nil // 转换为 int64
	case reflect.Uint:
		return int64(reflect.ValueOf(data).Uint()), nil // 转换为 int64
	case reflect.Uint32:
		return int64(reflect.ValueOf(data).Uint()), nil // 转换为 int64
	case reflect.Uint64:
		return int64(reflect.ValueOf(data).Uint()), nil // 转换为 int64'
	case reflect.Bool:
		if reflect.ValueOf(data).Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case reflect.Float32:
		return int64(reflect.ValueOf(data).Float()), nil
	case reflect.Float64:
		return int64(reflect.ValueOf(data).Float()), nil
	case reflect.String:
		value, _, err := pickNumber(reflect.ValueOf(data).String(), false)
		if err != nil {
			return 0, fmt.Errorf("convertToInt64 failed pickNumber failed %v, error: %s", data, err)
		}
		return value, nil
	}
	if v, ok := data.(*timestamppb.Timestamp); ok {
		return v.Seconds*1000000000 + int64(v.Nanos), nil
	}
	if v, ok := data.(*durationpb.Duration); ok {
		return v.Seconds*1000000000 + int64(v.Nanos), nil
	}
	return 0, fmt.Errorf("convertToInt64 failed Type not found: %T", data)
}

func checkMinMax(yamlData interface{}, minData interface{}, maxData interface{}) (interface{}, error) {
	yamlDataNative := yamlData
	minDataNative := minData
	maxDataNative := maxData

	returnNative := yamlDataNative

	var err error
	if yamlData != nil {
		yamlData, err = convertToInt64(yamlData)
		if err != nil {
			return nil, err
		}
	}
	if minData != nil {
		minData, err = convertToInt64(minData)
		if err != nil {
			return nil, err
		}
	}
	if maxData != nil {
		maxData, err = convertToInt64(maxData)
		if err != nil {
			return nil, err
		}
	}

	// 选出最终值
	if yamlData == nil {
		yamlData = minData
		returnNative = minDataNative
	} else if minData != nil {
		// 对比
		yamlDataV, ok := yamlData.(int64)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("convertField Check yamlData expected Int64, got %T", yamlData)
		}
		minDataV, ok := minData.(int64)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("convertField Check MinValue expected Int64, got %T", minData)
		}

		if minDataV > yamlDataV {
			returnNative = minDataNative
		}
	}

	if yamlData == nil {
		yamlData = maxData
		returnNative = maxDataNative
	} else if maxData != nil {
		// 对比
		yamlDataV, ok := yamlData.(int64)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("convertField Check yamlData expected Int64, got %T", yamlData)
		}
		maxDataV, ok := maxData.(int64)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("convertField Check maxDataV expected Int64, got %T", maxData)
		}

		if yamlDataV > maxDataV {
			returnNative = maxDataNative
		}
	}

	return returnNative, nil
}

func convertField(yamlData interface{}, minData interface{}, maxData interface{}, fd protoreflect.FieldDescriptor, logger *slog.Logger) (protoreflect.Value, error) {
	if yamlData == nil && minData == nil && maxData == nil {
		return protoreflect.Value{}, nil
	}

	yamlData, err := checkMinMax(yamlData, minData, maxData)
	if err != nil {
		return protoreflect.Value{}, err
	}

	// 更新最终值
	switch fd.Kind() {
	case protoreflect.BoolKind:
		if v, ok := yamlData.(bool); ok {
			return protoreflect.ValueOfBool(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected bool, got %T", yamlData)
	case protoreflect.Int32Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Sint32Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Int64Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(int64(v)), nil
	case protoreflect.Sint64Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(int64(v)), nil
	case protoreflect.Uint32Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Uint64Kind:
		v, err := convertToInt64(yamlData)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(uint64(v)), nil
	case protoreflect.StringKind:
		if v, ok := yamlData.(string); ok {
			return protoreflect.ValueOfString(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected string, got %T", yamlData)

	case protoreflect.FloatKind:
		if v, ok := yamlData.(float32); ok {
			return protoreflect.ValueOfFloat32(v), nil
		}
		if v, ok := yamlData.(float64); ok {
			return protoreflect.ValueOfFloat64(v), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected float32, got %T", yamlData)

	case protoreflect.MessageKind:
		if v, ok := yamlData.(*timestamppb.Timestamp); ok {
			return protoreflect.ValueOfMessage(v.ProtoReflect()), nil
		}
		if v, ok := yamlData.(*durationpb.Duration); ok {
			return protoreflect.ValueOfMessage(v.ProtoReflect()), nil
		}
		return protoreflect.Value{}, fmt.Errorf("expected Timestamp or Duration, got %T", yamlData)
	}

	return protoreflect.Value{}, fmt.Errorf("unsupported field type: %v", fd.Kind())
}

// 从一个Field内读出数据 非Message 且为最底层 嵌套终点
func parseField(yamlData interface{}, fd protoreflect.FieldDescriptor, logger *slog.Logger) (protoreflect.Value, error) {
	// 获取最大最小值
	var minValue interface{}
	var maxValue interface{}

	if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
		// 首先覆盖默认值
		if confMeta.DefaultValue != "" && yamlData == nil {
			var err error
			yamlData, err = parseStringToYamlData(confMeta.DefaultValue, fd, confMeta.SizeMode, logger)
			if err != nil {
				return protoreflect.Value{}, err
			}
		}

		// 取出极值
		if confMeta.MinValue != "" {
			var err error
			minValue, err = parseStringToYamlData(confMeta.MinValue, fd, confMeta.SizeMode, logger)
			if err != nil {
				return protoreflect.Value{}, err
			}
		}
		if confMeta.MaxValue != "" {
			var err error
			maxValue, err = parseStringToYamlData(confMeta.MaxValue, fd, confMeta.SizeMode, logger)
			if err != nil {
				return protoreflect.Value{}, err
			}
		}

		// 转换值
		if confMeta.SizeMode {
			// 需要从String 转为 Int
			if yamlData != nil {
				// 基础
				v, ok := yamlData.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("SizeMode true expected String, got %T", yamlData)
				}
				size, err := pickSize(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				yamlData = size
			}
			if minValue != nil {
				// 最小
				v, ok := minValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("SizeMode true expected String, got %T", minValue)
				}
				size, err := pickSize(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				minValue = size
			}
			if maxValue != nil {
				// 最大
				v, ok := maxValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("SizeMode true expected String, got %T", maxValue)
				}
				size, err := pickSize(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				maxValue = size
			}
		}
	}

	if fd.Kind() == protoreflect.MessageKind {
		// 转换值
		if fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) {
			if yamlData != nil {
				v, ok := yamlData.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("duration expected string, got %T", yamlData)
				}
				duration, err := pickDuration(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				yamlData = duration
			}
			if minValue != nil {
				v, ok := minValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("duration expected string, got %T", minValue)
				}
				duration, err := pickDuration(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				minValue = duration
			}
			if maxValue != nil {
				v, ok := maxValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("duration expected string, got %T", maxValue)
				}
				duration, err := pickDuration(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				maxValue = duration
			}
		} else if fd.Message().FullName() == proto.MessageName(&timestamppb.Timestamp{}) {
			if yamlData != nil {
				v, ok := yamlData.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("timestamp expected string, got %T", yamlData)
				}
				timestamp, err := pickTimestamp(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				yamlData = timestamp
			}
			if minValue != nil {
				v, ok := minValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("timestamp expected string, got %T", minValue)
				}
				timestamp, err := pickTimestamp(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				minValue = timestamp
			}
			if maxValue != nil {
				v, ok := maxValue.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("timestamp expected string, got %T", maxValue)
				}
				timestamp, err := pickTimestamp(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				maxValue = timestamp
			}
		} else {
			return protoreflect.Value{}, fmt.Errorf("%s expected Duration or Timestamp, got %T", fd.FullName(), yamlData)
		}
	}
	return convertField(yamlData, minValue, maxValue, fd, logger)
}

func ParseMessage(yamlData map[string]interface{}, msg proto.Message, logger *slog.Logger) error {
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
			innerData, ok := yamlData[fieldName]
			if !ok || innerData == nil {
				continue
			}
			innerList, ok := innerData.([]interface{})
			if !ok {
				// 可能是string的Array 切割
				innerString, ok := innerData.(string)
				if !ok {
					// 分割 innerString
					continue
				}
				stringSlice := splitStringToArray(innerString)
				for _, v := range stringSlice {
					innerList = append(innerList, v)
				}
			}

			for _, item := range innerList {
				if fd.Kind() == protoreflect.MessageKind {
					if fd.Message().FullName() != proto.MessageName(&durationpb.Duration{}) &&
						fd.Message().FullName() != proto.MessageName(&timestamppb.Timestamp{}) {
						// Message
						innerMap, ok := item.(map[string]interface{})
						if ok {
							if err := ParseMessage(innerMap, msg.ProtoReflect().Mutable(fd).List().AppendMutable().Message().Interface(), logger); err != nil {
								return err
							}
						} else {
							innerString, ok := item.(string)
							if ok {
								// 需要String视为Yaml解析
								yamlData = make(map[string]interface{})
								err := yaml.Unmarshal(lu.StringtoBytes(innerString), yamlData)
								if err != nil {
									return err
								}
								if err = ParseMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
									return err
								}
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
			continue
		}

		if fd.Kind() == protoreflect.MessageKind {
			if fd.Message().FullName() != proto.MessageName(&durationpb.Duration{}) &&
				fd.Message().FullName() != proto.MessageName(&timestamppb.Timestamp{}) {
				// 需要继续解析的字段
				if yamlData == nil {
					if err := ParseMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
					continue
				}
				if field_match {
					// 在同层查找
					if err := ParseMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
				} else {
					innerMap, ok := yamlData[fieldName].(map[string]interface{})
					if ok {
						if err := ParseMessage(innerMap, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
							return err
						}
					} else {
						innerString, ok := yamlData[fieldName].(string)
						if ok {
							// 需要String视为Yaml解析
							yamlData = make(map[string]interface{})
							err := yaml.Unmarshal(lu.StringtoBytes(innerString), yamlData)
							if err != nil {
								return err
							}
							if err = ParseMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
								return err
							}
						} else {
							logger.Warn("ParseMessage message field not found, use default", "field", fieldName)
							if err := ParseMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
								return err
							}
						}
					}
				}
				continue
			}
		}

		var fieldData interface{}
		if yamlData != nil {
			ok := false
			fieldData, ok = yamlData[fieldName]
			if !ok {
				logger.Warn("ParseMessage field not found, use default", "field", fieldName)
			}
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

func LoadConfigFromOriginData(originData interface{}, prefixPath string, configPb proto.Message, logger *slog.Logger) (err error) {
	parent := originData
	pathParts := strings.Split(prefixPath, ".")
	for i, pathPart := range pathParts {
		trimPart := strings.TrimSpace(pathPart)
		if trimPart == "" {
			continue
		}

		if lu.IsNil(parent) {
			err = fmt.Errorf("LoadConfigFromYaml data nil")
			break
		}

		arrayIndex, convErr := strconv.Atoi(trimPart)
		if convErr == nil {
			// 数组下标
			parentArray, ok := parent.([]interface{})
			if !ok {
				err = fmt.Errorf("LoadConfigFromYaml expected array at %s, got %T", strings.Join(pathParts[0:i+1], "."), reflect.TypeOf(parent).Elem().Name())
				break
			}
			if len(parentArray) <= arrayIndex {
				err = fmt.Errorf("LoadConfigFromYaml array index out of range at %s, got %d >= %d", strings.Join(pathParts[0:i+1], "."), arrayIndex, len(parentArray))
				break
			}
			parent = parentArray[arrayIndex]
		} else {
			// 字符串key
			parentMap, ok := parent.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("LoadConfigFromYaml expected map at %s, got %T", strings.Join(pathParts[0:i+1], "."), reflect.TypeOf(parent).Elem().Name())
				break
			}
			parent, ok = parentMap[trimPart]
			if !ok {
				err = fmt.Errorf("LoadConfigFromYaml key not found at %s", strings.Join(pathParts[0:i+1], "."))
				break
			}
		}
	}

	if err != nil {
		logger.Error("load prefixPath failed", "err", err)
		// 使用初始值初始化
		parseErr := ParseMessage(nil, configPb, logger)
		if parseErr != nil {
			logger.Error("ParseMessage failed", "err", parseErr)
		}
		return
	}

	atappData, ok := parent.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("LoadConfigFromYaml expected map at %s, got %T", strings.Join(pathParts, "."), reflect.TypeOf(parent).Elem().Name())
		return
	}
	err = ParseMessage(atappData, configPb, logger)
	return
}

func LoadConfigFromYaml(configPath string, prefixPath string, configPb proto.Message, logger *slog.Logger) (yamlData map[string]interface{}, err error) {
	var data []byte
	data, err = os.ReadFile(configPath)
	if err != nil {
		return
	}

	yamlData = make(map[string]interface{})
	err = yaml.Unmarshal(data, yamlData)
	if err != nil {
		return
	}

	err = LoadConfigFromOriginData(yamlData, prefixPath, configPb, logger)
	return
}
