package atframework_component_dispatcher

type TaskActionImpl interface {
	MutableActorExecutorFromMessage(startData DispatcherStartData) *ActorExecutor

	Name() string
	GetTaskId() uint64

	Run(DispatcherStartData) error
	GetDispatcherTargetObject() *ActorExecutor
	GetDispatcher() DispatcherImpl
	GetTypeName() string

	OnSuccess()
	OnFailed()
	OnTimeout()
	OnComplete()

	GetTraceInheritOption() *TraceInheritOption
	GetTraceStartOption() *TraceStartOption
}
