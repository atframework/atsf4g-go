// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"
	"strconv"
	"strings"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
)

type TaskActionUserSendGmCommand struct {
	*component_dispatcher.TaskActionCSBase[*service_protocol.CSUserGMCommandReq, *service_protocol.SCUserGMCommandRsp]
}

func (t *TaskActionUserSendGmCommand) Name() string {
	return "TaskActionUserSendGmCommand"
}

func (t *TaskActionUserSendGmCommand) Run(_startData *component_dispatcher.DispatcherStartData) error {
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	if len(request_body.GetArgs()) == 0 {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		response_body.Reply = []string{"No command provided"}
		return nil
	}

	cmd := request_body.GetArgs()[0]
	args := request_body.GetArgs()[1:]

	cmdKey := strings.ToLower(cmd)
	handle, exists := gmCommandCallbacks[cmdKey]
	if !exists {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		response_body.Reply = []string{fmt.Sprintf("Unknown command: %s", cmd)}
		return nil
	}

	var err error
	response_body.Reply, err = handle.callback(t, user, args)
	if err != nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		if response_body.Reply == nil {
			response_body.Reply = []string{fmt.Sprintf("Error executing command: %v", err)}
		} else {
			response_body.Reply = append(response_body.Reply, fmt.Sprintf("Error executing command: %v", err))
		}
	}

	return nil
}

type (
	gmCommandCallback = func(t *TaskActionUserSendGmCommand, user *data.User, args []string) ([]string, error)
	gmCommandHandle   struct {
		parameterComment string
		description      string
		callback         gmCommandCallback
	}
)

func registerGmCommandHandle(callbacks *map[string]*gmCommandHandle, cmd string, parameterComment string, description string, callback gmCommandCallback) {
	(*callbacks)[cmd] = &gmCommandHandle{
		parameterComment: parameterComment,
		description:      description,
		callback:         callback,
	}
}

func buildCommandCallbacks() map[string]*gmCommandHandle {
	callbacks := make(map[string]*gmCommandHandle)
	registerGmCommandHandle(&callbacks, "help", "", "Show this help message", (*TaskActionUserSendGmCommand).runGMCmdHelp)
	registerGmCommandHandle(&callbacks, "add-item", "<item_id> [count=1]", "Add an item to the user's inventory", (*TaskActionUserSendGmCommand).runGMCmdItemAddItem)
	registerGmCommandHandle(&callbacks, "remove-item", "<item_id> <count> [guid=0]", "Remove an item from the user's inventory", (*TaskActionUserSendGmCommand).runGMCmdItemRemoveItem)
	registerGmCommandHandle(&callbacks, "query-quest-status", "<questID>] ", "query quest status", (*TaskActionUserSendGmCommand).runGMCmdQueryQuestStatus)

	return callbacks
}

var gmCommandCallbacks map[string]*gmCommandHandle

func init() {
	gmCommandCallbacks = buildCommandCallbacks()
}

func (t *TaskActionUserSendGmCommand) runGMCmdHelp(_user *data.User, args []string) ([]string, error) {
	ret := make([]string, 0, len(gmCommandCallbacks))

	left_length := 0
	for cmd, handle := range gmCommandCallbacks {
		len := len(cmd) + 4 + len(handle.parameterComment)
		if len > left_length {
			left_length = len
		}
		if left_length >= 100 {
			left_length = 100
			break
		}
	}

	for cmd, handle := range gmCommandCallbacks {
		cmd := fmt.Sprintf("%s    %s", cmd, handle.parameterComment)
		cmdSuffix := ""
		if len(cmd) > left_length {
			cmdSuffix = strings.Repeat(" ", left_length-len(cmd))
		}
		ret = append(ret, fmt.Sprintf("%s%s    %s", cmd, cmdSuffix, handle.description))
	}

	return ret, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdItemAddItem(user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for add-item command")
	}

	itemId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid item_id: %v", err)
	}

	var count int64 = 1
	if len(args) >= 2 {
		var err error
		count, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count: %v", err)
		}
	}

	itemInstance, result := user.GenerateItemInstanceFromOffset(t.GetRpcContext(), &public_protocol_common.DItemOffset{
		TypeId: int32(itemId),
		Count:  count,
	})

	if result.IsError() {
		t.SetResponseCode(result.GetResponseCode())
		return nil, result.Error
	}

	guard, result := user.CheckAddItem(t.GetRpcContext(), []*public_protocol_common.DItemInstance{itemInstance})
	if result.IsError() {
		t.SetResponseCode(result.GetResponseCode())
		return nil, result.Error
	}

	addResult := user.AddItem(t.GetRpcContext(), guard, &data.ItemFlowReason{
		// TODO: 道具流水原因
	})
	if addResult.IsError() {
		t.SetResponseCode(addResult.GetResponseCode())
		return nil, addResult.Error
	}

	return []string{fmt.Sprintf("Add item success item_id=%d count=%d", itemId, count)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdItemRemoveItem(user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for remove-item command")
	}

	itemId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid item_id: %v", err)
	}

	var count int64 = 1
	if len(args) >= 2 {
		var err error
		count, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count: %v", err)
		}
	}

	var guid int64 = 0
	if len(args) >= 3 {
		var err error
		guid, err = strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid guid: %v", err)
		}
	}

	guard, result := user.CheckSubItem(t.GetRpcContext(), []*public_protocol_common.DItemBasic{
		{
			TypeId: int32(itemId),
			Guid:   guid,
			Count:  count,
		},
	})
	if result.IsError() {
		t.SetResponseCode(result.GetResponseCode())
		return nil, result.Error
	}

	addResult := user.SubItem(t.GetRpcContext(), guard, &data.ItemFlowReason{
		// TODO: 道具流水原因
	})
	if addResult.IsError() {
		t.SetResponseCode(addResult.GetResponseCode())
		return nil, addResult.Error
	}

	return []string{fmt.Sprintf("Sub item success item_id=%d count=%d", itemId, count)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdQueryQuestStatus(user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for query-quest-status command")
	}

	questID, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid query_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user quest manager not found")
	}

	result := mgr.QueryQuestStatus(int32(questID))

	return []string{fmt.Sprintf("Add item success status=%d", result)}, nil
}
