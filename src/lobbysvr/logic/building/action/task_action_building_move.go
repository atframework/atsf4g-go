// Copyright 2025 atframework

package lobbysvr_logic_building_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	building "github.com/atframework/atsf4g-go/service-lobbysvr/logic/building"
)

type TaskActionBuildingMove struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSBuildingMoveReq, *service_protocol.CSBuildingMoveRsp]
}

func (t *TaskActionBuildingMove) Name() string {
	return "TaskActionBuildingMove"
}

func (t *TaskActionBuildingMove) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	// reqBody := t.GetRequestBody()      // TODO
	// rspBody := t.MutableResponseBody() // TODO

	manager := data.GetModuleManager[building.UserBuildingManager](user)
	t.SetResponseCode(manager.BuildingMove())
	return nil
}
