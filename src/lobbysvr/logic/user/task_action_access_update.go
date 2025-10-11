// Copyright 2025 atframework

package lobbysvr_logic_user


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionAccessUpdate struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSAccessUpdateReq, *service_protocol.SCAccessUpdateRsp]
}

func (t *TaskActionAccessUpdate) Name() string {
	return "TaskActionAccessUpdate"
}

func (t *TaskActionAccessUpdate) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

