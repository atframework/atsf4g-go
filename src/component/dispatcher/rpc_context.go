package atframework_component_dispatcher

import (
	"context"
	"log/slog"
	"time"

	libatapp "github.com/atframework/libatapp-go"
)

type RpcContext struct {
	app        libatapp.AppImpl
	dispatcher DispatcherImpl
	taskAction TaskActionImpl

	Context  context.Context
	CancelFn context.CancelFunc
}

func (dispatcher *DispatcherBase) CreateRpcContext(rd DispatcherImpl) *RpcContext {
	return &RpcContext{
		app:        dispatcher.GetApp(),
		dispatcher: rd,
	}
}

func (ctx *RpcContext) GetInternalLogger() *slog.Logger {
	if ctx.app != nil {
		return ctx.app.GetDefaultLogger()
	}

	return slog.Default()
}

func (ctx *RpcContext) GetNow() time.Time {
	if ctx.dispatcher != nil {
		return ctx.dispatcher.GetNow()
	}

	return time.Now()
}

func (ctx *RpcContext) LogWithLevelContext(c context.Context, level slog.Level, msg string, args ...any) {
	var logger *slog.Logger = nil
	if ctx != nil {
		logger = ctx.GetInternalLogger()

		if c == nil {
			c = ctx.Context
		}
	}
	if logger == nil {
		logger = slog.Default()
	}

	if ctx != nil {
		if ctx.taskAction != nil {
			args = append(args, slog.Uint64("task_id", ctx.taskAction.GetTaskId()), slog.String("task_name", ctx.taskAction.Name()))
		} else if ctx.dispatcher != nil {
			args = append(args, slog.String("dispatcher", ctx.dispatcher.Name()))
		}
	}

	logger.Log(c, level, msg, args...)
}

func (ctx *RpcContext) LogWithLevel(level slog.Level, msg string, args ...any) {
	if ctx == nil || ctx.Context == nil {
		ctx.LogWithLevelContext(context.Background(), level, msg, args...)
	} else {
		ctx.LogWithLevelContext(ctx.Context, level, msg, args...)
	}
}

func (ctx *RpcContext) LogErrorContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContext(c, slog.LevelError, msg, args...)
}

func (ctx *RpcContext) LogError(msg string, args ...any) {
	ctx.LogWithLevel(slog.LevelError, msg, args...)
}

func (ctx *RpcContext) LogWarnContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContext(c, slog.LevelWarn, msg, args...)
}

func (ctx *RpcContext) LogWarn(msg string, args ...any) {
	ctx.LogWithLevel(slog.LevelWarn, msg, args...)
}

func (ctx *RpcContext) LogInfoContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContext(c, slog.LevelInfo, msg, args...)
}

func (ctx *RpcContext) LogInfo(msg string, args ...any) {
	ctx.LogWithLevel(slog.LevelInfo, msg, args...)
}

func (ctx *RpcContext) LogDebugContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContext(c, slog.LevelDebug, msg, args...)
}

func (ctx *RpcContext) LogDebug(msg string, args ...any) {
	ctx.LogWithLevel(slog.LevelDebug, msg, args...)
}
