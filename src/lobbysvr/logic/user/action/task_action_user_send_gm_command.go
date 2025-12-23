// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	db "github.com/atframework/atsf4g-go/component-db"
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	logical_time "github.com/atframework/atsf4g-go/component-logical_time"
	logic_module_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/module_unlock"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

type TaskActionUserSendGmCommand struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSUserGMCommandReq, *service_protocol.SCUserGMCommandRsp]
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
	response_body.Reply, err = handle.callback(t, t.GetAwaitableContext(), user, args)
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
	gmCommandCallback = func(t *TaskActionUserSendGmCommand, ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error)
	gmCommandHandle   struct {
		parameterComment string
		description      string
		callback         gmCommandCallback
	}
)

func registerGmCommandHandle(callbacks map[string]*gmCommandHandle, cmd string, parameterComment string, description string, callback gmCommandCallback) {
	callbacks[cmd] = &gmCommandHandle{
		parameterComment: parameterComment,
		description:      description,
		callback:         callback,
	}
}

func buildCommandCallbacks() map[string]*gmCommandHandle {
	callbacks := make(map[string]*gmCommandHandle)
	registerGmCommandHandle(callbacks, "help", "", "Show this help message", (*TaskActionUserSendGmCommand).runGMCmdHelp)
	registerGmCommandHandle(callbacks, "add-item", "<item_id> [count=1]", "Add an item to the user's inventory", (*TaskActionUserSendGmCommand).runGMCmdItemAddItem)
	registerGmCommandHandle(callbacks, "remove-item", "<item_id> <count> [guid=0]", "Remove an item from the user's inventory", (*TaskActionUserSendGmCommand).runGMCmdItemRemoveItem)
	registerGmCommandHandle(callbacks, "set-user-exp", "<exp> ", "Set exp and reset user level", (*TaskActionUserSendGmCommand).runGMCmdUserSetExp)
	registerGmCommandHandle(callbacks, "run-user-code", "<module_name> <func@args1@args2...>, splite by '@'] ", "Run user code", (*TaskActionUserSendGmCommand).runGMCmdUserRunCode)
	registerGmCommandHandle(callbacks, "run-user-code-byctx", "<module_name> <func@args1@args2...>, splite by '@'] ", "Run user code by ctx", (*TaskActionUserSendGmCommand).runGMCmdUserByCtxRunCode)
	registerGmCommandHandle(callbacks, "quest-query-status", "<questID>", "query quest status", (*TaskActionUserSendGmCommand).runGMCmdQueryQuestStatus)
	registerGmCommandHandle(callbacks, "quest-received-reward", "<questID>", "query received reward", (*TaskActionUserSendGmCommand).runGMCmdReceivedQuestReward)
	registerGmCommandHandle(callbacks, "quest-force-unlock", "<questID>", "query force unlock", (*TaskActionUserSendGmCommand).runGMCmdQuestForceUnlock)
	registerGmCommandHandle(callbacks, "quest-force-finish", "<questID>", "query force finish", (*TaskActionUserSendGmCommand).runGMCmdQuestForceFinish)
	registerGmCommandHandle(callbacks, "set-server-time", "YYYY-MM-DD hh:mm:ss", "Set server time to specific date and time", (*TaskActionUserSendGmCommand).runGMCmdSetServerTime)
	registerGmCommandHandle(callbacks, "show-server-time", "", "Get current server time offset", (*TaskActionUserSendGmCommand).runGMCmdShowServerTime)
	registerGmCommandHandle(callbacks, "reset-server-time", "", "Reset server time to system time", (*TaskActionUserSendGmCommand).runGMCmdResetServerTime)
	registerGmCommandHandle(callbacks, "unlock-all-modules", "", "unlock all modules", (*TaskActionUserSendGmCommand).runGMCmdUnlockAllModules)
	registerGmCommandHandle(callbacks, "unlock-module", "<module_id>", "unlock special module", (*TaskActionUserSendGmCommand).runGMCmdUnlockModule)
	registerGmCommandHandle(callbacks, "query-module-status", "<module_id>", "query special module", (*TaskActionUserSendGmCommand).runGMCmdQueryModuleStatus)
	registerGmCommandHandle(callbacks, "del-account", "", "删除账号", (*TaskActionUserSendGmCommand).runGMCmdDelAccount)
	registerGmCommandHandle(callbacks, "copy-account", "<new_account_id>", "拷贝账号", (*TaskActionUserSendGmCommand).runGMCmdCopyAccount)

	return callbacks
}

