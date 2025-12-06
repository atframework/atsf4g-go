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
	"google.golang.org/protobuf/types/dynamicpb"

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

func parseStringToYamlData(stringValue string, fd protoreflect.FieldDescriptor, sizeMode bool, logger *slog.Logger) (interface{}, error) {
	if sizeMode {
		return stringValue, nil
	}
	switch fd.Kind() {
	case protoreflect.MessageKind:
		return stringValue, nil
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(stringValue)
		if err == nil {
			return v, nil
		}
		return nil, fmt.Errorf("expected bool, got %s err %s", stringValue, err)
	case protoreflect.Int32Kind:
	case protoreflect.Sint32Kind:
		var v int64
		var err error
		if strings.HasPrefix(stringValue, "0x") || strings.HasPrefix(stringValue, "0X") {
			v, err = strconv.ParseInt(stringValue, 16, 32)
		} else if strings.HasPrefix(stringValue, "0o") || strings.HasPrefix(stringValue, "0O") {
			v, err = strconv.ParseInt(stringValue[2:], 8, 32)
		} else {
			v, err = strconv.ParseInt(stringValue, 10, 32)
		}
		if err == nil {
			return int32(v), nil
		}
		return nil, fmt.Errorf("expected int32, got %s err %s", stringValue, err)
	case protoreflect.Int64Kind:
	case protoreflect.Sint64Kind:
		var v int64
		var err error
		if strings.HasPrefix(stringValue, "0x") || strings.HasPrefix(stringValue, "0X") {
			v, err = strconv.ParseInt(stringValue, 16, 64)
		} else if strings.HasPrefix(stringValue, "0o") || strings.HasPrefix(stringValue, "0O") {
			v, err = strconv.ParseInt(stringValue[2:], 8, 64)
		} else {
			v, err = strconv.ParseInt(stringValue, 10, 64)
		}
		if err == nil {
			return int64(v), nil
		}
		return nil, fmt.Errorf("expected int64, got %s err %s", stringValue, err)
	case protoreflect.Uint32Kind:
		var v uint64
		var err error
		if strings.HasPrefix(stringValue, "0x") || strings.HasPrefix(stringValue, "0X") {
			v, err = strconv.ParseUint(stringValue, 16, 32)
		} else if strings.HasPrefix(stringValue, "0o") || strings.HasPrefix(stringValue, "0O") {
			v, err = strconv.ParseUint(stringValue[2:], 8, 32)
		} else {
			v, err = strconv.ParseUint(stringValue, 10, 32)
		}
		if err == nil {
			return uint32(v), nil
		}
		return nil, fmt.Errorf("expected uint32, got %s err %s", stringValue, err)
	case protoreflect.Uint64Kind:
		var v uint64
		var err error
		if strings.HasPrefix(stringValue, "0x") || strings.HasPrefix(stringValue, "0X") {
			v, err = strconv.ParseUint(stringValue, 16, 64)
		} else if strings.HasPrefix(stringValue, "0o") || strings.HasPrefix(stringValue, "0O") {
			v, err = strconv.ParseUint(stringValue[2:], 8, 64)
		} else {
			v, err = strconv.ParseUint(stringValue, 10, 64)
		}
		if err == nil {
			return uint64(v), nil
		}
		return nil, fmt.Errorf("expected uint64, got %s err %s", stringValue, err)
	case protoreflect.StringKind:
		return stringValue, nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(stringValue, 32)
		if err == nil {
			return float32(v), nil
		}
		return nil, fmt.Errorf("expected float32, got %s err %s", stringValue, err)
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(stringValue, 64)
		if err == nil {
			return float64(v), nil
		}
		return nil, fmt.Errorf("expected float64, got %s err %s", stringValue, err)
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
		if v, ok := yamlData.(string); ok {
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return protoreflect.ValueOfFloat64(f), nil
			}
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

func pickSizeMode(value interface{}) (uint64, error) {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := reflect.ValueOf(value).Int()
		return uint64(v), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := reflect.ValueOf(value).Uint()
		return uint64(v), nil
	case reflect.String:
		return pickSize(value.(string))
	default:
		return 0, fmt.Errorf("SizeMode true expected String or int, got %T", value)
	}
}

// 从一个Field内读出数据 非Message 且为最底层 嵌套终点
func parseField(inputData interface{}, fd protoreflect.FieldDescriptor, logger *slog.Logger) (protoreflect.Value, error) {
	// 获取最大最小值
	var minValue interface{}
	var maxValue interface{}

	if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
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
			if inputData != nil {
				// 基础
				var err error
				inputData, err = pickSizeMode(inputData)
				if err != nil {
					return protoreflect.Value{}, err
				}
			}
			if minValue != nil {
				// 最小
				var err error
				minValue, err = pickSizeMode(minValue)
				if err != nil {
					return protoreflect.Value{}, err
				}
			}
			if maxValue != nil {
				// 最大
				var err error
				maxValue, err = pickSizeMode(maxValue)
				if err != nil {
					return protoreflect.Value{}, err
				}
			}
		}
	}

	if fd.Kind() == protoreflect.MessageKind {
		// 转换值
		if fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) {
			if inputData != nil {
				v, ok := inputData.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("duration expected string, got %T", inputData)
				}
				duration, err := pickDuration(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				inputData = duration
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
			if inputData != nil {
				v, ok := inputData.(string)
				if !ok {
					return protoreflect.Value{}, fmt.Errorf("timestamp expected string, got %T", inputData)
				}
				timestamp, err := pickTimestamp(v)
				if err != nil {
					return protoreflect.Value{}, err
				}
				inputData = timestamp
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
			return protoreflect.Value{}, fmt.Errorf("%s expected Duration or Timestamp, got %T", fd.FullName(), inputData)
		}
	}

	return convertField(inputData, minValue, maxValue, fd, logger)
}

