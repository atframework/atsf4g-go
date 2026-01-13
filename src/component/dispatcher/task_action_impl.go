package atframework_component_dispatcher

import (
	"fmt"
	"log/slog"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	config "github.com/atframework/atsf4g-go/component-config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionAwaitChannelData struct {
	resume *DispatcherResumeData
	killed *RpcResult
}

type BeforeYieldAction func() RpcResult
type BeforeYieldEnsureCallAction func()

type TaskActionStatus int32

const (
	TaskActionStatusInvalid TaskActionStatus = 0
	TaskActionStatusCreated TaskActionStatus = 1
	TaskActionStatusRunning TaskActionStatus = 2
	TaskActionStatusDone    TaskActionStatus = 3
	TaskActionStatusKilled  TaskActionStatus = 4
	TaskActionStatusTimeout TaskActionStatus = 5
)

type TaskActionImpl interface {
	Name() string
	GetTaskId() uint64
	GetTaskStartTime() time.Time
	GetNow() time.Time
	GetSysNow() time.Time
	SetImplementation(TaskActionImpl)

	GetStatus() TaskActionStatus
	IsExiting() bool
	IsRunning() bool
	IsFault() bool
	IsTimeout() bool

	// 最终业务流程执行
	Run(*DispatcherStartData) error

	// 权限检查（通常在解包后，Run之前执行）
	// 如果权限检查失败，返回 (错误，错误码) 或 (nil, 错误码) Run不会被调用
	// 如果权限检查成功，返回 (nil, 0) ，Run会被调用
	CheckPermission() (int32, error)

	// 允许公共逻辑在运行最终业务流程前执行前置或后置操作
	HookRun(*DispatcherStartData) error

	// 是否允许无Actor执行
	AllowNoActor() bool

	SetActorExecutor(*ActorExecutor)
	GetActorExecutor() *ActorExecutor
	GetDispatcher() DispatcherImpl
	GetTypeName() string

	GetResponseCode() int32
	SetResponseCode(code int32)
	GetTaskTimeout() time.Duration

	OnSuccess()
	OnFailed()
	OnTimeout()
	OnComplete()
	OnCleanup()
	OnSendResponse()

	// TODO: 链路跟踪逻辑
	GetTraceInheritOption() *TraceInheritOption
	GetTraceStartOption() *TraceStartOption
	GetRpcContext() RpcContext

	// 回包控制
	DisableResponse()
	IsResponseDisabled() bool
	SendResponse() error

	// 切出等待管理
	TrySetupAwait(awaitOptions *DispatcherAwaitOptions) (*chan TaskActionAwaitChannelData, error)
	TryFinishAwait(resumeData *DispatcherResumeData, notify bool) error
	TryKillAwait(killData *RpcResult) error
	TryKill(killData *RpcResult) error

	InitFinishCallback(callback func(RpcContext))
}

func popRunActorActions(app_action *libatapp.AppActionData) error {
	cb_actor := app_action.PrivateData.(*ActorExecutor)
	cb_actor.actionLock.Lock()
	cb_actor.actionStatus = ActorExecutorStatusFree

	max_loop_count := int(config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetTask().GetActorMaxLoopCount())

	for i := 0; i < max_loop_count; i++ {
		if cb_actor.pendingActions.Len() == 0 {
			break
		}

		if !lu.IsNil(cb_actor.getCurrentRunningAction()) {
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
				app_action.App.GetDefaultLogger().LogError("Actor task action run failed",
					slog.String("task_name", actor_action.action.Name()), slog.Uint64("task_id", actor_action.action.GetTaskId()), slog.Any("error", err))
			}
		}
	}

	// 如果还有待执行任务，继续执行，切换到pending状态，重新插入队列
	if cb_actor.pendingActions.Len() == 0 {
		cb_actor.actionStatus = ActorExecutorStatusFree
		cb_actor.actionLock.Unlock() // 手动释放 放置Panic时覆盖堆栈
		return nil
	} else {
		cb_actor.actionStatus = ActorExecutorStatusPending
		err := app_action.App.PushAction(popRunActorActions, nil, cb_actor)
		if err != nil {
			cb_actor.actionStatus = ActorExecutorStatusFree
			app_action.App.GetDefaultLogger().LogError("Push actor task action failed", slog.Any("error", err))
			cb_actor.actionLock.Unlock()
			return err
		}
	}
	cb_actor.actionLock.Unlock()
	return nil
}

func appendActorTaskAction(app libatapp.AppImpl, actor *ActorExecutor, action TaskActionImpl, run_action func() error) error {
	actor.actionLock.Lock()
	defer actor.actionLock.Unlock()

	// 如果队列过长，直接失败放弃
	// TODO: 走接口，进配置
	pendingLen := actor.pendingActions.Len()
	if !lu.IsNil(action) && !lu.IsNil(run_action) {
		if pendingLen > int(config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetTask().GetActorMaxPendingCount()) {
			action.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_ACTOR_MAX_PENDING_COUNT))
			action.SendResponse()
			app.GetDefaultLogger().LogError("Actor pending actions too many", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Int("response_code", int(action.GetResponseCode())))
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
			app.GetDefaultLogger().LogError("Push actor task action failed", slog.String("task_name", action.Name()), slog.Uint64("task_id", action.GetTaskId()), slog.Any("error", err))
			return err
		}
	}
	return nil
}
