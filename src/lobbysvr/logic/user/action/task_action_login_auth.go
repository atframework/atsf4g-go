package lobbysvr_logic_user_action

import (
	"log/slog"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionLoginAuth struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSLoginAuthReq, *service_protocol.SCLoginAuthRsp]
}

func (t *TaskActionLoginAuth) Name() string {
	return "TaskActionLoginAuth"
}

func (t *TaskActionLoginAuth) AllowNoActor() bool {
	return true
}

func (t *TaskActionLoginAuth) Run(_startData *component_dispatcher.DispatcherStartData) error {
	t.GetDispatcher().GetApp().GetLogger().Info("TaskActionLoginAuth Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", t.GetSession().GetSessionId()),
	)
	// TODO: 临时信任session，后续加入token验证和User绑定
	t.GetSession().BindUser(&data.User{})
	return nil
}