var gmCommandCallbacks map[string]*gmCommandHandle

func init() {
	gmCommandCallbacks = buildCommandCallbacks()
}

func (t *TaskActionUserSendGmCommand) runGMCmdHelp(ctx component_dispatcher.AwaitableContext, _user *data.User, args []string) ([]string, error) {
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

func (t *TaskActionUserSendGmCommand) runGMCmdItemAddItem(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for add-item command")
	}

	itemId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid item_id: %w", err)
	}

	var count int64 = 1
	if len(args) >= 2 {
		var err error
		count, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count: %w", err)
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
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_GM),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_GM_ADD_ITEM),
		Parameter:   int64(itemId),
	})
	if addResult.IsError() {
		t.SetResponseCode(addResult.GetResponseCode())
		return nil, addResult.Error
	}

	return []string{fmt.Sprintf("Add item success item_id=%d count=%d", itemId, count)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdItemRemoveItem(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for remove-item command")
	}

	itemId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid item_id: %w", err)
	}

	var count int64 = 1
	if len(args) >= 2 {
		var err error
		count, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count: %w", err)
		}
	}

	var guid int64 = 0
	if len(args) >= 3 {
		var err error
		guid, err = strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid guid: %w", err)
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
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_GM),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_GM_SUB_ITEM),
		Parameter:   int64(itemId),
	})
	if addResult.IsError() {
		t.SetResponseCode(addResult.GetResponseCode())
		return nil, addResult.Error
	}

	return []string{fmt.Sprintf("Sub item success item_id=%d count=%d", itemId, count)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdUserSetExp(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	exp := int64(0)
	if len(args) > 0 {
		exp = 0
		var err error
		exp, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid exp: %w", err)
		}
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user basic manager not found")
	}

	result := mgr.GmResetUserExp(t.GetRpcContext(), exp)
	if result.IsError() {
		t.SetResponseCode(result.GetResponseCode())
		return nil, result.Error
	}

	return []string{fmt.Sprintf("Set user exp success exp=%d, level=%d", exp, mgr.GetUserLevel())}, nil
}

const (
	moduleName = iota
	funcArgs
	end
)

func (t *TaskActionUserSendGmCommand) runGMCmdUserRunCode(ctx component_dispatcher.AwaitableContext, _ *data.User, args []string) ([]string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in runGMCmdUserRunCode: %v\n", r)
		}
	}()

	if len(args) < end {
		return nil, fmt.Errorf("invalid arguments: expected module and code, got %d", len(args))
	}

	return t.invokeModuleMethod(args[moduleName], args[funcArgs], nil)
}

func (t *TaskActionUserSendGmCommand) runGMCmdUserByCtxRunCode(ctx component_dispatcher.AwaitableContext, _ *data.User, args []string) ([]string, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in runGMCmdUserByCtxRunCode: %v\n", r)
		}
	}()

	if len(args) < end {
		return nil, fmt.Errorf("invalid arguments: expected module and code, got %d", len(args))
	}

	return t.invokeModuleMethod(args[moduleName], args[funcArgs], t.GetRpcContext())
}

