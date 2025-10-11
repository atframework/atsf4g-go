package atframework_component_dispatcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionAwaitChannelData struct {
	resume *DispatcherResumeData
	killed *RpcResult
}

type TaskActionImpl interface {
	Name() string
	GetTaskId() uint64
	GetTaskStartTime() time.Time
	GetNow() time.Time

	// 最终业务流程执行
	Run(*DispatcherStartData) error

	// 权限检查（通常在解包后，Run之前执行）
	// 如果权限检查失败，返回 (错误，错误码) 或 (nil, 错误码) Run不会被调用
	// 如果权限检查成功，返回 (nil, 0) ，Run会被调用
	CheckPermission(TaskActionImpl) (int32, error)

	// 允许公共逻辑在运行最终业务流程前执行前置或后置操作
	HookRun(TaskActionImpl, *DispatcherStartData) error

	// 是否允许无Actor执行
	AllowNoActor() bool

	GetActorExecutor() *ActorExecutor
	GetDispatcher() DispatcherImpl
	GetTypeName() string

	GetResponseCode() int32
	SetResponseCode(code int32)

	OnSuccess()
	OnFailed()
	OnTimeout()
	OnComplete()
	OnCleanup()

	// TODO: 链路跟踪逻辑
	GetTraceInheritOption(TaskActionImpl) *TraceInheritOption
	GetTraceStartOption(TaskActionImpl) *TraceStartOption
	GetRpcContext() *RpcContext

	// 回包控制
	DisableResponse()
	IsResponseDisabled() bool
	SendResponse() error

	// 切出等待管理
	TrySetupAwait(action TaskActionImpl, awaitOptions *DispatcherAwaitOptions) (*chan TaskActionAwaitChannelData, error)
	TryFinishAwait(action TaskActionImpl, resumeData *DispatcherResumeData) error
	TryKillAwait(action TaskActionImpl, killData *RpcResult) error
}

func popRunActorActions(app_action *libatapp.AppActionData) error {
	cb_actor := app_action.PrivateData.(*ActorExecutor)
	cb_actor.actionLock.Lock()
	defer cb_actor.actionLock.Unlock()

	cb_actor.actionStatus = ActorExecutorStatusFree

	// TODO: 单次循环进配置
	max_loop_count := 100

	for i := 0; i < max_loop_count; i++ {
		if cb_actor.pendingActions.Len() == 0 {
			break
		}

		if cb_actor.getCurrentRunningAction() != nil {
			// 当前有任务在运行，该任务会重新触发排队，跳出循环
			break
		}

		front := cb_actor.pendingActions.Front()
		actor_action := front.Value.(*ActorAction)
		cb_actor.pendingActions.Remove(front)

		if actor_action.callback != nil {
			cb_actor.actionLock.Unlock()

			err := actor_action.callback()

			cb_actor.actionLock.Lock()
			if err != nil {
				app_action.App.GetLogger().Error("Actor task action run failed",
					slog.String("task_name", actor_action.action.Name()), slog.Uint64("task_id", actor_action.action.GetTaskId()), slog.Any("error", err))
			}
		}
	}

	// 如果还有待执行任务，继续执行，切换到pending状态，重新插入队列
	if cb_actor.pendingActions.Len() == 0 {
		cb_actor.actionStatus = ActorExecutorStatusFree
		return nil
	} else {
		cb_actor.actionStatus = ActorExecutorStatusPending
		err := app_action.App.PushAction(popRunActorActions, nil, cb_actor)
		if err != nil {
			cb_actor.actionStatus = ActorExecutorStatusFree
			app_action.App.GetLogger().Error("Push actor task action failed", slog.Any("error", err))
			return err
		}
	}

	return nil
}