type ConfigExistedIndex struct {
	ExistedSet  map[string]struct{}
	MapKeyIndex map[string]int
}

func (i *ConfigExistedIndex) MutableExistedSet() map[string]struct{} {
	if i.ExistedSet == nil {
		i.ExistedSet = make(map[string]struct{})
	}

	return i.ExistedSet
}

func (i *ConfigExistedIndex) MutableMapKeyIndex() map[string]int {
	if i.MapKeyIndex == nil {
		i.MapKeyIndex = make(map[string]int)
	}

	return i.MapKeyIndex
}

func CreateConfigExistIndex() *ConfigExistedIndex {
	return &ConfigExistedIndex{
		ExistedSet:  make(map[string]struct{}),
		MapKeyIndex: make(map[string]int),
	}
}

func makeExistedMapKeyIndexKey(existedSetPrefix string, fd protoreflect.FieldDescriptor, mk protoreflect.MapKey) string {
	keyFd := fd.MapKey()

	switch keyFd.Kind() {
	case protoreflect.BoolKind:
		if mk.Bool() {
			return fmt.Sprintf("%s%s.1", existedSetPrefix, fd.Name())
		} else {
			return fmt.Sprintf("%s%s.0", existedSetPrefix, fd.Name())
		}
	case protoreflect.StringKind:
		return fmt.Sprintf("%s%s.%s", existedSetPrefix, fd.Name(), mk.String())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind:
		return fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), mk.Int())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind:
		return fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), mk.Int())
	case protoreflect.Uint32Kind:
		return fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), mk.Uint())
	case protoreflect.Uint64Kind:
		return fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), mk.Uint())
	}
	return fmt.Sprintf("%s%s.%s", existedSetPrefix, fd.FullName(), mk.String())
}

