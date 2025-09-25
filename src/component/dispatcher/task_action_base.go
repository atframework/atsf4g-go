package atframework_component_dispatcher

import (
	"fmt"
	"sync"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type TaskActionBase struct {
	taskId uint64

	responseCode int32

	actorExecutor *ActorExecutor
	dispatcher    DispatcherImpl

	disableResponse bool

	currentAwaitingLock    sync.Mutex
	currentAwaitingOption  *DispatcherAwaitOptions
	currentAwaitingChannel *chan TaskActionAwaitChannelData
}

func CreateTaskActionBase(rd DispatcherImpl, actorExecutor *ActorExecutor) TaskActionBase {
	return TaskActionBase{
		taskId:          rd.AllocSequence(),
		responseCode:    0,
		actorExecutor:   actorExecutor,
		dispatcher:      rd,
		disableResponse: false,
		// currentAwaitingLock:    sync.Mutex{},
		currentAwaitingOption:  nil,
		currentAwaitingChannel: nil,
	}
}

func (t *TaskActionBase) GetTaskId() uint64 {
	return t.taskId
}

func (t *TaskActionBase) CheckPermission(action TaskActionImpl) (int32, error) {
	if !action.AllowNoActor() && action.GetActorExecutor() == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_LOGIN), nil
	}
	return 0, nil
}

func (t *TaskActionBase) HookRun(action TaskActionImpl, startData *DispatcherStartData) error {
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

func (t *TaskActionBase) GetTraceInheritOption(_action TaskActionImpl) *TraceInheritOption {
	return &TraceInheritOption{}
}

func (t *TaskActionBase) GetTraceStartOption(_action TaskActionImpl) *TraceStartOption {
	return &TraceStartOption{}
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

	if t.currentAwaitingOption == awaitOptions {
		return nil
	}

	if t.currentAwaitingOption != nil {
		if t.currentAwaitingOption.Type == awaitOptions.Type && t.currentAwaitingOption.Sequence == awaitOptions.Sequence {
			// 相同等待选项，直接复用
			return nil
		}

		return fmt.Errorf("task %s, %d TrySetupAwait awaitOptions failed, already awaiting %v:%v , can not await %v:%v again",
			action.Name(), action.GetTaskId(), t.currentAwaitingOption.Type, t.currentAwaitingOption.Sequence,
			awaitOptions.Type, awaitOptions.Sequence,
		)
	}

	return nil
}

func (t *TaskActionBase) TrySetupAwait(action TaskActionImpl, awaitOptions *DispatcherAwaitOptions) (*chan TaskActionAwaitChannelData, error) {
	t.currentAwaitingLock.Lock()
	defer t.currentAwaitingLock.Unlock()

	err := t.trySetAwait(action, awaitOptions)
	if err != nil {
		return nil, err
	}

	if t.currentAwaitingChannel == nil {
		ch := make(chan TaskActionAwaitChannelData, 1)
		t.currentAwaitingChannel = &ch
	}

	return t.currentAwaitingChannel, nil
}

func (t *TaskActionBase) TryFinishAwait(action TaskActionImpl, resumeData *DispatcherResumeData) error {
	t.currentAwaitingLock.Lock()
	defer t.currentAwaitingLock.Unlock()

	if resumeData == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData can not be nil", action.Name(), action.GetTaskId())
	}

	if t.currentAwaitingOption == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait no current awaiting", action.Name(), action.GetTaskId())
	}

	if t.currentAwaitingOption.Type != resumeData.Message.Type || t.currentAwaitingOption.Sequence != resumeData.Sequence {
		return fmt.Errorf("task %s, %d TryFinishAwait resumeData mismatch, current awaiting %v:%v , got %v:%v",
			action.Name(), action.GetTaskId(), t.currentAwaitingOption.Type, t.currentAwaitingOption.Sequence,
			resumeData.Message.Type, resumeData.Sequence,
		)
	}

	if t.currentAwaitingChannel == nil {
		return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	select {
	case *t.currentAwaitingChannel <- TaskActionAwaitChannelData{resume: resumeData}:
		t.currentAwaitingOption = nil
	default:
		return fmt.Errorf("task %s, %d TryFinishAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	return nil
}

func (t *TaskActionBase) TryKillAwait(action TaskActionImpl, killData *DispatcherKillData) error {
	t.currentAwaitingLock.Lock()
	defer t.currentAwaitingLock.Unlock()

	if killData == nil {
		return fmt.Errorf("task %s, %d TryKillAwait killData can not be nil", action.Name(), action.GetTaskId())
	}

	if t.currentAwaitingOption == nil {
		return fmt.Errorf("task %s, %d TryKillAwait no current awaiting", action.Name(), action.GetTaskId())
	}

	if t.currentAwaitingChannel == nil {
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	select {
	case *t.currentAwaitingChannel <- TaskActionAwaitChannelData{killed: killData}:
		t.currentAwaitingOption = nil
	default:
		return fmt.Errorf("task %s, %d TryKillAwait send to channel failed, no receiver", action.Name(), action.GetTaskId())
	}

	return nil
}
