// Copyright 2025 atframework

package lobbysvr_logic_building_action


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionBuildingMove struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSBuildingMoveReq, *service_protocol.CSBuildingMoveRsp]
}

func (t *TaskActionBuildingMove) Name() string {
	return "TaskActionBuildingMove"
}

func (t *TaskActionBuildingMove) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