func ParsePlainMessage(yamlData map[string]interface{}, msg proto.Message, logger *slog.Logger) error {
	len := msg.ProtoReflect().Descriptor().Fields().Len()
	for i := 0; i < len; i++ {
		fd := msg.ProtoReflect().Descriptor().Fields().Get(i)
		fieldName := string(fd.Name())
		fieldMatch := false

		if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
			if confMeta.FieldMatch != nil && confMeta.FieldMatch.FieldName != "" && confMeta.FieldMatch.FieldValue != "" {
				// 存在跳过规则
				if value, ok := yamlData[confMeta.FieldMatch.FieldName].(string); ok {
					// 存在
					if value == confMeta.FieldMatch.FieldValue {
						fieldMatch = true
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
							if err := ParsePlainMessage(innerMap, msg.ProtoReflect().Mutable(fd).List().AppendMutable().Message().Interface(), logger); err != nil {
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
								if err = ParsePlainMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
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
					if err := ParsePlainMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
					continue
				}
				if fieldMatch {
					// 在同层查找
					if err := ParsePlainMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
						return err
					}
				} else {
					innerMap, ok := yamlData[fieldName].(map[string]interface{})
					if ok {
						if err := ParsePlainMessage(innerMap, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
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
							if err = ParsePlainMessage(yamlData, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
								return err
							}
						} else {
							logger.Warn("ParseMessage message field not found, use default", "field", fieldName)
							if err := ParsePlainMessage(nil, msg.ProtoReflect().Mutable(fd).Message().Interface(), logger); err != nil {
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
			return fmt.Errorf("parseField error fieldName %s err %v", fieldName, err)
		}
		if value.IsValid() {
			if !msg.ProtoReflect().Get(fd).IsValid() {
				msg.ProtoReflect().Set(fd, value)
			}
		}
	}

	return nil
}

func dumpYamlIntoMessageFieldValue(yamlData interface{}, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil || yamlData == nil {
		return false
	}

	var fieldExistedKey string
	if dumpExistedSet != nil {
		if fd.IsList() {
			fieldExistedKey = fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), dst.ProtoReflect().Get(fd).List().Len())
		} else {
			fieldExistedKey = fmt.Sprintf("%s%s", existedSetPrefix, fd.Name())
		}
	}

	fieldValue := protoreflect.Value{}
	var err error

	if fd.Kind() == protoreflect.MessageKind {
		if fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) ||
			fd.Message().FullName() == proto.MessageName(&timestamppb.Timestamp{}) {
			fieldValue, err = parseField(yamlData, fd, logger)
			if err != nil {
				return false
			}
		} else {
			// Message
			innerMap, ok := yamlData.(map[string]interface{})
			if !ok {
				return false
			}

			fieldMessage := dynamicpb.NewMessage(fd.Message())
			if fieldMessage == nil {
				return false
			}

			if !dumpYamlIntoMessage(innerMap, fieldMessage, logger, dumpExistedSet, existedSetPrefix+string(fd.Name())+".") {
				return false
			}

			fieldValue = protoreflect.ValueOfMessage(fieldMessage)
		}
	} else {
		fieldValue, err = parseField(yamlData, fd, logger)
		if err != nil {
			return false
		}
	}

	if !fieldValue.IsValid() {
		return false
	}

	if fd.IsList() {
		dst.ProtoReflect().Mutable(fd).List().Append(fieldValue)
	} else {
		dst.ProtoReflect().Set(fd, fieldValue)
	}

	if dumpExistedSet != nil {
		dumpExistedSet.MutableExistedSet()[fieldExistedKey] = struct{}{}
	}

	return true
}

func dumpYamlIntoMessageFieldItem(yamlData map[string]interface{}, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil || len(yamlData) == 0 {
		return false
	}

	fieldName := string(fd.Name())
	fieldData, ok := yamlData[fieldName]
	if !ok || fieldData == nil {
		return false
	}

	if fd.IsMap() {
		innerMap, ok := fieldData.(map[string]interface{})
		if !ok {
			return false
		}

		hasValue := false

		nextMapExistedKeyIndex := 0
		// 这边需要循环Value
		for k, v := range innerMap {
			mapExistedKeyIndex := nextMapExistedKeyIndex
			nextMapExistedKeyIndex++
			// 使用 dynamicpb 创建临时消息来收集环境变量数据
			tempMsg := dynamicpb.NewMessage(fd.Message())
			if tempMsg == nil {
				return false
			}

			// mapKeyExistedKey := makeExistedMapKeyIndexKey(existedSetPrefix, fd, protoreflect.ValueOfString(k).MapKey(), mapExistedKeyIndex)
			mapMessagePrefix := fmt.Sprintf("%s%s.%d.", existedSetPrefix, fd.Name(), mapExistedKeyIndex)
			keyOk := dumpYamlIntoMessageFieldValue(k, tempMsg, fd.MapKey(), logger, dumpExistedSet, mapMessagePrefix)
			valueOk := dumpYamlIntoMessageFieldValue(v, tempMsg, fd.MapValue(), logger, dumpExistedSet, mapMessagePrefix)
			if !keyOk || !valueOk {
				continue
			}

			keyValue := tempMsg.ProtoReflect().Get(fd.MapKey())
			valueValue := tempMsg.ProtoReflect().Get(fd.MapValue())

			if !keyValue.IsValid() || !valueValue.IsValid() {
				continue
			}

			dst.ProtoReflect().Mutable(fd).Map().Set(keyValue.MapKey(), valueValue)
			hasValue = true
			if dumpExistedSet != nil {
				mapExistedKey := makeExistedMapKeyIndexKey(existedSetPrefix, fd, keyValue.MapKey())
				dumpExistedSet.MutableExistedSet()[fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), mapExistedKeyIndex)] = struct{}{}
				dumpExistedSet.MutableMapKeyIndex()[mapExistedKey] = mapExistedKeyIndex
			}
		}

		return hasValue
	}

	if fd.IsList() {
		hasValue := false

		innerList, ok := fieldData.([]interface{})
		if !ok {
			// 如果不是数组，fallback为单字段模式
			if dumpYamlIntoMessageFieldValue(fieldData, dst, fd, logger, dumpExistedSet, existedSetPrefix) {
				return true
			}

			return false
		}

		for _, item := range innerList {
			if dumpYamlIntoMessageFieldValue(item, dst, fd, logger, dumpExistedSet, existedSetPrefix) {
				hasValue = true
			} else {
				break
			}
		}

		return hasValue
	} else {
		fieldMatch := false
		if fd.Message() != nil {
			if confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta); confMeta != nil {
				if confMeta.FieldMatch != nil && confMeta.FieldMatch.FieldName != "" && confMeta.FieldMatch.FieldValue != "" {
					// 存在跳过规则
					if value, ok := yamlData[confMeta.FieldMatch.FieldName].(string); ok {
						// 存在
						if value == confMeta.FieldMatch.FieldValue {
							fieldMatch = true
						} else {
							return false
						}
					}
				}
			}
		}

		// 同层级展开
		if fieldMatch && fd.Message() != nil {
			tempMsg := dynamicpb.NewMessage(fd.Message())
			if tempMsg == nil {
				return false
			}

			if dumpYamlIntoMessage(yamlData, tempMsg, logger, dumpExistedSet, existedSetPrefix) {
				dst.ProtoReflect().Set(fd, protoreflect.ValueOfMessage(tempMsg.ProtoReflect()))
				return true
			} else {
				return false
			}
		}

		return dumpYamlIntoMessageFieldValue(fieldData, dst, fd, logger, dumpExistedSet, existedSetPrefix)
	}
}

