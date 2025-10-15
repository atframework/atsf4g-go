// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionUserGetInfo struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSUserGetInfoReq, *service_protocol.SCUserGetInfoRsp]
}

func (t *TaskActionUserGetInfo) Name() string {
	return "TaskActionUserGetInfo"
}

func (t *TaskActionUserGetInfo) Run(_startData *component_dispatcher.DispatcherStartData) error {
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	if request_body.GetNeedUserInfo() {
		response_body.UserInfo = &public_protocol_pbdesc.DUserInfo{
			Profile:   user.GetAccountInfo().GetProfile(),
			UserLevel: user.GetUserData().GetUserLevel(),
			UserStat: &public_protocol_pbdesc.DUserStat{
				RegisterTime:  user.GetLoginInfo().GetBusinessRegisterTime(),
				LastLoginTime: user.GetLoginInfo().GetBusinessLoginTime(),
			},
		}
	}

	if request_body.GetNeedUserOptions() {
		response_body.UserOptions = user.GetUserOptions().GetCustomOptions()
	}

	return nil
}
