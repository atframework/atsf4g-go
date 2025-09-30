package atframework_component_dispatcher

import (
	"fmt"
	"sync"
	"time"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type TaskActionBase struct {
	taskId uint64

	responseCode   int32
	prepareHookRun bool
	rpcContext     *RpcContext
	startTime      time.Time

	actorExecutor *ActorExecutor
	dispatcher    DispatcherImpl

	disableResponse bool

	currentAwaiting *struct {
		Lock    sync.Mutex
		Option  *DispatcherAwaitOptions
		Channel *chan TaskActionAwaitChannelData
	}
}

func CreateTaskActionBase(rd DispatcherImpl, actorExecutor *ActorExecutor) TaskActionBase {
	return TaskActionBase{
		taskId:          rd.AllocSequence(),
		responseCode:    0,
		prepareHookRun:  false,
		rpcContext:      nil,
		startTime:       rd.GetNow(),
		actorExecutor:   actorExecutor,
		dispatcher:      rd,
		disableResponse: false,
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

func (t *TaskActionBase) GetNow() time.Time {
	return t.dispatcher.GetNow()
}

func (t *TaskActionBase) CheckPermission(action TaskActionImpl) (int32, error) {
	if !action.AllowNoActor() && action.GetActorExecutor() == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY), nil
	}

	return 0, nil
}

func (t *TaskActionBase) PrepareHookRun(action TaskActionImpl, startData *DispatcherStartData) {
	if t.prepareHookRun {
		return
	}
	t.prepareHookRun = true

	if startData != nil && t.rpcContext == nil {
		t.rpcContext = startData.MessageRpcContext
	}
}

func (t *TaskActionBase) HookRun(action TaskActionImpl, startData *DispatcherStartData) error {
	t.PrepareHookRun(action, startData)

	responseCode, err := action.CheckPermission(action)
	if err != nil || responseCode < 0 {
		action.SetResponseCode(responseCode)
		return err
	}

	return action.Run(startData)
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
}

func (t *TaskActionBase) GetTraceInheritOption(_action TaskActionImpl) *TraceInheritOption {
	return &TraceInheritOption{}
}

func (t *TaskActionBase) GetTraceStartOption(_action TaskActionImpl) *TraceStartOption {
	return &TraceStartOption{}
}

func (t *TaskActionBase) GetRpcContext() *RpcContext {
	return t.rpcContext
}

func (t *TaskActionBase) trySetAwait(action TaskActionImpl, awaitOptions *DispatcherAwaitOptions) error {
	if awaitOptions == nil {
		return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions can not be nil", action.Name(), action.GetTaskId())
	}

	actor := action.GetActorExecutor()
	if actor != nil {
		currentAction := actor.getCurrentRunningAction()
		if currentAction != nil && currentAction != action {
			return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions failed, action is running in actor, can not await", action.Name(), action.GetTaskId())
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
			action.Name(), action.GetTaskId(), t.currentAwaiting.Option.Type, t.currentAwaiting.Option.Sequence,
			awaitOptions.Type, awaitOptions.Sequence,
		)
	}

	return nil
}

func (t *TaskActionBase) TrySetupAwait(action TaskActionImpl, awaitOptions *DispatcherAwaitOptions) (*chan TaskActionAwaitChannelData, error) {
	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	err := t.trySetAwait(action, awaitOptions)
	if err != nil {
		return nil, err
	}

	if t.currentAwaiting.Channel == nil {
		ch := make(chan TaskActionAwaitChannelData, 1)
		t.currentAwaiting.Channel = &ch
	}

	return t.currentAwaiting.Channel, nil
}

func (t *TaskActionBase) TryFinishAwait(action TaskActionImpl, resumeData *DispatcherResumeData) error {
	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	if resumeData == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData can not be nil", action.Name(), action.GetTaskId())
	}

	if t.currentAwaiting.Option == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait no current awaiting", action.Name(), action.GetTaskId())
	}

	if t.currentAwaiting.Option.Type != resumeData.Message.Type || t.currentAwaiting.Option.Sequence != resumeData.Sequence {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData mismatch, current awaiting %v:%v , got %v:%v",
			action.Name(), action.GetTaskId(), t.currentAwaiting.Option.Type, t.currentAwaiting.Option.Sequence,
			resumeData.Message.Type, resumeData.Sequence,
		)
	}

	if t.currentAwaiting.Channel == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	select {
	case *t.currentAwaiting.Channel <- TaskActionAwaitChannelData{resume: resumeData}:
		t.currentAwaiting.Option = nil
	default:
		return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	return nil
}

func (t *TaskActionBase) TryKillAwait(action TaskActionImpl, killData *DispatcherErrorResult) error {
	t.currentAwaiting.Lock.Lock()
	defer t.currentAwaiting.Lock.Unlock()

	if killData == nil {
		return fmt.Errorf("task %s, %d TryKillAwait killData can not be nil", action.Name(), action.GetTaskId())
	}

	if t.currentAwaiting.Option == nil {
		return fmt.Errorf("task %s, %d TryKillAwait no current awaiting", action.Name(), action.GetTaskId())
	}

	if t.currentAwaiting.Channel == nil {
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	select {
	case *t.currentAwaiting.Channel <- TaskActionAwaitChannelData{killed: killData}:
		t.currentAwaiting.Option = nil
	default:
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	return nil
}