func dumpYamlIntoMessage(yamlData map[string]interface{}, dst proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if dst == nil {
		return false
	}
	if len(yamlData) == 0 {
		return false
	}

	ret := false

	// protoreflect.Fie
	fields := dst.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		res := dumpYamlIntoMessageFieldItem(yamlData, dst, fd, logger, dumpExistedSet, existedSetPrefix)
		ret = ret || res
	}

	return ret
}

func LoadConfigFromOriginData(originData interface{}, prefixPath string, configPb proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) (err error) {
	parent := originData
	pathParts := strings.Split(prefixPath, ".")
	for i, pathPart := range pathParts {
		trimPart := strings.TrimSpace(pathPart)
		if trimPart == "" {
			continue
		}

		if lu.IsNil(parent) {
			err = fmt.Errorf("LoadConfigFromOriginData data nil")
			break
		}

		arrayIndex, convErr := strconv.Atoi(trimPart)
		if convErr == nil {
			// 数组下标
			parentArray, ok := parent.([]interface{})
			if !ok {
				err = fmt.Errorf("LoadConfigFromOriginData expected array at %s, got %T", strings.Join(pathParts[0:i+1], "."), reflect.TypeOf(parent).Elem().Name())
				break
			}
			if len(parentArray) <= arrayIndex {
				err = fmt.Errorf("LoadConfigFromOriginData array index out of range at %s, got %d >= %d", strings.Join(pathParts[0:i+1], "."), arrayIndex, len(parentArray))
				break
			}
			parent = parentArray[arrayIndex]
		} else {
			// 字符串key
			parentMap, ok := parent.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("LoadConfigFromOriginData expected map at %s, got %T", strings.Join(pathParts[0:i+1], "."), reflect.TypeOf(parent).Elem().Name())
				break
			}
			parent, ok = parentMap[trimPart]
			if !ok {
				err = fmt.Errorf("LoadConfigFromOriginData key not found at %s", strings.Join(pathParts[0:i+1], "."))
				break
			}
		}
	}

	if err != nil {
		logger.Error("load prefixPath failed", "err", err)
		// 使用初始值初始化
		parseErr := LoadDefaultConfigMessageFields(configPb, logger, dumpExistedSet, existedSetPrefix)
		if parseErr != nil {
			logger.Error("LoadDefaultConfigMessageFields failed", "err", parseErr)
		}
		return
	}

	atappData, ok := parent.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("LoadConfigFromOriginData expected map at %s, got %T", strings.Join(pathParts, "."), reflect.TypeOf(parent).Elem().Name())
		return
	}

	dumpYamlIntoMessage(atappData, configPb, logger, dumpExistedSet, existedSetPrefix)
	return
}

