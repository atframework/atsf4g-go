package atframework_component_dispatcher

type taskActionAsyncInvoke struct {
	TaskActionNoMessageBase
	name     string
	callable func(ctx AwaitableContext) RpcResult
}

func (t *taskActionAsyncInvoke) Name() string {
	return t.name
}

func (t *taskActionAsyncInvoke) Run(_startData *DispatcherStartData) error {
	result := t.callable(t.GetAwaitableContext())
	t.SetResponseCode(result.GetResponseCode())
	return result.GetStandardError()
}
