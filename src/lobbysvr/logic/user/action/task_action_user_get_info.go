// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	logic_inventory "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory"

	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
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
			ubmgr.DumpUserInfo(response_body.MutableUserInfo())
		} else {
			t.LogError("UserBasicManager not found")
		}
	}

	if request_body.GetNeedUserOptions() {
		response_body.UserOptions = user.GetUserOptions().GetCustomOptions()
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

			response_body.MutableUserInventory().AppendItem(item)
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
	return nil
}
