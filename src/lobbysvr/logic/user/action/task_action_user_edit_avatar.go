// Copyright 2026 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionUserEditAvatar struct {
	user_controller.TaskActionCSBase[*service_protocol.CSUserEditAvatarReq, *service_protocol.SCUserEditAvatarRsp]
}

func (t *TaskActionUserEditAvatar) Name() string {
	return "TaskActionUserEditAvatar"
}

func (t *TaskActionUserEditAvatar) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](user)
	if mgr == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		return fmt.Errorf("UserBasicManager not found")
	}

	result := mgr.EditAvatar(t.GetAwaitableContext(), request_body.GetAvatar())
	if result.IsError() {
		t.SetResponseCode(result.GetResponseCode())
		return result.GetStandardError()
	}

	return nil
}
