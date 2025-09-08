package atframework_component_dispatcher

type TaskActionImpl interface {
	Name() string
	GetTaskId() uint64

	Run(DispatcherStartData) error
	GetDispatcherTargetObject() *DispatcherTargetObject
	GetDispatcher() DispatcherImpl
	GetTypeName() string

	OnSuccess()
	OnFailed()
	OnTimeout()
	OnComplete()

	GetTraceInheritOption() *TraceInheritOption
	GetTraceStartOption() *TraceStartOption
}