// invokeModuleMethod 通过反射调用模块方法。
// ctx 为 nil 时，方法不需要 context 参数；否则作为第一个参数传入.
func (t *TaskActionUserSendGmCommand) invokeModuleMethod(moduleName, codeString string, ctx interface{},
) ([]string, error) {
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 通过模块名获取 manager
	mgr := user.GetModuleManagerByName(moduleName)
	if mgr == nil {
		return nil, fmt.Errorf("module manager '%s' not found for user", moduleName)
	}

	// 解析代码字符串，格式: funcName@arg1@arg2@...
	parts := strings.Split(codeString, "@")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid code string format")
	}

	funcName := parts[0]
	params := parts[1:]

	// 通过反射获取方法
	method := reflect.ValueOf(mgr).MethodByName(funcName)
	if !method.IsValid() {
		return nil, fmt.Errorf(
			"method '%s' not found on manager '%s'",
			funcName,
			moduleName,
		)
	}

	// 获取方法的类型信息
	ft := method.Type()
	var callArgs []reflect.Value

	// 如果需要 context 参数，先添加
	startIdx := 0
	if ctx != nil {
		callArgs = append(callArgs, reflect.ValueOf(ctx))
		startIdx = 1
	}

	// 构建函数参数
	for i := startIdx; i < ft.NumIn(); i++ {
		pt := ft.In(i)

		// 从 params 中获取对应参数值，不存在则使用空字符串
		paramIdx := i - startIdx
		var paramValue string
		if paramIdx < len(params) {
			paramValue = params[paramIdx]
		}

		// 使用反射工具转换参数类型
		pv, err := lu.AssignValue(pt, paramValue)
		if err != nil {
			return nil, fmt.Errorf("parameter %d conversion failed: %w", i, err)
		}
		callArgs = append(callArgs, pv)
	}

	// 调用方法
	returnValues := method.Call(callArgs)

	// 处理返回值
	result := lu.FormatValues(returnValues)
	if len(result) == 0 {
		result = append(
			result,
			fmt.Sprintf(
				"Method '%s' on module '%s' executed successfully",
				funcName,
				moduleName,
			),
		)
	}

	return result, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdQueryQuestStatus(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
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

	return []string{fmt.Sprintf("quest success status=%d", result)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdReceivedQuestReward(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for received-quest-reward command")
	}

	questID, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid quest_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user quest manager not found")
	}

	rewardItem, result := mgr.ReceivedQuestReward(ctx, int32(questID), false)

	for _, item := range rewardItem {
		fmt.Printf("Received item: TypeId=%d, Count=%d, Guid=%d\n", item.GetTypeId(), item.GetCount(), item.GetGuid())
	}

	return []string{fmt.Sprintf("received quest reward status=%d", result)}, nil
}

// ====================== GM 时间控制命令 =========================

const (
	// 时间格式：yyyy:mm:dd hh:mm:ss
	gmTimeFormat = "2006-01-02 15:04:05"
)

