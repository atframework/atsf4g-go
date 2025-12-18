// Copyright 2025 atframework

package lobbysvr_logic_mall_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mall "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mall"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMallPurchase struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSMallPurchaseReq, *service_protocol.SCMallPurchaseRsp]
}

func (t *TaskActionMallPurchase) Name() string {
	return "TaskActionMallPurchase"
}

func (t *TaskActionMallPurchase) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	manager := data.UserGetModuleManager[logic_mall.UserMallManager](user)
	if manager == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return fmt.Errorf("user mall manager not found")
	}
	t.SetResponseCode(manager.MallPurchase(t.GetRpcContext(), request_body.GetProductId(), request_body.GetSortId(), request_body.GetExpectCostItems(), response_body))

	return nil
}
