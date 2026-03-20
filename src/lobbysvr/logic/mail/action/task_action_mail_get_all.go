// Copyright 2026 atframework
// Translated from task_action_mail_get_all.cpp

package lobbysvr_logic_mail_action

import (
	"fmt"

	config "github.com/atframework/atsf4g-go/component/config"
	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	mail "github.com/atframework/atsf4g-go/component/mail"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMailGetAll struct {
	user_controller.TaskActionCSBase[*service_protocol.CSMailGetAllReq, *service_protocol.SCMailGetAllRsp]
}

func (t *TaskActionMailGetAll) Name() string {
	return "TaskActionMailGetAll"
}

func (t *TaskActionMailGetAll) Run(_startData *component_dispatcher.DispatcherStartData) error {
	requestBody := t.GetRequestBody()
	responseBody := t.MutableResponseBody()

	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.GetRpcContext().LogError("not logined")
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_NOT_LOGINED))
		return nil
	}

	if requestBody.GetMajorType() <= 0 || !config.IsValidUserMail(requestBody.GetMajorType()) {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return nil
	}

	responseBody.PageCount = requestBody.GetPageCount()
	responseBody.PageNo = requestBody.GetPageNo()
	responseBody.MajorType = requestBody.GetMajorType()

	mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
	if mailMgr == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return fmt.Errorf("user mail manager not found")
	}

	result := mailMgr.WaitForAsyncTask(t.GetRpcContext())
	if result.GetResponseCode() != 0 {
		t.GetRpcContext().LogError("TaskActionMailGetAll WaitForAsyncTask failed, code:", result.GetResponseCode())
		return nil
	}

	var skipCount int32 = 0
	if requestBody.GetPageCount() > 0 {
		if requestBody.GetPageNo() > 0 {
			skipCount = (requestBody.GetPageNo() - 1) * requestBody.GetPageCount()
		}
	}

	mailBox := mailMgr.GetMailBoxByMajorType(requestBody.GetMajorType())
	var mailCount int32 = 0

	if mailBox != nil {
		for _, mailData := range mailBox.Mails {
			if mail_data.IsMailDataShown(mailData) {
				mailCount++
				if requestBody.GetPageCount() > 0 && int32(len(responseBody.GetMails())) >= requestBody.GetPageCount() {
					continue
				}

				if skipCount > 0 {
					skipCount--
					continue
				}

				// 合并邮件内容和记录
				out := &public_protocol_pbdesc.DMailContent{}
				mail.MailMergeContentAndRecord(out, mailData.Content, mailData.Record)
				responseBody.Mails = append(responseBody.Mails, out)
			}
		}
	}

	responseBody.MailCount = mailCount

	return nil
}
