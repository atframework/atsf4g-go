// Copyright 2025 atframework

package lobbysvr_logic_activate_action

import (
	"fmt"

	config "github.com/atframework/atsf4g-go/component-config"
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionUserActivate struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSUserActivateReq, *service_protocol.SCUserActivateRsp]
}

func (t *TaskActionUserActivate) Name() string {
	return "TaskActionUserActivate"
}

func (t *TaskActionUserActivate) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody() // TODO
	// response_body := t.MutableResponseBody() // TODO

	activateCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelActivateById(request_body.ActivateId)
	if activateCfg == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("activateCfg is nil, activateId: %d", request_body.ActivateId)
	}

	// 条件检查
	if logic_condition.HasLimitData(activateCfg.GetCommonCondition()) {
		conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](user)
		if conditionMgr == nil {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
			return fmt.Errorf("conditionMgr is nil")
		}
		ok := conditionMgr.CheckBasicLimit(t.GetRpcContext(),
			activateCfg.GetCommonCondition(), logic_condition.CreateEmptyRuleCheckerRuntime())
		if !ok.IsOK() {
			t.SetResponseCode(ok.GetResponseCode())
			return fmt.Errorf("activate conditions not met, activateId: %d", request_body.ActivateId)
		}
	}

	switch activateCfg.GetActivateEvent().GetEventOneofCase() {
	case public_protocol_common.DActivate_EnEventID_QuestId:
		// 激活任务
		questMgr := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
		if questMgr == nil {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
			return fmt.Errorf("UserQuestManager is nil")
		}

		activateResult := questMgr.ActivateQuest(t.GetRpcContext(), activateCfg.GetActivateEvent().GetQuestId())
		if !activateResult.IsOK() {
			t.SetResponseCode(activateResult.GetResponseCode())
			return fmt.Errorf("activate quest failed, quest_id: %d", activateCfg.GetActivateEvent().GetQuestId())
		}
	default:
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("unsupported activate type: %d", activateCfg.GetActivateEvent().GetEventOneofCase())
	}
	return nil
}
