package atsf4g_go_robot_protocol_quest

import (
	"fmt"
	"strconv"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

// QuestReceiveRewardRpc 发送任务领奖请求
func QuestReceiveRewardRpc(user user_data.User, questID int32) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: []int32{questID},
	}

	return user_data.SendQuestReceiveReward(user, csBody)
}

// QuestReceiveRewardsRpc 发送多个任务领奖请求
func QuestReceiveRewardsRpc(user user_data.User, questIDs []int32) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: questIDs,
	}

	return user_data.SendQuestReceiveReward(user, csBody)
}

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommand([]string{"quest", "receive"}, QuestReceiveRewardCmd, "<quest_id>", "领取任务奖励", nil)
	utils.RegisterCommand([]string{"quest", "receiveMulti"}, QuestReceiveRewardsCmd, "<quest_id1> [quest_id2] ...", "批量领取任务奖励", nil)
}

func QuestReceiveRewardCmd(cmd []string) string {
	if len(cmd) < 1 {
		return "Args Error"
	}

	questID, err := strconv.ParseInt(cmd[0], 10, 32)
	if err != nil {
		return err.Error()
	}

	err = QuestReceiveRewardRpc(user_data.GetCurrentUser(), int32(questID))
	if err != nil {
		return err.Error()
	}
	return ""
}

func QuestReceiveRewardsCmd(cmd []string) string {
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

	err := QuestReceiveRewardsRpc(user_data.GetCurrentUser(), questIDs)
	if err != nil {
		return err.Error()
	}
	return ""
}
