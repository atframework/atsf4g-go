// Copyright 2025 atframework

package lobbysvr_logic_action


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
)

type TaskActionLoginAuth struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSLoginAuthReq, *service_protocol.SCLoginAuthRsp]
}

func (t *TaskActionLoginAuth) Name() string {
	return "TaskActionLoginAuth"
}

func (t *TaskActionLoginAuth) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

