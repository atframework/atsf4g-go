package atsf4g_go_robot_cmd

import (
	"strconv"

	base "github.com/atframework/atsf4g-go/robot/base"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	protocol "github.com/atframework/atsf4g-go/robot/protocol"
)

// ========================= 注册指令 =========================
func init() {
	RegisterUserCommand([]string{"quest", "receive"}, QuestReceiveRewardCmd, "<quest_id>", "领取任务奖励", nil)
	RegisterUserCommand([]string{"quest", "receiveMulti"}, QuestReceiveRewardsCmd, "<quest_id1> [quest_id2] ...", "批量领取任务奖励", nil)
	RegisterUserCommand([]string{"quest", "activate"}, QuestActivateCmd, "<activate_id>", "激活任务", nil)
}

func QuestReceiveRewardCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questID, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}

	err = action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.QuestReceiveRewardRpc(task, int32(questID))
		return rpcErr
	}, "QuestReceiveReward Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func QuestReceiveRewardsCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questIDs := make([]int32, 0, len(cmd))
	for _, qidStr := range cmd {
		questID, err := strconv.ParseInt(qidStr, 10, 32)
		if err != nil {
			return err.Error()
		}
		questIDs = append(questIDs, int32(questID))
	}

	var err error
	err = action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.QuestReceiveRewardsRpc(task, questIDs)
		return rpcErr
	}, "QuestReceiveRewards Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}

func QuestActivateCmd(action base.TaskActionImpl, user user_data.User, cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questID, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}

	err = action.AwaitTask(user.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) error {
		_, _, rpcErr := protocol.QuestActivateRpc(task, int32(questID))
		return rpcErr
	}, "QuestActivate Task"))
	if err != nil {
		return err.Error()
	}
	return ""
}