func LoadConfigOriginYaml(configPath string) (yamlData map[string]interface{}, err error) {
	var data []byte
	data, err = os.ReadFile(configPath)
	if err != nil {
		return
	}

	yamlData = make(map[string]interface{})
	err = yaml.Unmarshal(data, yamlData)
	return
}

func LoadConfigFromYaml(configPath string, prefixPath string, configPb proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) (yamlData map[string]interface{}, err error) {
	yamlData, err = LoadConfigOriginYaml(configPath)
	if err != nil {
		return
	}

	err = LoadConfigFromOriginData(yamlData, prefixPath, configPb, logger, dumpExistedSet, existedSetPrefix)
	return
}

func dumpEnvironemntIntoMessageFieldValueBasic(envPrefix string, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil {
		return false
	}

	envVal := os.Getenv(envPrefix)
	if envVal == "" {
		return false
	}

	parsedVal, err := parseField(envVal, fd, logger)
	if err != nil {
		return false
	}

	if !parsedVal.IsValid() {
		return false
	}

	if fd.IsList() {
		if parsedVal.IsValid() {
			if dumpExistedSet != nil {
				existedSetKey := fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), dst.ProtoReflect().Get(fd).List().Len())
				dumpExistedSet.MutableExistedSet()[existedSetKey] = struct{}{}
			}

			dst.ProtoReflect().Mutable(fd).List().Append(parsedVal)
			return true
		}

		return false
	}

	existedSetKey := fmt.Sprintf("%s%s", existedSetPrefix, fd.Name())
	if parsedVal.IsValid() {
		existed := false
		if dumpExistedSet != nil {
			_, existed = dumpExistedSet.MutableExistedSet()[existedSetKey]
		}
		if !existed {
			dst.ProtoReflect().Set(fd, parsedVal)
			return true
		}
	} else {
		dst.ProtoReflect().Set(fd, parsedVal)
		if dumpExistedSet != nil {
			dumpExistedSet.MutableExistedSet()[existedSetKey] = struct{}{}
		}
		return true
	}

	return false
}

func dumpEnvironemntIntoMessageFieldValueMessage(envPrefix string, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil {
		return false
	}

	if fd.Message() == nil && !fd.IsMap() {
		return false
	}

	if fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) ||
		fd.Message().FullName() == proto.MessageName(&timestamppb.Timestamp{}) {
		// 基础类型处理
		return dumpEnvironemntIntoMessageFieldValueBasic(envPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
	}

	if fd.IsMap() || fd.IsList() {
		var index int
		if fd.IsMap() {
			index = dst.ProtoReflect().Get(fd).Map().Len()
		} else {
			index = dst.ProtoReflect().Get(fd).List().Len()
		}
		// 使用 dynamicpb 创建临时消息来收集环境变量数据
		tempMsg := dynamicpb.NewMessage(fd.Message())
		if tempMsg == nil {
			return false
		}

		var nextExistedSetPrefix string
		if dumpExistedSet != nil {
			nextExistedSetPrefix = fmt.Sprintf("%s%s.%d.", existedSetPrefix, fd.Name(), index)
		}

		if dumpEnvironemntIntoMessage(envPrefix, tempMsg, logger, dumpExistedSet, nextExistedSetPrefix) {
			if fd.IsMap() {
				mapKeyFd := fd.MapKey()
				mapValueFd := fd.MapValue()
				mapKey := tempMsg.ProtoReflect().Get(mapKeyFd)
				mapValue := tempMsg.ProtoReflect().Get(mapValueFd)
				dst.ProtoReflect().Mutable(fd).Map().Set(mapKey.MapKey(), mapValue)

				if dumpExistedSet != nil {
					dumpExistedSet.MutableMapKeyIndex()[makeExistedMapKeyIndexKey(existedSetPrefix, fd, mapKey.MapKey())] = index
				}
			} else {
				// 使用 AppendMutable 获取正确类型的消息，然后复制字段
				dst.ProtoReflect().Mutable(fd).List().Append(protoreflect.ValueOfMessage(tempMsg.ProtoReflect()))
			}

			if dumpExistedSet != nil {
				dumpExistedSet.MutableExistedSet()[fmt.Sprintf("%s%s.%d", existedSetPrefix, fd.Name(), index)] = struct{}{}
			}
			return true
		} else {
			return false
		}

	} else {
		subMsg := dynamicpb.NewMessage(fd.Message())
		if subMsg == nil {
			return false
		}

		var nextExistedSetPrefix string
		if dumpExistedSet != nil {
			nextExistedSetPrefix = fmt.Sprintf("%s%s.", existedSetPrefix, fd.Name())
		}
		if dumpEnvironemntIntoMessage(envPrefix, subMsg, logger, dumpExistedSet, nextExistedSetPrefix) {
			dst.ProtoReflect().Set(fd, protoreflect.ValueOfMessage(subMsg.ProtoReflect()))
			return true
		} else {
			return false
		}
	}
}

