package atframework_component_dispatcher

import (
	"context"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	"github.com/atframework/libatapp-go"
)

type TaskActionNoMessageBase struct {
	TaskActionBase
}

func (t *TaskActionNoMessageBase) AllowNoActor() bool {
	return true
}

func CreateNoMessageTaskAction[TaskActionType TaskActionImpl](
	rd DispatcherImpl,
	ctx RpcContext,
	actor *ActorExecutor,
	createFn func(DispatcherImpl, *ActorExecutor, time.Duration) TaskActionType,
) (TaskActionType, DispatcherStartData) {
	ta := createFn(rd, actor, config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetTask().GetNomsg().GetTimeout().AsDuration())
	ta.SetImplementation(ta)
	awaitableContext := rd.CreateAwaitableContext()
	if !lu.IsNil(ctx) && ctx.GetContext() != nil {
		awaitableContext.SetContextCancelFn(context.WithCancel(ctx.GetContext()))
	} else {
		awaitableContext.SetContextCancelFn(context.WithCancel(context.Background()))
	}
	awaitableContext.BindAction(ta)
	libatapp.AtappGetModule[*TaskManager](rd.GetApp()).InsertTaskAction(awaitableContext, ta)

	return ta, DispatcherStartData{
		Message:           nil,
		PrivateData:       nil,
		MessageRpcContext: awaitableContext,
	}
}

func CreateNoMessageTaskActionBase(rd DispatcherImpl, actor *ActorExecutor, timeout time.Duration) TaskActionNoMessageBase {
	return TaskActionNoMessageBase{
		TaskActionBase: CreateTaskActionBase(rd, actor, timeout),
	}
}
