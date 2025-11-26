package atframework_component_dispatcher

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

var taskManagerReflectType reflect.Type

func init() {
	var _ libatapp.AppModuleImpl = (*TaskManager)(nil)
	taskManagerReflectType = reflect.TypeOf((*TaskManager)(nil)).Elem()
}

func GetReflectTypeTaskManager() reflect.Type {
	return taskManagerReflectType
}

type TaskManager struct {
	libatapp.AppModuleBase
	taskActionIdMap sync.Map
	taskIdAllocator atomic.Uint64
}

func CreateTaskManager(owner libatapp.AppImpl) *TaskManager {
	return &TaskManager{
		AppModuleBase: libatapp.CreateAppModuleBase(owner),
	}
}

func (t *TaskManager) Name() string { return "TaskManager" }

func (t *TaskManager) GetReflectType() reflect.Type {
	return taskManagerReflectType
}

func (t *TaskManager) Init(initCtx context.Context) error {
	return nil
}

func (t *TaskManager) Tick(parent context.Context) bool {
	return false
}

func (t *TaskManager) GetTaskActionById(taskId uint64) TaskActionImpl {
	if v, ok := t.taskActionIdMap.Load(taskId); ok {
		if action, ok := v.(TaskActionImpl); ok {
			return action
		}
	}
	return nil
}

func (t *TaskManager) AllocTaskId() uint64 {
	ret := t.taskIdAllocator.Add(1)
	if ret <= 1 {
		t.taskIdAllocator.Store(
			// 使用时间戳作为初始值, 避免与重启前的值冲突
			uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()),
		)
		ret = t.taskIdAllocator.Add(1)
	}

	return ret
}