func (t *TaskActionUserSendGmCommand) runGMCmdSetServerTime(ctx component_dispatcher.AwaitableContext, _user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for set-server-time command, expected format: yyyy:mm:dd hh:mm:ss")
	}

	// 解析时间格式 yyyy:mm:dd hh:mm:ss
	timeStr := strings.Join(args, " ")
	// 替换格式：yyyy:mm:dd hh:mm:ss -> 2006-01-02 15:04:05
	// 只替换日期部分的冒号
	parts := strings.Split(timeStr, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid time format, expected: yyyy:mm:dd hh:mm:ss, got: %s", timeStr)
	}
	// 日期部分冒号替换为连字符
	datePart := strings.ReplaceAll(parts[0], ":", "-")
	timePart := parts[1]
	normalizedTimeStr := datePart + " " + timePart

	// 使用本地时区解析时间（避免 UTC 偏移问题）
	newTime, err := time.ParseInLocation(gmTimeFormat, normalizedTimeStr, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid time format, expected: yyyy:mm:dd hh:mm:ss, got: %s, error: %w", timeStr, err)
	}

	dispatcher := t.GetRpcContext().GetAction().GetDispatcher()
	if dispatcher == nil {
		return nil, fmt.Errorf("dispatcher not found")
	}

	currentTime := dispatcher.GetNow()

	// 校验：不允许设置过去的时间
	if newTime.Before(currentTime) {
		return nil, fmt.Errorf("cannot set time to past, current time: %s, requested time: %s",
			currentTime.Format(gmTimeFormat), newTime.Format(gmTimeFormat))
	}
	// 设置新时间
	logical_time.SetGlobalLogicalOffset(time.Until(newTime))

	// 持久化时间偏移到文件
	offset := logical_time.GetGlobalLogicalOffset()
	if err := persistTimeOffset(offset); err != nil {
		return nil, fmt.Errorf("failed to persist time offset: %w", err)
	}

	// 获取设置后的时间进行验证
	verifyTime := dispatcher.GetNow()

	return []string{
		fmt.Sprintf("Server time set to %s", newTime.Format(gmTimeFormat)),
		fmt.Sprintf("Time offset: %v", offset),
		fmt.Sprintf("Current server time: %s", verifyTime.Format(gmTimeFormat)),
	}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdShowServerTime(ctx component_dispatcher.AwaitableContext, _user *data.User, args []string) ([]string, error) {
	dispatcher := t.GetRpcContext().GetAction().GetDispatcher()
	if dispatcher == nil {
		return nil, fmt.Errorf("dispatcher not found")
	}

	offset := logical_time.GetGlobalLogicalOffset()
	nowTime := t.GetRpcContext().GetNow()

	return []string{
		fmt.Sprintf("Current server time offset: %v", offset),
		fmt.Sprintf("Current server time: %s", nowTime.Format(gmTimeFormat)),
		fmt.Sprintf("System time: %s", t.GetRpcContext().GetSysNow().Format(gmTimeFormat)),
	}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdResetServerTime(ctx component_dispatcher.AwaitableContext, _user *data.User, args []string) ([]string, error) {
	logical_time.SetGlobalLogicalOffset(0)

	// 持久化时间偏移到文件（重置为0）
	if err := persistTimeOffset(0); err != nil {
		return nil, fmt.Errorf("failed to persist time offset: %w", err)
	}

	return []string{"Server time reset to system time"}, nil
}

// persistTimeOffset 将时间偏移持久化到文件
func persistTimeOffset(offset time.Duration) error {
	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exePath := filepath.Dir(ex)
	timeOffsetPath := filepath.Join(exePath, "..", "..", "timeOffset.txt")

	timeOffsetPath, err = filepath.Abs(timeOffsetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 写入文件（单位：秒）
	content := fmt.Sprintf("%d", int64(offset.Seconds()))
	err = os.WriteFile(timeOffsetPath, []byte(content), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write time offset file %s: %w", timeOffsetPath, err)
	}

	return nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdUnlockAllModules(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	mgr := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user module unlock manager not found")
	}

	mgr.GMUnlockAllModules(t.GetRpcContext())

	return []string{"All modules unlocked"}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdUnlockModule(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for unlock-module command")
	}

	moduleId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid module_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user module unlock manager not found")
	}

	mgr.GMUnlockModule(t.GetRpcContext(), int32(moduleId))

	return []string{fmt.Sprintf("Module %d unlocked", moduleId)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdQueryModuleStatus(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for query-module-status command")
	}

	moduleId, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid module_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_module_unlock.UserModuleUnlockManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user module unlock manager not found")
	}

	unlocked := mgr.GMQueryModuleStatus(int32(moduleId))

	statusStr := "locked"
	if unlocked {
		statusStr = "unlocked"
	}

	return []string{fmt.Sprintf("Module %d is %s", moduleId, statusStr)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdQuestForceUnlock(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for received-quest-reward command")
	}

	questID, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid quest_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user quest manager not found")
	}

	result := mgr.GMForceUnlockQuest(ctx, int32(questID))

	return []string{fmt.Sprintf("received quest reward status=%d", result)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdQuestForceFinish(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for received-quest-reward command")
	}

	questID, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid quest_id: %w", err)
	}

	mgr := data.UserGetModuleManager[logic_quest.UserQuestManager](user)
	if mgr == nil {
		return nil, fmt.Errorf("user quest manager not found")
	}

	result := mgr.GMForceFinishQuest(ctx, int32(questID))

	return []string{fmt.Sprintf("received quest reward status=%d", result)}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdDelAccount(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	component_dispatcher.AsyncThen(ctx, "del account", user.GetActorExecutor(), ctx.GetAction(), func(childCtx cd.AwaitableContext) {
		libatapp.AtappGetModule[*uc.UserManager](childCtx.GetApp()).Remove(childCtx, user.GetZoneId(), user.GetUserId(), user, true)
		db.DatabaseTableAccessDelWithZoneIdUserId(childCtx, user.GetZoneId(), user.GetUserId())
		db.DatabaseTableLoginLockDelWithUserId(childCtx, user.GetUserId())
		db.DatabaseTableUserDelWithZoneIdUserId(childCtx, user.GetZoneId(), user.GetUserId())
	})
	return []string{""}, nil
}

func (t *TaskActionUserSendGmCommand) runGMCmdCopyAccount(ctx component_dispatcher.AwaitableContext, user *data.User, args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("invalid arguments for lottery-reset-pool <pool id> command")
	}

	newUserId, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid count: %w", err)
	}

	if newUserId == user.GetUserId() {
		return nil, fmt.Errorf("new user id cannot be the same as the current user id")
	}

	dstUserTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result := user.DumpToDB(ctx, dstUserTb)
	if result.IsError() {
		// 走到这会丢数据
		result.LogError(ctx, "dump user to db failed")
		return nil, fmt.Errorf("dump user to db failed")
	}

	var version uint64
	dstUserTb.UserId = newUserId
	db.DatabaseTableUserUpdateZoneIdUserId(ctx, dstUserTb, &version, true)
	copy := user.GetLoginLockInfo().Clone()
	copy.UserId = newUserId
	copy.ExpectTableUserDbVersion = 0
	db.DatabaseTableLoginLockUpdateUserId(ctx, copy, &version, true)

	return []string{""}, nil
}