func appendActorTaskAction(app libatapp.AppImpl, actor *ActorExecutor, action TaskActionImpl, run_action func() error) error {
	actor.actionLock.Lock()
	defer actor.actionLock.Unlock()

	// 如果队列过长，直接失败放弃
	// TODO: 走接口，进配置
	pendingLen := actor.pendingActions.Len()
	if action != nil && run_action != nil {
		if pendingLen > 100000 {
			action.SetResponseCode(-2)
			action.SendResponse()
			app.GetLogger().Error("Actor pending actions too many", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Int("response_code", int(action.GetResponseCode())))
			return fmt.Errorf("actor pending actions too many")
		}

		pendingLen += 1
		actor.pendingActions.PushBack(&ActorAction{
			action:   action,
			callback: run_action,
		})
	}

	// 如果不在待执行队列中，插入待执行队列
	if pendingLen > 0 && actor.actionStatus == ActorExecutorStatusFree {
		actor.actionStatus = ActorExecutorStatusPending
		err := app.PushAction(popRunActorActions, nil, actor)
		if err != nil {
			actor.actionStatus = ActorExecutorStatusFree
			app.GetLogger().Error("Push actor task action failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			return err
		}
	}
	return nil
}

func RunTaskAction(app libatapp.AppImpl, action TaskActionImpl, startData *DispatcherStartData) error {
	run_action := func() error {
		// TODO: 链路跟踪Start和对象上下文继承

		actor := action.GetActorExecutor()
		if actor != nil {
			actor.takeCurrentRunningAction(action)
		}
		cleanupCurrentAction := func() {
			if actor != nil {
				actor.releaseCurrentRunningAction(app, action, false)
			}

			if startData != nil && startData.MessageRpcContext.CancelFn != nil {
				cancelFn := startData.MessageRpcContext.CancelFn
				startData.MessageRpcContext.CancelFn = nil
				cancelFn()
			}

			action.OnCleanup()
		}
		defer cleanupCurrentAction()

		err := action.HookRun(action, startData)

		if err != nil {
			if action.GetResponseCode() == 0 {
				// 未设置错误
				action.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_UNKNOWN))
			}

			app.GetLogger().Error("TaskAction run failed",
				slog.String("task_name", action.Name()),
				slog.Uint64("task_id", action.GetTaskId()),
				slog.Any("error", err), slog.Int("response_code", int(action.GetResponseCode())))

			// 超时错误码/错误
			if action.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT) || errors.Is(err, context.DeadlineExceeded) {
				action.OnTimeout()
			}
			action.OnFailed()

		} else {
			if action.GetResponseCode() < 0 {
				app.GetLogger().Error("TaskAction run failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Int("response_code", int(action.GetResponseCode())))

				// 超时错误码
				if action.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT) {
					action.OnTimeout()
				}
				action.OnFailed()
			} else {
				app.GetLogger().Debug("TaskAction run success", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()))

				action.OnSuccess()
			}
		}

		action.OnComplete()

		if !action.IsResponseDisabled() {
			err = action.SendResponse()
			if err != nil {
				app.GetLogger().Error("TaskAction send response failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			}
		}

		// TODO: 链路跟踪End
		// TODO: 任务完成事件

		return err
	}

	// 任务调度层排队
	actor := action.GetActorExecutor()
	if actor != nil {
		return appendActorTaskAction(app, actor, action, run_action)
	} else {
		return run_action()
	}
}

func YieldTaskAction(app libatapp.AppImpl, action TaskActionImpl, awaitOptions *DispatcherAwaitOptions) (*DispatcherResumeData, *RpcResult) {
	// TODO: 已经超时或者被Killed，不允许再切出

	// 暂停任务逻辑, 让出令牌
	awaitChannel, err := action.TrySetupAwait(action, awaitOptions)
	if err != nil || awaitChannel == nil {
		app.GetLogger().Error("task YieldTaskAction TrySetupAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		return nil, &RpcResult{Error: err, ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)}
	}
	actor := action.GetActorExecutor()
	if actor != nil {
		actor.releaseCurrentRunningAction(app, action, true)
	}

	// TODO: TaskManager管理超时和等待数据

	// Wait for either resume or kill data from the awaitChannel
	awaitResult, ok := <-*awaitChannel
	// You can now use awaitResult.resume or awaitResult.killed as needed

	// 恢复占用令牌
	if actor != nil {
		actor.takeCurrentRunningAction(action)
	}

	if !ok {
		// Channel was closed unexpectedly
		app.GetLogger().Error("task YieldTaskAction await channel closed unexpectedly", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()))
		return nil, &RpcResult{Error: errors.New("await channel closed"), ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)}
	}

	return awaitResult.resume, awaitResult.killed
}

func ResumeTaskAction(app libatapp.AppImpl, action TaskActionImpl, resumeData *DispatcherResumeData) error {
	// TODO: TaskManager移除超时和等待数据

	err := action.TryFinishAwait(action, resumeData)
	if err != nil {
		app.GetLogger().Error("task ResumeTaskAction TryFinishAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		return err
	}

	return nil
}

func KillTaskAction(app libatapp.AppImpl, action TaskActionImpl, killData *RpcResult) error {
	// TODO: TaskManager移除超时和等待数据

	err := action.TryKillAwait(action, killData)
	if err != nil {
		app.GetLogger().Error("task KillTaskAction TryKillAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		return err
	}
	return nil
}
