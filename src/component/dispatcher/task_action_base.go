package atframework_component_dispatcher

type TaskActionBase struct {
	taskId uint64

	dispatcherTargetObject *DispatcherTargetObject
	dispatcher             DispatcherImpl
	typeName               string

	traceInheritOption TraceInheritOption
	traceStartOption   TraceStartOption
}

func (t *TaskActionBase) GetTaskId() uint64 {
	return t.taskId
}

func (t *TaskActionBase) GetDispatcherTargetObject() *DispatcherTargetObject {
	return t.dispatcherTargetObject
}

func (t *TaskActionBase) GetDispatcher() DispatcherImpl {
	return t.dispatcher
}

func (t *TaskActionBase) GetTypeName() string {
	return t.typeName
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
