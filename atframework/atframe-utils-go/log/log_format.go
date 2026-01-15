package libatapp

import (
	"log/slog"
	"strconv"
	"time"
)

type CallerInfo struct {
	Now         time.Time
	LogLevel    slog.Level
	RotateIndex uint32
	Frame       *frameInfo
}

func LogFormat(format string, sb LogFormatBufferWriter, caller CallerInfo, customFormatP func(LogFormatBufferWriter, time.Time)) string {
	if format == "" {
		return ""
	}

	levelName := LevelNameResolver(caller.LogLevel)

	now := caller.Now.In(time.Local)
	parts := newTimeParts(now)

	needParse := false

	for i := 0; i < len(format); i++ {
		ch := format[i]
		if !needParse {
			if ch == '%' {
				needParse = true
				continue
			}
			sb.WriteByte(ch)
			continue
		}
		needParse = false
		if ch == '%' {
			needParse = true
			continue
		}
		switch ch {
		case 'Y': // 四位年份
			appendFourDigits(sb, parts.year)
		case 'y': // 两位年份
			appendTwoDigits(sb, parts.year%100)
		case 'm': // 两位月份
			appendTwoDigits(sb, parts.month)
		case 'j': // 三位年份中的天数
			appendThreeDigits(sb, parts.yearDay)
		case 'd': // 两位日期
			appendTwoDigits(sb, parts.day)
		case 'w': // 星期几
			sb.WriteByte(byte('0' + parts.weekDay))
		case 'H': // 24小时制，两位小时数
			appendTwoDigits(sb, parts.hour)
		case 'I': // 12小时制，两位小时数
			appendTwoDigits(sb, parts.hour12)
		case 'M': // 两位分钟数
			appendTwoDigits(sb, parts.minute)
		case 'S': // 两位秒数
			appendTwoDigits(sb, parts.second)
		case 'F': // 等同于 %Y-%m-%d
			appendDate(sb, parts)
		case 'T': // 等同于 %H:%M:%S
			appendTime(sb, parts)
		case 'P': // 等同于 %Y-%m-%d %H:%M:%S.毫秒
			if customFormatP != nil {
				customFormatP(sb, now)
				break
			}
			appendDate(sb, parts)
			sb.WriteByte(' ')
			appendTime(sb, parts)
			sb.WriteByte('.')
			appendThreeDigits(sb, now.Nanosecond()/1_000_000)
		case 'R': // 等同于 %H:%M
			appendHourMinute(sb, parts)
		case 'f': // 微秒，五位
			appendSubSecond(sb, now)
		case 'L': // 日志级别名称
			sb.WriteString(levelName)
		case 'l': // 日志级别ID
			sb.WriteString(strconv.Itoa(int(caller.LogLevel)))
		case 's': // 文件路径，项目目录用~代替
			if caller.Frame != nil {
				appendFilePath(sb, caller.Frame.file)
			} else {
				sb.WriteString("unknow_path")
			}
		case 'k': // 仅文件名
			if caller.Frame != nil {
				appendFileName(sb, caller.Frame.file)
			} else {
				sb.WriteString("unknow_file")
			}
		case 'n': // 行号
			if caller.Frame != nil {
				sb.WriteString(strconv.FormatUint(uint64(caller.Frame.line), 10))
			} else {
				sb.WriteString("0")
			}
		case 'C': // 函数名
			if caller.Frame != nil && caller.Frame.function != "" {
				sb.WriteString(caller.Frame.function)
			} else {
				sb.WriteString("unknow_function")
			}
		case 'N': // 日志文件轮转索引
			sb.WriteString(strconv.FormatUint(uint64(caller.RotateIndex), 10))
		default: // 原样输出
			sb.WriteByte(ch)
		}
	}

	return sb.String()
}

func LevelNameResolver(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return " WARN"
	case level >= slog.LevelInfo:
		return " INFO"
	default:
		return "DEBUG"
	}
}

type timeParts struct {
	year    int
	month   int
	day     int
	hour    int
	hour12  int
	minute  int
	second  int
	weekDay int
	yearDay int
}

func newTimeParts(t time.Time) timeParts {
	hour12 := t.Hour()%12 + 1
	return timeParts{
		year:    t.Year(),
		month:   int(t.Month()),
		day:     t.Day(),
		hour:    t.Hour(),
		hour12:  hour12,
		minute:  t.Minute(),
		second:  t.Second(),
		weekDay: int(t.Weekday()),
		yearDay: t.YearDay() - 1,
	}
}

func appendTwoDigits(sb LogFormatBufferWriter, value int) {
	sb.WriteByte(byte('0' + value/10))
	sb.WriteByte(byte('0' + value%10))
}

func appendThreeDigits(sb LogFormatBufferWriter, value int) {
	sb.WriteByte(byte('0' + value/100))
	sb.WriteByte(byte('0' + (value/10)%10))
	sb.WriteByte(byte('0' + value%10))
}

func appendFourDigits(sb LogFormatBufferWriter, value int) {
	sb.WriteByte(byte('0' + value/1000))
	sb.WriteByte(byte('0' + (value/100)%10))
	sb.WriteByte(byte('0' + (value/10)%10))
	sb.WriteByte(byte('0' + value%10))
}

func appendDate(sb LogFormatBufferWriter, parts timeParts) {
	appendFourDigits(sb, parts.year)
	sb.WriteByte('-')
	appendTwoDigits(sb, parts.month)
	sb.WriteByte('-')
	appendTwoDigits(sb, parts.day)
}

func appendTime(sb LogFormatBufferWriter, parts timeParts) {
	appendTwoDigits(sb, parts.hour)
	sb.WriteByte(':')
	appendTwoDigits(sb, parts.minute)
	sb.WriteByte(':')
	appendTwoDigits(sb, parts.second)
}

func appendHourMinute(sb LogFormatBufferWriter, parts timeParts) {
	appendTwoDigits(sb, parts.hour)
	sb.WriteByte(':')
	appendTwoDigits(sb, parts.minute)
}

func appendSubSecond(sb LogFormatBufferWriter, now time.Time) {
	micro := now.Nanosecond() / 1_000
	value := micro / 10
	digits := [5]byte{}
	for i := 4; i >= 0; i-- {
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	sb.Write(digits[:])
}

func appendLevelName(sb LogFormatBufferWriter, name string) {
	if name == "" {
		return
	}
	if len(name) > 5 {
		name = name[:5]
	}
	sb.WriteString(name)
	for i := len(name); i < 5; i++ {
		sb.WriteByte(' ')
	}
}

func appendFilePath(sb LogFormatBufferWriter, filePath string) {
	if filePath == "" {
		return
	}
	sb.WriteString(filePath)
}

func appendFileName(sb LogFormatBufferWriter, filePath string) {
	if filePath == "" {
		return
	}
	last := -1
	for i := 0; i < len(filePath); i++ {
		if filePath[i] == '/' || filePath[i] == '\\' {
			last = i
		}
	}
	sb.WriteString(filePath[last+1:])
}

func commonPrefixLength(a, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	strip := 0
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			break
		}
		strip = i + 1
	}
	return strip
}
