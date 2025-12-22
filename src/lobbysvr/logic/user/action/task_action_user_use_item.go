// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionUserUseItem struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSUserUseItemReq, *service_protocol.SCUserUseItemRsp]
}

func (t *TaskActionUserUseItem) Name() string {
	return "TaskActionUserUseItem"
}

func (t *TaskActionUserUseItem) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()
	gainItem, reuslt := user.UseItem(t.GetRpcContext(), request_body.GetItem(), request_body.GetUseParam(), &data.ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_USER),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_USER_USE_ITEM),
		Parameter:   int64(request_body.GetItem().GetTypeId()),
	})
	if reuslt.IsError() {
		t.SetResponseCode(reuslt.GetResponseCode())
		return nil
	}
	response_body.GainItem = gainItem

	return nil
}
