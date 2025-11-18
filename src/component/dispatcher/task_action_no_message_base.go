package atframework_component_dispatcher

import (
	"context"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

type TaskActionNoMessageBase struct {
	TaskActionBase
}

func (t *TaskActionNoMessageBase) AllowNoActor() bool {
	return true
}

func CreateTaskActionNoMessageBase[TaskActionType TaskActionImpl](
	rd DispatcherImpl,
	ctx RpcContext,
	actor *ActorExecutor,
	createFn func(TaskActionNoMessageBase) TaskActionType,
) (TaskActionType, DispatcherStartData) {
	ta := createFn(TaskActionNoMessageBase{
		TaskActionBase: CreateTaskActionBase(rd, actor),
	})
	awaitableContext := rd.CreateAwaitableContext()
	if !lu.IsNil(ctx) && ctx.GetContext() != nil {
		awaitableContext.SetContextCancelFn(context.WithCancel(ctx.GetContext()))
	}
	awaitableContext.BindAction(ta)

	return ta, DispatcherStartData{
		Message:           nil,
		PrivateData:       nil,
		MessageRpcContext: awaitableContext,
	}
}
