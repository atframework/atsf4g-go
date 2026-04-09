// Copyright 2026 atframework

package lobbysvr_logic_mall_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mall "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mall"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionMallRefreshRandomSheet struct {
	user_controller.TaskActionCSBase[*service_protocol.CSMallRefreshRandomSheetReq, *service_protocol.SCMallRefreshRandomSheetRsp]
}

func (t *TaskActionMallRefreshRandomSheet) Name() string {
	return "TaskActionMallRefreshRandomSheet"
}

func (t *TaskActionMallRefreshRandomSheet) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	// response_body := t.MutableResponseBody()

	manager := data.UserGetModuleManager[logic_mall.UserMallManager](user)
	if manager == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return fmt.Errorf("user mall manager not found")
	}

	ret := manager.RefreshMallRandomSheet(t.GetRpcContext(), request_body.GetMallSheetId(), request_body.GetExpectCostItems())
	if ret.IsError() {
		t.SetResponseCode(ret.GetResponseCode())
		return fmt.Errorf("refresh mall random sheet failed, code: %d", ret.GetResponseCode())
	}
	t.SetResponseCode(int32(ret.GetResponseCode()))

	return nil
}
