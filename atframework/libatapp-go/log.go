package libatapp

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

type logBuffer []byte

// Having an initial size gives a dramatic speedup.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*logBuffer)(&b)
	},
}

func newlogBuffer() *logBuffer {
	return bufPool.Get().(*logBuffer)
}

func (b *logBuffer) Free() {
	*b = (*b)[:0]
	bufPool.Put(b)
}

func (b *logBuffer) String() string {
	return lu.BytestoString(*b)
}

func (b *logBuffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *logBuffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

func (b *logBuffer) WriteByte(c byte) {
	*b = append(*b, c)
}

func (b *logBuffer) Bytes() []byte {
	return *b
}

func (b *logBuffer) Len() int {
	return len(*b)
}

func (b *logBuffer) AppendLogLevel(level slog.Level) {
	switch {
	case level >= slog.LevelError:
		b.WriteString("[ERROR]")
	case level >= slog.LevelWarn:
		b.WriteString("[ WARN]")
	case level >= slog.LevelInfo:
		b.WriteString("[ INFO]")
	default:
		b.WriteString("[DEBUG]")
	}
}

type timestampCacheEntry struct {
	second int64
	prefix string
}

var appendTimestampCache atomic.Value

func init() {
	appendTimestampCache.Store(timestampCacheEntry{second: -1})
}

func appendTimestamp(b *logBuffer, t time.Time) {
	entry := appendTimestampCache.Load().(timestampCacheEntry)
	second := t.Unix()
	if entry.second != second {
		prefix := t.Format(time.DateTime)
		entry = timestampCacheEntry{second: second, prefix: prefix}
		appendTimestampCache.Store(entry)
	}

	*b = append(*b, entry.prefix...)
	*b = append(*b, '.')
	millis := int(t.Nanosecond() / 1_000_000)
	*b = append(*b, byte('0'+millis/100))
	millis %= 100
	*b = append(*b, byte('0'+millis/10))
	*b = append(*b, byte('0'+millis%10))
}

type LogWriter interface {
	io.Writer
	// 在Reload后切换日志时需要Close
	Close()
	// 某些带缓存的Writer 需要手动Flush
	Flush() error
}

type logHandlerWriter struct {
	out LogWriter

	minLevel slog.Level
	maxLevel slog.Level

	enableStackTrace bool
	stackTraceLevel  slog.Level

	autoFlushLevel slog.Level
}

type logHandlerImpl struct {
	writers []logHandlerWriter

	frameInfoCache *sync.Map // pc -> runtime.Frame
}

type frameInfo struct {
	function string
	file     string
	line     int
}

func (h *logHandlerImpl) getFrameInfo(pc uintptr) *frameInfo {
	if f, ok := h.frameInfoCache.Load(pc); ok {
		return f.(*frameInfo)
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	info := frameInfo{
		function: frame.Function,
		file:     filepath.Base(frame.File),
		line:     frame.Line,
	}
	h.frameInfoCache.Store(pc, &info)
	return &info
}

// 获取完整堆栈（缓存）
func (h *logHandlerImpl) getStack(pc uintptr) string {
	buf := make([]uintptr, 32)
	n := runtime.Callers(3, buf)
	// 找到pc所在位置
	for i := 0; i < n; i++ {
		if buf[i] == pc {
			buf = buf[i:]
			n -= i
			break
		}
	}

	var stack []*frameInfo
	for i := range buf[:n] {
		if buf[i] == 0 {
			break
		}
		stack = append(stack, h.getFrameInfo(buf[i]))
	}

	sb := newlogBuffer()
	defer sb.Free()
	for _, f := range stack {
		sb.WriteString("  at ")
		sb.WriteString(f.function)
		sb.WriteString(" (")
		sb.WriteString(f.file)
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(f.line))
		sb.WriteString(")\n")
	}

	return sb.String()
}

func (h *logHandlerWriter) Enabled(level slog.Level) bool {
	return level >= h.minLevel && level <= h.maxLevel
}

func (h *logHandlerImpl) Enabled(_ context.Context, level slog.Level) bool {
	for k := range h.writers {
		if h.writers[k].Enabled(level) {
			return true
		}
	}
	return false
}

// Handle不需要是线程安全
func (h *logHandlerImpl) Handle(_ context.Context, r slog.Record) error {
	// 主信息
	sb := newlogBuffer()
	defer sb.Free()
	sb.WriteByte('[')
	appendTimestamp(sb, r.Time)
	sb.WriteString("]")
	sb.AppendLogLevel(r.Level)
	sb.WriteString("(")
	if r.PC != 0 {
		frameInfo := h.getFrameInfo(r.PC)
		sb.WriteString(frameInfo.file)
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(frameInfo.line))
	} else {
		sb.WriteString("unknown:0")
	}
	sb.WriteString("): ")
	sb.WriteString(r.Message)

	// 额外字段
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteByte(' ')
		sb.WriteString(a.Key)
		sb.WriteByte('=')
		sb.WriteString(a.Value.Resolve().String())
		return true
	})
	sb.WriteByte('\n')

	var stackTrace *logBuffer
	for k := range h.writers {
		if !h.writers[k].Enabled(r.Level) {
			continue
		}
		// 写入基础日志
		if r.PC != 0 && h.writers[k].enableStackTrace && r.Level >= h.writers[k].stackTraceLevel {
			// 需要StackTrace
			if stackTrace == nil {
				// 生成
				stackTrace = newlogBuffer()
				stackTrace.Write(sb.Bytes())
				stackTrace.WriteString("Stacktrace:\n")
				stackTrace.WriteString(h.getStack(r.PC))
				defer stackTrace.Free()
			}
			// 写入StackTrace
			h.writers[k].out.Write(stackTrace.Bytes())
		} else {
			h.writers[k].out.Write(sb.Bytes())
		}

		if r.Level >= h.writers[k].autoFlushLevel {
			h.writers[k].out.Flush()
		}
	}
	return nil
}

func (h *logHandlerImpl) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // 简化实现，不处理
}

func (h *logHandlerImpl) WithGroup(name string) slog.Handler {
	return h // 简化实现，不处理
}

func ConvertLogLevel(level string) slog.Level {
	switch {
	case level == "debug" || level == "DEBUG":
		return slog.LevelDebug
	case level == "info" || level == "INFO":
		return slog.LevelInfo
	case level == "warn" || level == "warning" || level == "WARN" || level == "WARNING":
		return slog.LevelWarn
	case level == "error" || level == "ERROR":
		return slog.LevelError
	case level == "fatal" || level == "FATAL":
		return slog.LevelError
	}

	return slog.LevelInfo
}

func GetCaller(skip int) uintptr {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller, and skip]
	runtime.Callers(2+skip, pcs[:])
	return pcs[0]
}

func LogInner(sysnow time.Time, logger *slog.Logger, pc uintptr, ctx context.Context, level slog.Level, msg string, args ...any) {
	if lu.IsNil(ctx) {
		ctx = context.Background()
	}
	if !logger.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(sysnow, level, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}
