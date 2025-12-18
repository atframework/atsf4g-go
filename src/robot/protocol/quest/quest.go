package atsf4g_go_robot_protocol_quest

import (
	"strconv"

	base "github.com/atframework/atsf4g-go/robot/base"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

// QuestReceiveRewardRpc 发送任务领奖请求
func QuestReceiveRewardRpc(action *user_data.TaskActionUser, questID int32) (int32, *lobysvr_protocol_pbdesc.SCQuestReceiveRewardRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: []int32{questID},
	}
	return user_data.SendQuestReceiveReward(action, csBody, true)
}

// QuestReceiveRewardsRpc 发送多个任务领奖请求
func QuestReceiveRewardsRpc(action *user_data.TaskActionUser, questIDs []int32) (int32, *lobysvr_protocol_pbdesc.SCQuestReceiveRewardRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: questIDs,
	}
	return user_data.SendQuestReceiveReward(action, csBody, true)
}

func QuestActivateRpc(action *user_data.TaskActionUser, activateID int32) (int32, *lobysvr_protocol_pbdesc.SCUserActivateRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSUserActivateReq{
		ActivateId: activateID,
	}
	return user_data.SendUserActivate(action, csBody, true)
}

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommand([]string{"quest", "receive"}, QuestReceiveRewardCmd, "<quest_id>", "领取任务奖励", nil)
	utils.RegisterCommand([]string{"quest", "receiveMulti"}, QuestReceiveRewardsCmd, "<quest_id1> [quest_id2] ...", "批量领取任务奖励", nil)
	utils.RegisterCommand([]string{"quest", "activate"}, QuestActivateCmd, "<activate_id>", "激活任务", nil)
}

func QuestReceiveRewardCmd(action base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questID, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}

	action.AwaitTask(user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		_, _, rpcErr := QuestReceiveRewardRpc(task, int32(questID))
		if rpcErr != nil {
			err = rpcErr
			return
		}
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}

func QuestReceiveRewardsCmd(action base.TaskActionImpl, cmd []string) string {
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
	action.AwaitTask(user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		_, _, rpcErr := QuestReceiveRewardsRpc(task, questIDs)
		if rpcErr != nil {
			err = rpcErr
			return
		}
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}

func QuestActivateCmd(action base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questID, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}

	action.AwaitTask(user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		_, _, rpcErr := QuestActivateRpc(task, int32(questID))
		if rpcErr != nil {
			err = rpcErr
			return
		}
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}
