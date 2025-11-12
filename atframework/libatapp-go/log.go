package libatapp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"sync"
	"time"
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
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

func (b *logBuffer) String() string {
	return string(*b)
}

func (b *logBuffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *logBuffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

func (b *logBuffer) Len() int {
	return len(*b)
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
		sb.WriteString(fmt.Sprintf("  at %s (%s:%d)\n", f.function, f.file, f.line))
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
	// 时间
	ts := r.Time.Format(time.DateTime)

	// 文件 + 行号
	var file string
	if r.PC != 0 {
		frameInfo := h.getFrameInfo(r.PC)
		file = fmt.Sprintf("%s:%d", frameInfo.file, frameInfo.line)
	}

	// 主信息
	sb := newlogBuffer()
	defer sb.Free()
	fmt.Fprintf(sb, "[%s][%s](%s): %s", ts, r.Level.String(), file, r.Message)

	// 额外字段
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(sb, " %s=%v", a.Key, a.Value)
		return true
	})
	fmt.Fprintf(sb, "\n")

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
				fmt.Fprintf(stackTrace, "%sStacktrace:\n%s", sb.String(), h.getStack(r.PC))
			}
			// 写入StackTrace
			h.writers[k].out.Write([]byte(stackTrace.String()))
		} else {
			h.writers[k].out.Write([]byte(sb.String()))
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

func LogInner(logger *slog.Logger, pc uintptr, ctx context.Context, level slog.Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !logger.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}
