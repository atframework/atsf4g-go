package atframework_component_dispatcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionImpl interface {
	MutableActorExecutorFromMessage(startData *DispatcherStartData) *ActorExecutor

	Name() string
	GetTaskId() uint64

	// 最终业务流程执行
	Run(DispatcherStartData) error

	// 允许公共逻辑在运行最终业务流程前执行前置或后置操作
	HookRun(TaskActionImpl, DispatcherStartData) error

	GetActorExecutor() *ActorExecutor
	GetDispatcher() DispatcherImpl
	GetTypeName() string

	GetResponseCode() int32
	SetResponseCode(code int32)

	OnSuccess()
	OnFailed()
	OnTimeout()
	OnComplete()

	// TODO: 链路跟踪逻辑
	GetTraceInheritOption() *TraceInheritOption
	GetTraceStartOption() *TraceStartOption

	// 回包控制
	DisableResponse()
	IsResponseDisabled() bool
	SendResponse() error
}

func popRunActorActions(app_action *libatapp.AppActionData) error {
	cb_actor := app_action.PrivateData.(*ActorExecutor)
	cb_actor.action_lock.Lock()
	defer cb_actor.action_lock.Unlock()

	cb_actor.action_status = ActorExecutorStatusRunning

	// TODO: 单次循环进配置
	max_loop_count := 100

	for i := 0; i < max_loop_count; i++ {
		if cb_actor.pending_actions.Len() == 0 {
			break
		}

		front := cb_actor.pending_actions.Front()
		actor_action := front.Value.(*ActorAction)
		cb_actor.pending_actions.Remove(front)

		if actor_action.callback != nil {
			cb_actor.action_lock.Unlock()

			err := actor_action.callback()

			cb_actor.action_lock.Lock()
			if err != nil {
				app_action.App.GetLogger().Error("Actor task action run failed",
					slog.String("task_name", actor_action.action.Name()), slog.Uint64("task_id", actor_action.action.GetTaskId()), slog.Any("error", err))
			}
		}
	}

	// 如果还有待执行任务，继续执行，切换到pending状态，重新插入队列
	if cb_actor.pending_actions.Len() == 0 {
		cb_actor.action_status = ActorExecutorStatusFree
		return nil
	} else {
		cb_actor.action_status = ActorExecutorStatusPending
		err := app_action.App.PushAction(popRunActorActions, nil, cb_actor)
		if err != nil {
			cb_actor.action_status = ActorExecutorStatusFree
			app_action.App.GetLogger().Error("Push actor task action failed", slog.Any("error", err))
			return err
		}
	}

	return nil
}

func appendActorTaskAction(app libatapp.AppImpl, actor *ActorExecutor, action TaskActionImpl, run_action func() error) error {
	actor.action_lock.Lock()
	defer actor.action_lock.Unlock()

	// 如果队列过长，直接失败放弃
	// TODO: 走接口，进配置
	if actor.pending_actions.Len() > 100000 {
		action.SetResponseCode(-2)
		action.SendResponse()
		app.GetLogger().Error("Actor pending actions too many", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Int("response_code", int(action.GetResponseCode())))
		return fmt.Errorf("actor pending actions too many")
	}

	actor.pending_actions.PushBack(&ActorAction{
		action:   action,
		callback: run_action,
	})

	// TODO: 如果不在待执行队列中，插入待执行队列
	if actor.action_status == ActorExecutorStatusFree {
		actor.action_status = ActorExecutorStatusPending
		err := app.PushAction(popRunActorActions, nil, actor)
		if err != nil {
			actor.action_status = ActorExecutorStatusFree
			app.GetLogger().Error("Push actor task action failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			return err
		}
	}
	return nil
}

func RunTaskAction(app libatapp.AppImpl, action TaskActionImpl, startData DispatcherStartData) error {
	run_action := func() error {
		// TODO: 链路跟踪Start和对象上下文继承

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