func dumpEnvironemntIntoMessageFieldValue(envPrefix string, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil {
		return false
	}

	if fd.IsMap() || fd.Kind() == protoreflect.MessageKind {
		return dumpEnvironemntIntoMessageFieldValueMessage(envPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
	} else {
		return dumpEnvironemntIntoMessageFieldValueBasic(envPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
	}
}

func dumpEnvironemntIntoMessageFieldItem(envPrefix string, dst proto.Message, fd protoreflect.FieldDescriptor,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if fd == nil || dst == nil {
		return false
	}

	// 将字段名转换为大写形式以匹配环境变量命名惯例
	fieldNameUpper := strings.ToUpper(string(fd.Name()))
	var envKeyPrefix string
	if len(envPrefix) == 0 {
		envKeyPrefix = fieldNameUpper
	} else {
		envKeyPrefix = fmt.Sprintf("%s_%s", envPrefix, fieldNameUpper)
	}

	if fd.IsList() || fd.IsMap() {
		hasValue := false
		for i := 0; ; i++ {
			if dumpEnvironemntIntoMessageFieldValue(fmt.Sprintf("%s_%d", envKeyPrefix, i), dst, fd, logger, dumpExistedSet, existedSetPrefix) {
				hasValue = true
			} else {
				break
			}
		}

		// Fallback to no-index key
		if !hasValue {
			hasValue = dumpEnvironemntIntoMessageFieldValue(envKeyPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
		}
		return hasValue
	} else {
		return dumpEnvironemntIntoMessageFieldValue(envKeyPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
	}
}

func dumpEnvironemntIntoMessage(envPrefix string, dst proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if dst == nil {
		return false
	}

	ret := false

	// protoreflect.Fie
	fields := dst.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		res := dumpEnvironemntIntoMessageFieldItem(envPrefix, dst, fd, logger, dumpExistedSet, existedSetPrefix)
		ret = ret || res
	}

	return ret
}

func LoadConfigFromEnvironemnt(envPrefix string, configPb proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) (bool, error) {
	if logger == nil || configPb == nil {
		return false, fmt.Errorf("dumpEnvironemntIntoMessage logger or configPb is nil")
	}

	return dumpEnvironemntIntoMessage(envPrefix, configPb, logger, dumpExistedSet, existedSetPrefix), nil
}

func dumpDefaultConfigMessageField(configPb proto.Message, fd protoreflect.FieldDescriptor, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) {
	if logger == nil || configPb == nil || fd == nil {
		return
	}

	if fd.ContainingOneof() != nil {
		if configPb.ProtoReflect().WhichOneof(fd.ContainingOneof()) != nil {
			return
		}
	}

	allowStringDefaultValue := fd.Message() == nil && !fd.IsMap()
	if !allowStringDefaultValue && !fd.IsMap() {
		allowStringDefaultValue = fd.Message().FullName() == proto.MessageName(&durationpb.Duration{}) ||
			fd.Message().FullName() == proto.MessageName(&timestamppb.Timestamp{})
	}

	confMeta := proto.GetExtension(fd.Options(), atframe_protocol.E_CONFIGURE).(*atframe_protocol.AtappConfigureMeta)
	if allowStringDefaultValue {
		if confMeta == nil {
			return
		}

		if confMeta.DefaultValue == "" {
			return
		}

		if fd.IsList() {
			return
		}

		if dumpExistedSet != nil {
			checkKey := fmt.Sprintf("%s%s", existedSetPrefix, fd.Name())
			_, existed := dumpExistedSet.MutableExistedSet()[checkKey]
			if existed {
				return
			}
		}
	}

	if fd.Message() == nil && !fd.IsMap() {
		v, err := parseField(confMeta.DefaultValue, fd, logger)
		if err != nil && v.IsValid() {
			if dumpExistedSet != nil {
				dumpExistedSet.MutableExistedSet()[fmt.Sprintf("%s%s", existedSetPrefix, fd.Name())] = struct{}{}
			}
		}
		return
	}

	// Map展开默认值
	if fd.IsMap() {
		mapValueFd := fd.MapValue()
		if mapValueFd.Message() == nil {
			return
		}
		nextMapIndex := 0
		// parseField
		configPb.ProtoReflect().Mutable(fd).Map().Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
			var foundIndex int
			var exists bool = false
			mapIndexKey := makeExistedMapKeyIndexKey(existedSetPrefix, fd, mk)
			if dumpExistedSet != nil {
				foundIndex, exists = dumpExistedSet.MutableMapKeyIndex()[mapIndexKey]
			}
			if !exists {
				foundIndex = nextMapIndex
				nextMapIndex++

				dumpExistedSet.MutableMapKeyIndex()[mapIndexKey] = foundIndex
			}

			LoadDefaultConfigMessageFields(v.Message().Interface(), logger, dumpExistedSet,
				fmt.Sprintf("%s%s.%d.", existedSetPrefix, fd.Name(), foundIndex))
			return true
		})

		return
	}

	// List展开默认值
	if fd.IsList() {
		if fd.Message() == nil {
			return
		}

		list := configPb.ProtoReflect().Get(fd).List()
		for i := 0; i < list.Len(); i++ {
			LoadDefaultConfigMessageFields(list.Get(i).Message().Interface(), logger, dumpExistedSet,
				fmt.Sprintf("%s%s.%d.", existedSetPrefix, fd.Name(), i))
		}
		return
	}

	// 普通Message默认值填充
	subMsg := configPb.ProtoReflect().Get(fd).Message()
	if subMsg == nil {
		return
	}

	LoadDefaultConfigMessageFields(subMsg.Interface(), logger, dumpExistedSet,
		fmt.Sprintf("%s%s.", existedSetPrefix, fd.Name()))
}

func LoadDefaultConfigMessageFields(configPb proto.Message, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) error {
	if logger == nil || configPb == nil {
		return fmt.Errorf("LoadDefaultConfigFields logger or configPb is nil")
	}

	if dumpExistedSet == nil {
		dumpExistedSet = CreateConfigExistIndex()
	}

	fields := configPb.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		dumpDefaultConfigMessageField(configPb, fd, logger, dumpExistedSet, existedSetPrefix)
	}

	return nil
}

