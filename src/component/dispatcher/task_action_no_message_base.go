package atframework_component_dispatcher

import (
	"context"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
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
	createFn func(*TaskActionNoMessageBase) TaskActionType,
) (TaskActionType, DispatcherStartData) {
	ta := createFn(&TaskActionNoMessageBase{
		TaskActionBase: CreateTaskActionBase(rd, actor, config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetTask().GetNomsg().GetTimeout().AsDuration()),
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
