// Copyright 2026 atframework

package lobbysvr_logic_mail_action

import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMailReceiveAttachmentsAll struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSMailReceiveAttachmentsAllReq, *service_protocol.SCMailReceiveAttachmentsAllRsp]
}

func (t *TaskActionMailReceiveAttachmentsAll) Name() string {
	return "TaskActionMailReceiveAttachmentsAll"
}

func (t *TaskActionMailReceiveAttachmentsAll) Run(_startData *component_dispatcher.DispatcherStartData) error {
	requestBody := t.GetRequestBody()
	responseBody := t.MutableResponseBody()

	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.GetRpcContext().LogError("not logined")
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_NOT_LOGINED))
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

	rpcResult := mailMgr.ReceiveMailAttachmentsAll(t.GetRpcContext(), responseBody.MutableMails(), requestBody.NeedRemove)
	if rpcResult.IsError() {
		if rpcResult.ResponseCode != 0 {
			t.SetResponseCode(rpcResult.ResponseCode)
		}
	}

	return nil
}