// dumpEnvironemntIntoLogCategorySink 从环境变量加载 sink 配置
// 环境变量格式: <envPrefix>_<CATEGORY_NAME>_<sink_index>_<FIELD_NAME>
func dumpEnvironemntIntoLogCategorySink(envPrefix string, categoryName string, categoryIndex int, sink proto.Message, sinkIndex int,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if sink == nil {
		return false
	}

	ret := false
	// 优先使用新格式: <envPrefix>_<CATEGORY_NAME>_<sink_index>
	// 回退到旧格式: <envPrefix>_CATEGORY_<category_index>_SINK_<sink_index>
	sinkEnvPrefixNew := fmt.Sprintf("%s_%s_%d", envPrefix, strings.ToUpper(categoryName), sinkIndex)
	sinkEnvPrefixOld := fmt.Sprintf("%s_CATEGORY_%d_SINK_%d", envPrefix, categoryIndex, sinkIndex)

	// 检查新格式是否存在
	sinkEnvPrefix := sinkEnvPrefixNew
	if os.Getenv(sinkEnvPrefixNew+"_TYPE") == "" && os.Getenv(sinkEnvPrefixOld+"_TYPE") != "" {
		sinkEnvPrefix = sinkEnvPrefixOld
	}

	fields := sink.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		res := dumpEnvironemntIntoMessageFieldItem(sinkEnvPrefix, sink, fd, logger, dumpExistedSet, existedSetPrefix)
		ret = ret || res
	}

	return ret
}