func (t *TaskManager) InsertTaskAction(ctx RpcContext, action TaskActionImpl) {
	t.taskActionIdMap.Store(action.GetTaskId(), action)
	if action.GetTaskTimeout() != 0 {
		timer := time.AfterFunc(action.GetTaskTimeout(), func() {
			result := CreateRpcResultError(context.DeadlineExceeded, public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
			KillTaskAction(ctx, action, &result)
		})
		if timer != nil {
			action.InitFinishCallback(func(childCtx RpcContext) {
				timer.Stop()
			})
		}
	}
	action.InitFinishCallback(func(childCtx RpcContext) {
		t.taskActionIdMap.Delete(action.GetTaskId())
	})
}

// TODO: AfterFunc 使用时间轮

func (t *TaskManager) StartTaskAction(ctx RpcContext, action TaskActionImpl, startData *DispatcherStartData) error {
	run_action := func() error {
		// TODO: 链路跟踪Start和对象上下文继承
		actor := action.GetActorExecutor()
		if actor != nil {
			actor.takeCurrentRunningAction(action)
		}
		cleanupCurrentAction := func() {
			if actor != nil {
				actor.releaseCurrentRunningAction(t.GetApp(), action, false)
			}

			if startData != nil && startData.MessageRpcContext.GetCancelFn() != nil {
				cancelFn := startData.MessageRpcContext.GetCancelFn()
				startData.MessageRpcContext.SetCancelFn(nil)
				cancelFn()
			}

			action.OnCleanup()
		}
		defer cleanupCurrentAction()

		err := action.HookRun(startData)

		if err != nil {
			if action.GetResponseCode() == 0 {
				// 未设置错误
				action.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_UNKNOWN))
			}

			t.GetApp().GetDefaultLogger().Error("TaskAction run failed",
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
				t.GetApp().GetDefaultLogger().Error("TaskAction run failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Int("response_code", int(action.GetResponseCode())))

				// 超时错误码
				if action.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT) {
					action.OnTimeout()
				}
				action.OnFailed()
			} else {
				t.GetApp().GetDefaultLogger().Debug("TaskAction run success", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()))

				action.OnSuccess()
			}
		}

		action.OnComplete()

		if !action.IsResponseDisabled() {
			err = action.SendResponse()
			if err != nil {
				t.GetApp().GetDefaultLogger().Error("TaskAction send response failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			}
		}

		// TODO: 链路跟踪End

		return err
	}

	// 任务调度层排队
	actor := action.GetActorExecutor()
	if actor != nil {
		return appendActorTaskAction(t.GetApp(), actor, action, run_action)
	}
	return t.GetApp().PushAction(func(appAction *libatapp.AppActionData) error {
		err := run_action()
		if err != nil {
			ctx.LogError("Actor task action run failed",
				slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		}
		return nil
	}, nil, nil)
}

func YieldTaskAction(ctx AwaitableContext, action TaskActionImpl, awaitOptions *DispatcherAwaitOptions, beforeYield BeforeYieldAction) (*DispatcherResumeData, *RpcResult) {
	currentTask := ctx.GetAction()
	if lu.IsNil(currentTask) {
		ctx.LogError("should in task")
		result := CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
		return nil, &result
	}

	// 已经超时或者被Killed，不允许再切出
	if currentTask.IsExiting() {
		ctx.LogError("current task already Exiting")
		result := CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
		return nil, &result
	}

	// 暂停任务逻辑, 让出令牌
	awaitChannel, err := action.TrySetupAwait(awaitOptions)
	if err != nil || awaitChannel == nil {
		ctx.LogError("task YieldTaskAction TrySetupAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		result := CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return nil, &result
	}

	if beforeYield != nil {
		result := beforeYield()
		if result.IsError() {
			// 释放等待逻辑 继续执行
			err := action.TryFinishAwait(&DispatcherResumeData{
				Message: &DispatcherRawMessage{
					Type: awaitOptions.Type,
				},
				Sequence: awaitOptions.Sequence,
			}, false)
			if err != nil {
				ctx.LogError("task ResumeTaskAction TryFinishAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			}
			return nil, &result
		}
	}

	actor := action.GetActorExecutor()
	if actor != nil {
		actor.releaseCurrentRunningAction(ctx.GetApp(), action, true)
	}

	// 注册等待超时
	var awaitTimer *time.Timer
	if awaitOptions.Timeout != 0 {
		awaitTimer = time.AfterFunc(awaitOptions.Timeout, func() {
			var result RpcResult
			if awaitOptions.TimeoutAllow {
				result = CreateRpcResultOk()
			} else {
				result = CreateRpcResultError(context.DeadlineExceeded, public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
			}
			awaitTimer.Stop()
			awaitTimer = nil
			err := action.TryKillAwait(&result)
			if err != nil {
				ctx.LogError("task TryKillAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			}
		})
	}

	// Wait for either resume or kill data from the awaitChannel
	awaitResult, ok := <-*awaitChannel
	// You can now use awaitResult.resume or awaitResult.killed as needed

	if awaitTimer != nil {
		awaitTimer.Stop()
	}

	// 恢复占用令牌
	if actor != nil {
		actor.takeCurrentRunningAction(action)
	}

	if !ok {
		// Channel was closed unexpectedly
		ctx.LogError("task YieldTaskAction await channel closed unexpectedly", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()))
		return nil, &RpcResult{Error: errors.New("await channel closed"), ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)}
	}

	if lu.IsNil(awaitResult.killed) {
		awaitResult.killed = &RpcResult{
			Error:        nil,
			ResponseCode: 0,
		}
	}

	return awaitResult.resume, awaitResult.killed
}

func ResumeTaskAction(ctx RpcContext, action TaskActionImpl, resumeData *DispatcherResumeData) error {
	err := action.TryFinishAwait(resumeData, true)
	if err != nil {
		ctx.LogError("task ResumeTaskAction TryFinishAwait failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		return err
	}

	return nil
}

func KillTaskAction(ctx RpcContext, action TaskActionImpl, killData *RpcResult) error {
	err := action.TryKill(killData)
	if err != nil {
		ctx.LogError("task KillTaskAction TryKill failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
		return err
	}
	return nil
}

func AsyncInvoke(ctx RpcContext, name string, actor *ActorExecutor, invoke func(childCtx AwaitableContext) RpcResult) TaskActionImpl {
	childTask, startData := CreateNoMessageTaskAction(libatapp.AtappGetModule[*NoMessageDispatcher](GetReflectTypeNoMessageDispatcher(), ctx.GetApp()),
		ctx, actor, func(rd DispatcherImpl, actor *ActorExecutor, timeout time.Duration) *taskActionAsyncInvoke {
			ta := &taskActionAsyncInvoke{
				TaskActionNoMessageBase: CreateNoMessageTaskActionBase(rd, actor, timeout),
				name:                    name,
				callable:                invoke,
			}
			return ta
		},
	)

	if err := libatapp.AtappGetModule[*TaskManager](GetReflectTypeTaskManager(), ctx.GetApp()).StartTaskAction(ctx, childTask, &startData); err != nil {
		ctx.LogError("AsyncInvoke StartTaskAction failed", slog.String("task_name", childTask.Name()), slog.Any("error", err))
		return nil
	}

	return childTask
}

func AsyncThen(ctx RpcContext, name string, actor *ActorExecutor, waiting TaskActionImpl, invoke func()) {
	if lu.IsNil(waiting) || waiting.IsExiting() {
		invoke()
	}
	taskAction := AsyncInvoke(ctx, name, actor, func(childCtx AwaitableContext) RpcResult {
		result := AwaitTask(childCtx, waiting)
		invoke()
		return result
	})
	if lu.IsNil(taskAction) {
		ctx.LogError("Try to invoke task failed, try to call it directly", "name", name)
		invoke()
	}
}

func AsyncThenStartTask(ctx RpcContext, actor *ActorExecutor, waiting TaskActionImpl, startTask TaskActionImpl, startData *DispatcherStartData) {
	AsyncThen(ctx, startTask.Name(), actor, waiting, func() {
		if err := libatapp.AtappGetModule[*TaskManager](GetReflectTypeTaskManager(), ctx.GetApp()).StartTaskAction(ctx, startTask, startData); err != nil {
			ctx.LogError("AsyncInvoke StartTaskAction failed", slog.String("task_name", startTask.Name()), slog.Any("error", err))
		}
	})
}

func Wait(ctx AwaitableContext, waitTime time.Duration) RpcResult {
	currentTask := ctx.GetAction()
	if lu.IsNil(currentTask) || currentTask.GetTaskId() == 0 {
		ctx.LogError("should in task")
		return CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	if waitTime <= 0 {
		return CreateRpcResultOk()
	}

	_, result := YieldTaskAction(ctx, currentTask, &DispatcherAwaitOptions{
		Type:         0,
		Sequence:     currentTask.GetTaskId(),
		Timeout:      waitTime,
		TimeoutAllow: true,
	}, nil)
	return *result
}

func AwaitTask(ctx AwaitableContext, waitingTask TaskActionImpl) RpcResult {
	currentTask := ctx.GetAction()
	if lu.IsNil(currentTask) || currentTask.GetTaskId() == 0 {
		ctx.LogError("should in task")
		return CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}
	// 等待某一Task
	if lu.IsNil(waitingTask) || waitingTask.IsExiting() {
		return CreateRpcResultOk()
	}

	if currentTask.GetTaskId() == waitingTask.GetTaskId() {
		return CreateRpcResultOk()
	}

	_, result := YieldTaskAction(ctx, currentTask, &DispatcherAwaitOptions{
		Type:     uint64(uintptr(unsafe.Pointer(&currentTask))),
		Sequence: waitingTask.GetTaskId(),
	}, func() RpcResult {
		waitingTask.InitFinishCallback(func(childCtx RpcContext) {
			err := ResumeTaskAction(childCtx, currentTask, &DispatcherResumeData{
				Message: &DispatcherRawMessage{
					Type: uint64(uintptr(unsafe.Pointer(&currentTask))),
				},
				Sequence:    waitingTask.GetTaskId(),
				PrivateData: nil,
			})
			if err != nil {
				childCtx.LogError("ResumeTaskAction Failed", "err", err)
			}
		})
		return CreateRpcResultOk()
	})
	return *result
}

func AwaitTasks(ctx AwaitableContext, waitingTasks []TaskActionImpl) RpcResult {
	currentTask := ctx.GetAction()
	if lu.IsNil(currentTask) || currentTask.GetTaskId() == 0 {
		ctx.LogError("should in task")
		return CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}
	for i := range waitingTasks {
		if currentTask.IsTimeout() {
			return CreateRpcResultError(context.DeadlineExceeded, public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
		}

		if currentTask.IsFault() || currentTask.IsExiting() {
			return CreateRpcResultError(context.DeadlineExceeded, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_EXITING)
		}
		AwaitTask(ctx, waitingTasks[i])
	}
	return CreateRpcResultOk()
}
