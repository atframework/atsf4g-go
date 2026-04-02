// Copyright 2026 atframework

package lobbysvr_logic_mail_action

import (
	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMailReceiveAttachments struct {
	user_controller.TaskActionCSBase[*service_protocol.CSMailReceiveAttachmentsReq, *service_protocol.SCMailReceiveAttachmentsRsp]
}

func (t *TaskActionMailReceiveAttachments) Name() string {
	return "TaskActionMailReceiveAttachments"
}

func (t *TaskActionMailReceiveAttachments) Run(_startData *component_dispatcher.DispatcherStartData) error {
	requestBody := t.GetRequestBody()
	responseBody := t.MutableResponseBody()

	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.GetRpcContext().LogError("not logined")
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_NOT_LOGINED))
		return nil
	}

	if len(requestBody.MailIds) == 0 {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return nil
	}

	mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
	if mailMgr == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		return nil
	}
	result := mailMgr.WaitForAsyncTask(t.GetAwaitableContext())
	if result.GetResponseCode() != 0 {
		t.GetRpcContext().LogError("TaskActionMailGetAll WaitForAsyncTask failed, code:", result.GetResponseCode())
		return nil
	}

	receivedItemReset := map[int32]int64{}

	for _, mailId := range requestBody.MailIds {
		result := &public_protocol_pbdesc.DMailOperationResult{}

		rpcResult := mailMgr.ReceiveMailAttachments(t.GetRpcContext(), mailId, result, requestBody.NeedRemove)
		if rpcResult.IsError() {
			if rpcResult.ResponseCode != 0 {
				t.SetResponseCode(rpcResult.ResponseCode)
			}
			t.LogError("received mail failed, mail id:", result.Record.GetMailId())
			continue
		}
		for _, itemOffset := range result.GetAttachments() {
			receivedItemReset[itemOffset.GetTypeId()] += itemOffset.GetCount()
		}
	}

	for typeID, count := range receivedItemReset {
		responseBody.ReceivedItems = append(responseBody.ReceivedItems, &public_protocol_common.DItemOffset{
			TypeId: typeID,
			Count:  count,
		})
	}

	return nil
}