// dumpEnvironemntIntoLogCategory 从环境变量加载 category 配置
func dumpEnvironemntIntoLogCategory(envPrefix string, category *atframe_protocol.AtappLogCategory, categoryIndex int,
	logger *slog.Logger, dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) bool {
	if category == nil {
		return false
	}

	ret := false
	categoryName := category.GetName()
	if categoryName == "" {
		return false
	}

	// 加载 category 的基础字段（除了 sink）
	categoryEnvPrefix := fmt.Sprintf("%s_CATEGORY", envPrefix)
	fields := category.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()

	// 加载 category 的非 sink 字段
	itemEnvPrefix := fmt.Sprintf("%s_%d", categoryEnvPrefix, categoryIndex)
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		// 跳过 sink 字段，后面单独处理
		if fd.Name() == "sink" {
			continue
		}
		res := dumpEnvironemntIntoMessageFieldItem(itemEnvPrefix, category, fd, logger, dumpExistedSet, existedSetPrefix)
		ret = ret || res
	}

	// 加载 sink 列表
	// 优先使用新格式: <envPrefix>_<CATEGORY_NAME>_<sink_index>_<FIELD_NAME>
	// 回退到旧格式: <envPrefix>_CATEGORY_<category_index>_SINK_<sink_index>_<FIELD_NAME>
	for sinkIndex := 0; ; sinkIndex++ {
		// 检查是否存在该 sink（通过检查 TYPE 字段）
		sinkTypeEnvKeyNew := fmt.Sprintf("%s_%s_%d_TYPE", envPrefix, strings.ToUpper(categoryName), sinkIndex)
		sinkTypeEnvKeyOld := fmt.Sprintf("%s_CATEGORY_%d_SINK_%d_TYPE", envPrefix, categoryIndex, sinkIndex)

		sinkType := os.Getenv(sinkTypeEnvKeyNew)
		if sinkType == "" {
			sinkType = os.Getenv(sinkTypeEnvKeyOld)
		}
		if sinkType == "" {
			break
		}

		// 创建新的 sink 并添加到 category
		sinkList := category.ProtoReflect().Mutable(fields.ByName("sink")).List()
		newSink := sinkList.AppendMutable().Message().Interface()

		if dumpEnvironemntIntoLogCategorySink(envPrefix, categoryName, categoryIndex, newSink, sinkIndex, logger, dumpExistedSet, existedSetPrefix) {
			ret = true
		}
	}

	return ret
}

// LoadLogConfigFromEnvironemnt 从环境变量加载日志配置
// 支持特殊的 sink 配置格式: <前缀>_<CATEGORY_NAME>_<sink 下标>_<大写字段名>
func LoadLogConfigFromEnvironemnt(envPrefix string, logConfigPb *atframe_protocol.AtappLog, logger *slog.Logger,
	dumpExistedSet *ConfigExistedIndex, existedSetPrefix string,
) (bool, error) {
	if logger == nil || logConfigPb == nil {
		return false, fmt.Errorf("LoadLogConfigFromEnvironemnt logger or logConfigPb is nil")
	}

	ret := false

	// 加载 log 的基础字段
	fields := logConfigPb.ProtoReflect().Descriptor().Fields()
	fieldSize := fields.Len()
	for i := 0; i < fieldSize; i++ {
		fd := fields.Get(i)
		// 跳过 category 字段，后面单独处理

		if fd.Name() == "category" {
			continue
		}
		res := dumpEnvironemntIntoMessageFieldItem(envPrefix, logConfigPb, fd, logger, dumpExistedSet, existedSetPrefix)
		ret = ret || res
	}

	// 加载 category 列表
	categoryEnvPrefix := fmt.Sprintf("%s_CATEGORY", envPrefix)
	for categoryIndex := 0; ; categoryIndex++ {
		// 检查是否存在该 category（通过检查 NAME 字段）
		categoryNameEnvKey := fmt.Sprintf("%s_%d_NAME", categoryEnvPrefix, categoryIndex)
		categoryName := os.Getenv(categoryNameEnvKey)
		if categoryName == "" {
			break
		}

		// 创建新的 category 并添加到 log
		categoryFd := fields.ByName("category")
		categoryList := logConfigPb.ProtoReflect().Mutable(categoryFd).List()
		newCategory := categoryList.AppendMutable().Message().Interface().(*atframe_protocol.AtappLogCategory)

		// 先设置 name，以便后续处理可以获取
		newCategory.Name = categoryName

		if dumpEnvironemntIntoLogCategory(envPrefix, newCategory, categoryIndex, logger, dumpExistedSet, existedSetPrefix) {
			ret = true
		}
	}

	return ret, nil
}
