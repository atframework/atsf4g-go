// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_inventory "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory"
	logic_mall "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mall"
	logic_module_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/module_unlock"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
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
		response_body.UserProfile = user.GetAccountInfo().GetProfile()
		ubmgr := data.UserGetModuleManager[logic_user.UserBasicManager](user)
		if ubmgr != nil {
			response_body.UserInfo = ubmgr.DumpUserInfo()
		} else {
			t.LogError("UserBasicManager not found")
		}
	}

	if request_body.GetNeedUserOptions() {
		ubmgr := data.UserGetModuleManager[logic_user.UserBasicManager](user)
		if ubmgr != nil {
			response_body.UserOptions = ubmgr.DumpUserOptions()
		} else {
			t.LogError("UserBasicManager not found")
		}
	}

	if request_body.GetNeedUserInventory() {
		inventoryMgr := data.UserGetModuleManager[logic_inventory.UserInventoryManager](user)
		if inventoryMgr == nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return fmt.Errorf("user inventory manager not found")
		}

		inventoryMgr.ForeachItem(func(item *public_protocol_common.DItemInstance) bool {
			if item == nil {
				return true
			}

			response_body.MutableNormalizeItem().AppendItem(item)
			return true
		})
	}
	if request_body.GetNeedUserQuest() {
		questManager := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
		if questManager == nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return fmt.Errorf("user quest manager not found")
		}
		questManager.DumpQuestInfo(response_body.MutableUserQuest())
	}

	if request_body.GetNeedUserConditionCounter() {
		conditionManager := data.UserGetModuleManager[logic_condition.UserConditionManager](user)
		if conditionManager == nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return fmt.Errorf("user condition manager not found")
		}
		conditionManager.DumpConditionCounterData(response_body.MutableUserConditionCounter())
	}

	if request_body.GetNeedUserModuleUnlock() {
		moduleUnlockManager := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](user)
		if moduleUnlockManager == nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return fmt.Errorf("user module unlock manager not found")
		}
		moduleUnlockManager.DumpModuleUnlockData(response_body.MutableUserModuleUnlock())
	}

	if request_body.GetNeedUserMall() {
		mallMgr := data.UserGetModuleManager[logic_mall.UserMallManager](user)
		if mallMgr == nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return fmt.Errorf("user mall manager not found")
		}
		response_body.UserMall = mallMgr.FetchData()
	}

	return nil
}
