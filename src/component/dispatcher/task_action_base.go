package atframework_component_dispatcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionBase struct {
	impl   TaskActionImpl
	taskId uint64
	status TaskActionStatus
	kill   atomic.Bool

	responseCode     int32
	prepareHookRun   bool
	awaitableContext AwaitableContext
	startTime        time.Time
	timeout          time.Duration

	actorExecutor *ActorExecutor
	dispatcher    DispatcherImpl

	disableResponse bool

	currentAwaiting *struct {
		Lock    sync.Mutex
		Option  *DispatcherAwaitOptions
		Channel *chan TaskActionAwaitChannelData
	}

	initCallbackLock sync.Mutex
	onFinishCallback []func(RpcContext)
	callbackFinish   bool
}

func CreateTaskActionBase(rd DispatcherImpl, actorExecutor *ActorExecutor, timeout time.Duration) TaskActionBase {
	return TaskActionBase{
		taskId:           libatapp.AtappGetModule[*TaskManager](GetReflectTypeTaskManager(), rd.GetApp()).AllocTaskId(),
		status:           TaskActionStatusCreated,
		responseCode:     0,
		prepareHookRun:   false,
		awaitableContext: nil,
		startTime:        rd.GetSysNow(),
		timeout:          timeout,
		actorExecutor:    actorExecutor,
		dispatcher:       rd,
		disableResponse:  false,
		currentAwaiting: &struct {
			Lock    sync.Mutex
			Option  *DispatcherAwaitOptions
			Channel *chan TaskActionAwaitChannelData
		}{
			Lock:    sync.Mutex{},
			Option:  nil,
			Channel: nil,
		},
	}
}

func (t *TaskActionBase) GetTaskId() uint64 {
	return t.taskId
}

func (t *TaskActionBase) GetTaskStartTime() time.Time {
	return t.startTime
}

func (t *TaskActionBase) GetTaskTimeout() time.Duration {
	return t.timeout
}

func (t *TaskActionBase) GetNow() time.Time {
	if !lu.IsNil(t.awaitableContext) {
		return t.awaitableContext.GetNow()
	}

	return t.dispatcher.GetNow()
}

func (t *TaskActionBase) GetSysNow() time.Time {
	if !lu.IsNil(t.awaitableContext) {
		return t.awaitableContext.GetSysNow()
	}

	return t.dispatcher.GetSysNow()
}

func (t *TaskActionBase) SetImplementation(impl TaskActionImpl) {
	t.impl = impl
}

func (t *TaskActionBase) GetStatus() TaskActionStatus {
	if t == nil {
		return TaskActionStatusInvalid
	}

	if t.status >= TaskActionStatusDone {
		return t.status
	}

	if t.kill.Load() {
		return TaskActionStatusKilled
	}

	return t.status
}

func (t *TaskActionBase) IsExiting() bool {
	return t.GetStatus() >= TaskActionStatusDone
}

func (t *TaskActionBase) IsRunning() bool {
	return t.GetStatus() == TaskActionStatusRunning
}

func (t *TaskActionBase) IsFault() bool {
	return t.GetStatus() >= TaskActionStatusKilled
}

func (t *TaskActionBase) IsTimeout() bool {
	return t.GetStatus() == TaskActionStatusTimeout
}

func (t *TaskActionBase) CheckPermission() (int32, error) {
	if !t.impl.AllowNoActor() && t.impl.GetActorExecutor() == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY), nil
	}

	return 0, nil
}

func (t *TaskActionBase) setStatus(status TaskActionStatus) {
	if t == nil {
		return
	}

	t.status = status
}

func (t *TaskActionBase) PrepareHookRun(startData *DispatcherStartData) {
	if t == nil {
		return
	}

	if t.prepareHookRun {
		return
	}
	t.prepareHookRun = true

	if startData != nil && lu.IsNil(t.awaitableContext) {
		t.awaitableContext = startData.MessageRpcContext
	}

	if t.GetStatus() <= TaskActionStatusCreated {
		t.setStatus(TaskActionStatusRunning)
	}
}

