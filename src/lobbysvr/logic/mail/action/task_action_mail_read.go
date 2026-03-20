// Copyright 2026 atframework
// Translated from task_action_mail_read.cpp

package lobbysvr_logic_mail_action

import (
	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMailRead struct {
	user_controller.TaskActionCSBase[*service_protocol.CSMailReadReq, *service_protocol.SCMailReadRsp]
}

func (t *TaskActionMailRead) Name() string {
	return "TaskActionMailRead"
}

func (t *TaskActionMailRead) Run(_startData *component_dispatcher.DispatcherStartData) error {
	requestBody := t.GetRequestBody()
	responseBody := t.MutableResponseBody()

	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.GetRpcContext().LogError("not logined")
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_NOT_LOGINED))
		return nil
	}

	if len(requestBody.GetMailIds()) == 0 {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return nil
	}

	mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
	if mailMgr == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		return nil
	}

	result := mailMgr.WaitForAsyncTask(t.GetRpcContext())
	if result.GetResponseCode() != 0 {
		t.GetRpcContext().LogError("TaskActionMailGetAll WaitForAsyncTask failed, code:", result.GetResponseCode())
		return nil
	}

	for _, mailId := range requestBody.GetMailIds() {

		result := &public_protocol_pbdesc.DMailOperationResult{}
		res := mailMgr.ReadMail(t.GetRpcContext(), mailId, result, requestBody.GetNeedRemove())
		if res < 0 {
			t.SetResponseCode(res)
		}
		responseBody.Mails = append(responseBody.Mails, result)
	}

	return nil
}
