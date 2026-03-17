// Copyright 2026 atframework

package lobbysvr_logic_module_unlock_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_module_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/module_unlock"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionModuleUnlockReceivedReward struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSModuleUnlockReceivedRewardReq, *service_protocol.SCModuleUnlockReceivedRewardRsp]
}

func (t *TaskActionModuleUnlockReceivedReward) Name() string {
	return "TaskActionModuleUnlockReceivedReward"
}

func (t *TaskActionModuleUnlockReceivedReward) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	manager := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](user)
	if manager == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return fmt.Errorf("user module unlock manager not found")
	}

	if request_body == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("request body is nil")
	}

	moduleIDs := request_body.GetUnlockModules()
	if len(moduleIDs) == 0 {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("module ids is empty")
	}

	for _, moduleId := range moduleIDs {
		result, rewards := manager.ReceviedReward(t.GetRpcContext(), moduleId)
		response_body.Results = append(response_body.Results, &service_protocol.ModuleUnlockReceivedResult{
			ModuleId:    moduleId,
			Ret:         result.ResponseCode,
			RewardItems: rewards,
		})
		if result.IsError() {
			t.GetRpcContext().LogError("failed to receive module unlock reward",
				"module_id", moduleId,
				"error", result.Error,
				"response_code", result.ResponseCode,
			)
		}
	}
	return nil
}
