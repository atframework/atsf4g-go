package atframework_component_dispatcher

type TaskActionBase struct {
	taskId uint64

	responseCode int32

	actorExecutor *ActorExecutor
	dispatcher    DispatcherImpl
	typeName      string

	traceInheritOption TraceInheritOption
	traceStartOption   TraceStartOption

	disableResponse bool
}

func (t *TaskActionBase) GetTaskId() uint64 {
	return t.taskId
}

func (t *TaskActionBase) HookRun(action TaskActionImpl, startData DispatcherStartData) error {
	return action.Run(startData)
}

func (t *TaskActionBase) GetActorExecutor() *ActorExecutor {
	return t.actorExecutor
}

func (t *TaskActionBase) GetDispatcher() DispatcherImpl {
	return t.dispatcher
}

func (t *TaskActionBase) GetTypeName() string {
	return t.typeName
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

func (t *TaskActionBase) GetTraceInheritOption() *TraceInheritOption {
	return &t.traceInheritOption
}

func (t *TaskActionBase) GetTraceStartOption() *TraceStartOption {
	return &t.traceStartOption
}
