package atframework_component_dispatcher

import (
	"context"
	"log/slog"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	libatapp "github.com/atframework/libatapp-go"
)

type RpcContextImpl struct {
	app        libatapp.AppImpl
	dispatcher DispatcherImpl
	taskAction TaskActionImpl

	rpcContext context.Context
	cancelFn   context.CancelFunc
}

type AwaitableContextImpl struct {
	RpcContextImpl
}

type RpcContext interface {
	GetNow() time.Time
	GetApp() libatapp.AppImpl
	GetAction() TaskActionImpl
	BindAction(action TaskActionImpl)

	GetContext() context.Context
	GetCancelFn() context.CancelFunc
	SetContext(ctx context.Context)
	SetCancelFn(cancelFn context.CancelFunc)
	SetContextCancelFn(ctx context.Context, cancelFn context.CancelFunc)
	SetTaskAction(action TaskActionImpl)

	// ====================== 通用日志接口 =========================

	LogWithLevelContextWithCaller(pc uintptr, c context.Context, level slog.Level, msg string, args ...any)
	LogWithLevelWithCaller(pc uintptr, level slog.Level, msg string, args ...any)

	LogErrorContext(c context.Context, msg string, args ...any)
	LogError(msg string, args ...any)
	LogWarnContext(c context.Context, msg string, args ...any)
	LogWarn(msg string, args ...any)
	LogInfoContext(c context.Context, msg string, args ...any)
	LogInfo(msg string, args ...any)
	LogDebugContext(c context.Context, msg string, args ...any)
	LogDebug(msg string, args ...any)
}

type AwaitableContext interface {
	Awaitable()
	RpcContext
}

func (ctx *AwaitableContextImpl) Awaitable() {}

func (ctx *RpcContextImpl) getInternalLogger() *slog.Logger {
	if ctx.app != nil {
		return ctx.app.GetDefaultLogger()
	}

	return slog.Default()
}

func (ctx *RpcContextImpl) GetNow() time.Time {
	if ctx.dispatcher != nil {
		return ctx.dispatcher.GetNow()
	}

	return time.Now()
}

func (ctx *RpcContextImpl) GetApp() libatapp.AppImpl {
	return ctx.app
}

func (ctx *RpcContextImpl) GetAction() TaskActionImpl {
	return ctx.taskAction
}

func (ctx *RpcContextImpl) BindAction(action TaskActionImpl) {
	ctx.taskAction = action
}

func (ctx *RpcContextImpl) GetContext() context.Context {
	return ctx.rpcContext
}

func (ctx *RpcContextImpl) GetCancelFn() context.CancelFunc {
	return ctx.cancelFn
}

func (ctx *RpcContextImpl) SetContext(c context.Context) {
	ctx.rpcContext = c
}

func (ctx *RpcContextImpl) SetCancelFn(cancelFn context.CancelFunc) {
	ctx.cancelFn = cancelFn
}

func (ctx *RpcContextImpl) SetContextCancelFn(c context.Context, cancelFn context.CancelFunc) {
	ctx.rpcContext = c
	ctx.cancelFn = cancelFn
}

func (ctx *RpcContextImpl) SetTaskAction(action TaskActionImpl) {
	ctx.taskAction = action
}

// ====================== 通用日志接口 =========================

func (ctx *RpcContextImpl) LogWithLevelContextWithCaller(pc uintptr, c context.Context, level slog.Level, msg string, args ...any) {
	var logger *slog.Logger = nil
	if ctx != nil {
		logger = ctx.getInternalLogger()

		if c == nil {
			c = ctx.rpcContext
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

	libatapp.LogInner(logger, pc, c, level, msg, args...)
}

func (ctx *RpcContextImpl) LogWithLevelWithCaller(pc uintptr, level slog.Level, msg string, args ...any) {
	if lu.IsNil(ctx) || ctx.rpcContext == nil {
		ctx.LogWithLevelContextWithCaller(pc, context.Background(), level, msg, args...)
	} else {
		ctx.LogWithLevelContextWithCaller(pc, ctx.rpcContext, level, msg, args...)
	}
}

// ====================== 业务日志接口 =========================

func (ctx *RpcContextImpl) LogErrorContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelError, msg, args...)
}

func (ctx *RpcContextImpl) LogError(msg string, args ...any) {

	ctx.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelError, msg, args...)
}

func (ctx *RpcContextImpl) LogWarnContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelWarn, msg, args...)
}

func (ctx *RpcContextImpl) LogWarn(msg string, args ...any) {
	ctx.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelWarn, msg, args...)
}

func (ctx *RpcContextImpl) LogInfoContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelInfo, msg, args...)
}

func (ctx *RpcContextImpl) LogInfo(msg string, args ...any) {
	ctx.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelInfo, msg, args...)
}

func (ctx *RpcContextImpl) LogDebugContext(c context.Context, msg string, args ...any) {
	ctx.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelDebug, msg, args...)
}

func (ctx *RpcContextImpl) LogDebug(msg string, args ...any) {
	ctx.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelDebug, msg, args...)
}
