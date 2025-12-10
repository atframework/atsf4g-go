package atframework_component_dispatcher

import (
	"context"
	"log/slog"
	"time"

	libatapp "github.com/atframework/libatapp-go"

	logical_time "github.com/atframework/atsf4g-go/component-logical_time"
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
	GetSysNow() time.Time
	GetApp() libatapp.AppImpl
	GetAction() TaskActionImpl
	BindAction(action TaskActionImpl)

	GetContext() context.Context
	GetCancelFn() context.CancelFunc
	SetContext(ctx context.Context)
	SetCancelFn(cancelFn context.CancelFunc)
	SetContextCancelFn(ctx context.Context, cancelFn context.CancelFunc)

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

func (ctx *RpcContextImpl) getInternalLogger() *libatapp.Logger {
	if ctx.app != nil {
		return ctx.app.GetDefaultLogger()
	}

	return nil
}

func (ctx *RpcContextImpl) GetNow() time.Time {
	if ctx.dispatcher != nil {
		return ctx.dispatcher.GetNow()
	}

	return logical_time.GetLogicalNow()
}

func (ctx *RpcContextImpl) GetSysNow() time.Time {
	if ctx.dispatcher != nil {
		return ctx.dispatcher.GetSysNow()
	}

	return logical_time.GetSysNow()
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

// ====================== 通用日志接口 =========================

func (ctx *RpcContextImpl) LogWithLevelContextWithCaller(pc uintptr, c context.Context, level slog.Level, msg string, args ...any) {
	var logger *libatapp.Logger = nil
	if ctx != nil {
		logger = ctx.getInternalLogger()

		if c == nil {
			c = ctx.rpcContext
		}
	}

	if ctx != nil {
		if ctx.taskAction != nil {
			args = append(args, slog.Uint64("task_id", ctx.taskAction.GetTaskId()), slog.String("task_name", ctx.taskAction.Name()))
			attr := ctx.taskAction.GetActorExecutor().LogAttr()
			for _, a := range attr {
				args = append(args, a)
			}
		} else if ctx.dispatcher != nil {
			args = append(args, slog.String("dispatcher", ctx.dispatcher.Name()))
		}
		logger.LogInner(ctx.GetSysNow(), pc, c, level, msg, args...)
	} else {
		logger.LogInner(logical_time.GetSysNow(), pc, c, level, msg, args...)
	}
}

func (ctx *RpcContextImpl) LogWithLevelWithCaller(pc uintptr, level slog.Level, msg string, args ...any) {
	if ctx == nil || ctx.rpcContext == nil {
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
