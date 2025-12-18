// Copyright 2025 atframework

package lobbysvr_logic_quest_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionQuestUpdateStatus struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSQuestUpdateStatusReq, *service_protocol.SCQuestUpdateStatusRsp]
}

func (t *TaskActionQuestUpdateStatus) Name() string {
	return "TaskActionQuestUpdateStatus"
}

func (t *TaskActionQuestUpdateStatus) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	requesBody := t.GetRequestBody()
	// response_body := t.MutableResponseBody()

	manager := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
	if manager == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return fmt.Errorf("user quest manager not found")
	}

	if requesBody == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("request body is nil")
	}
	
	manager.ClientQueryQuestUpdateStatus(t.GetRpcContext())

	return nil
}
