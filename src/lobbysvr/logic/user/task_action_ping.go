// Copyright 2025 atframework

package lobbysvr_logic_user


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
)

type TaskActionPing struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSPingReq, *service_protocol.SCPongRsp]
}

func (t *TaskActionPing) Name() string {
	return "TaskActionPing"
}

func (t *TaskActionPing) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

