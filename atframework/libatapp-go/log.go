package libatapp

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type logFileHandler struct {
	level            slog.Level
	enableStackTrace bool
	stackTraceLevel  slog.Level

	frameInfoCache sync.Map // pc -> runtime.Frame
	stackCache     sync.Map // stackKey -> string
}

type logFileInfo struct {
	file string
	line int
}

func (h *logFileHandler) getFrameInfo(pc uintptr) *logFileInfo {
	if f, ok := h.frameInfoCache.Load(pc); ok {
		return f.(*logFileInfo)
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	info := logFileInfo{
		file: filepath.Base(frame.File),
		line: frame.Line,
	}
	h.frameInfoCache.Store(pc, &info)
	return &info
}

// 获取完整堆栈（缓存）
func (h *logFileHandler) getStack(pc uintptr) string {
	if s, ok := h.stackCache.Load(pc); ok {
		return s.(string)
	}

	buf := make([]uintptr, 32)
	n := runtime.Callers(5, buf)

	frames := runtime.CallersFrames(buf[:n])

	type frameInfo struct {
		fn, file string
		line     int
	}
	var stack []frameInfo
	for {
		f, more := frames.Next()
		stack = append(stack, frameInfo{fn: f.Function, file: filepath.Base(f.File), line: f.Line})
		if !more {
			break
		}
	}
	trimCount := 2
	if len(stack) > trimCount {
		stack = stack[:len(stack)-trimCount]
	}

	var sb strings.Builder
	for _, f := range stack {
		sb.WriteString(fmt.Sprintf("  at %s (%s:%d)\n", f.fn, f.file, f.line))
	}

	stackStr := sb.String()
	h.stackCache.Store(pc, stackStr)
	return stackStr
}

func (h *logFileHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *logFileHandler) Handle(_ context.Context, r slog.Record) error {
	// 时间
	ts := r.Time.Format(time.DateTime)

	// 文件 + 行号
	var file string
	if r.PC != 0 {
		frameInfo := h.getFrameInfo(r.PC)
		file = fmt.Sprintf("%s:%d", frameInfo.file, frameInfo.line)
	}

	// 主信息
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "[%s][%s](%s): %s", r.Level.String(), ts, file, r.Message)

	// 额外字段
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(sb, " %s=%v", a.Key, a.Value)
		return true
	})

	if h.enableStackTrace && r.Level >= h.stackTraceLevel && r.PC != 0 {
		sb.WriteString("\nStacktrace:\n")
		sb.WriteString(h.getStack(r.PC))
	}

	fmt.Println(sb.String())
	return nil
}

func (h *logFileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // 简化实现，不处理
}

func (h *logFileHandler) WithGroup(name string) slog.Handler {
	return h // 简化实现，不处理
}

func (h *logFileHandler) SetLevel(level slog.Level) {
	h.level = level
}

func (h *logFileHandler) SetStackTrace(enable bool, level slog.Level) {
	h.enableStackTrace = enable
	h.stackTraceLevel = level
}