func (t *TaskActionBase) HookRun(startData *DispatcherStartData) error {
	t.PrepareHookRun(startData)

	responseCode, err := t.impl.CheckPermission()
	if err != nil || responseCode < 0 {
		t.impl.SetResponseCode(responseCode)
		t.setStatus(TaskActionStatusKilled)
		return err
	}

	err = t.impl.Run(startData)
	if t.GetStatus() <= TaskActionStatusRunning {
		if err != nil || t.GetResponseCode() < 0 {
			if t.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT) || errors.Is(err, context.DeadlineExceeded) {
				t.setStatus(TaskActionStatusTimeout)
			} else {
				t.setStatus(TaskActionStatusKilled)
			}
		} else {
			t.setStatus(TaskActionStatusDone)
		}
	}

	return err
}

func (t *TaskActionBase) GetActorExecutor() *ActorExecutor {
	return t.actorExecutor
}

func (t *TaskActionBase) GetDispatcher() DispatcherImpl {
	return t.dispatcher
}

func (t *TaskActionBase) GetTypeName() string {
	return "TaskActionBase [type not set]"
}

func (t *TaskActionBase) GetResponseCode() int32 {
	return t.responseCode
}

func (t *TaskActionBase) SetResponseCode(code int32) {
	t.responseCode = code
}

func (t *TaskActionBase) SetResponseError(code public_protocol_pbdesc.EnErrorCode) {
	t.responseCode = int32(code)
}

func (t *TaskActionBase) DisableResponse() {
	t.disableResponse = true
}

func (t *TaskActionBase) IsResponseDisabled() bool {
	return t.disableResponse
}

func (t *TaskActionBase) SendResponse() error {
	return nil
}

func (t *TaskActionBase) OnSuccess() {}

func (t *TaskActionBase) OnFailed() {}

func (t *TaskActionBase) OnTimeout() {}

func (t *TaskActionBase) OnComplete() {}

func (t *TaskActionBase) OnCleanup() {
	if t.currentAwaiting.Channel != nil {
		close(*t.currentAwaiting.Channel)
		t.currentAwaiting.Channel = nil
	}
	t.initCallbackLock.Lock()
	for _, v := range t.onFinishCallback {
		v(t.GetRpcContext())
	}
	t.callbackFinish = true
	t.initCallbackLock.Unlock()
}

func (t *TaskActionBase) GetTraceInheritOption() *TraceInheritOption {
	return &TraceInheritOption{}
}

func (t *TaskActionBase) GetTraceStartOption() *TraceStartOption {
	return &TraceStartOption{}
}

func (t *TaskActionBase) GetRpcContext() RpcContext {
	return t.awaitableContext
}

func (t *TaskActionBase) GetAwaitableContext() AwaitableContext {
	return t.awaitableContext
}

func (t *TaskActionBase) trySetAwait(awaitOptions *DispatcherAwaitOptions) error {
	if awaitOptions == nil {
		return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions can not be nil", t.impl.Name(), t.impl.GetTaskId())
	}

	actor := t.impl.GetActorExecutor()
	if actor != nil {
		currentAction := actor.getCurrentRunningAction()
		if currentAction != nil && currentAction != t.impl {
			return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions failed, action is running in actor, can not await", t.impl.Name(), t.impl.GetTaskId())
		}
	}

	if t.currentAwaiting.Option == awaitOptions {
		return nil
	}

	if t.currentAwaiting.Option != nil {
		if t.currentAwaiting.Option.Type == awaitOptions.Type && t.currentAwaiting.Option.Sequence == awaitOptions.Sequence {
			// 相同等待选项，直接复用
			return nil
		}

		return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions failed, already awaiting %v:%v , can not await %v:%v again",
			t.impl.Name(), t.impl.GetTaskId(), t.currentAwaiting.Option.Type, t.currentAwaiting.Option.Sequence,
			awaitOptions.Type, awaitOptions.Sequence,
		)
	}

	t.currentAwaiting.Option = awaitOptions
	return nil
}

