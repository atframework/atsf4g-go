package atframework_component_dispatcher

type TaskActionNoMessageBase struct {
	TaskActionBase
}

func CreateTaskActionNoMessageBase(
	rd DispatcherImpl,
	actor *ActorExecutor,
) TaskActionNoMessageBase {
	return TaskActionNoMessageBase{
		TaskActionBase: CreateTaskActionBase(rd, actor),
	}
}

func (t *TaskActionNoMessageBase) AllowNoActor() bool {
	return true
}
