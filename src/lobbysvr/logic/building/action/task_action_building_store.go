// Copyright 2025 atframework

package lobbysvr_logic_building_action


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionBuildingStore struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSBuildingStoreReq, *service_protocol.CSBuildingStoreRsp]
}

func (t *TaskActionBuildingStore) Name() string {
	return "TaskActionBuildingStore"
}

func (t *TaskActionBuildingStore) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