func (t *TaskActionBase) TrySetupAwait(awaitOptions *DispatcherAwaitOptions) (*chan TaskActionAwaitChannelData, error) {
	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	err := t.trySetAwait(awaitOptions)
	if err != nil {
		return nil, err
	}

	if t.currentAwaiting.Channel == nil {
		ch := make(chan TaskActionAwaitChannelData, 1)
		t.currentAwaiting.Channel = &ch
	}

	return t.currentAwaiting.Channel, nil
}

func (t *TaskActionBase) TryFinishAwait(resumeData *DispatcherResumeData, notify bool) error {
	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	if resumeData == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData can not be nil", t.impl.Name(), t.impl.GetTaskId())
	}

	if t.currentAwaiting.Option == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait no current awaiting", t.impl.Name(), t.impl.GetTaskId())
	}

	if t.currentAwaiting.Option.Type != resumeData.Message.Type || t.currentAwaiting.Option.Sequence != resumeData.Sequence {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData mismatch, current awaiting %v:%v , got %v:%v",
			t.impl.Name(), t.impl.GetTaskId(), t.currentAwaiting.Option.Type, t.currentAwaiting.Option.Sequence,
			resumeData.Message.Type, resumeData.Sequence,
		)
	}

	if !notify {
		t.currentAwaiting.Option = nil
	} else {
		if t.currentAwaiting.Channel == nil {
			return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", t.impl.Name(), t.impl.GetTaskId())
		}

		select {
		case *t.currentAwaiting.Channel <- TaskActionAwaitChannelData{resume: resumeData}:
			t.currentAwaiting.Option = nil
		default:
			return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", t.impl.Name(), t.impl.GetTaskId())
		}
	}

	return nil
}

func (t *TaskActionBase) tryKillAwait(killData *RpcResult) error {
	if t.currentAwaiting.Option == nil {
		return nil
	}

	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	if killData == nil {
		return fmt.Errorf("task %s, %d TryKillAwait killData can not be nil", t.impl.Name(), t.impl.GetTaskId())
	}

	if t.currentAwaiting.Option == nil {
		return fmt.Errorf("task %s, %d TryKillAwait no current awaiting", t.impl.Name(), t.impl.GetTaskId())
	}

	if t.currentAwaiting.Channel == nil {
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", t.impl.Name(), t.impl.GetTaskId())
	}

	select {
	case *t.currentAwaiting.Channel <- TaskActionAwaitChannelData{killed: killData}:
		t.currentAwaiting.Option = nil
	default:
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", t.impl.Name(), t.impl.GetTaskId())
	}

	return nil
}

func (t *TaskActionBase) TryKill(killData *RpcResult) error {
	t.kill.Store(true)
	return t.TryKillAwait(killData)
}

func (t *TaskActionBase) TryKillAwait(killData *RpcResult) error {
	err := t.tryKillAwait(killData)
	return err
}

func (t *TaskActionBase) InitFinishCallback(callback func(RpcContext)) {
	t.initCallbackLock.Lock()
	if t.callbackFinish {
		// 直接调用
		callback(t.GetRpcContext())
	} else {
		t.onFinishCallback = append(t.onFinishCallback, callback)
	}
	t.initCallbackLock.Unlock()
}

// ====================== 业务日志接口 =========================

func (t *TaskActionBase) LogWithLevel(level slog.Level, msg string, args ...any) {
	t.GetRpcContext().LogWithLevelWithCaller(libatapp.GetCaller(1), level, msg, args...)
}

func (t *TaskActionBase) LogError(msg string, args ...any) {
	t.GetRpcContext().LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelError, msg, args...)
}

func (t *TaskActionBase) LogWarn(msg string, args ...any) {
	t.GetRpcContext().LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelWarn, msg, args...)
}

func (t *TaskActionBase) LogInfo(msg string, args ...any) {
	t.GetRpcContext().LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelInfo, msg, args...)
}

func (t *TaskActionBase) LogDebug(msg string, args ...any) {
	t.GetRpcContext().LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelDebug, msg, args...)
}
