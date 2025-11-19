package atsf4g_go_robot_protocol_quest

import (
	"fmt"
	"strconv"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"google.golang.org/protobuf/proto"
)

// QuestReceiveRewardRpc 发送任务领奖请求
func QuestReceiveRewardRpc(user user_data.User, questID int32) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	return user.SendReq(makeQuestReceiveRewardMessage(user, questID))
}

func makeQuestReceiveRewardMessage(user user_data.User, questID int32) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: []int32{questID},
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.quest_receive_reward", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

// QuestReceiveRewardsRpc 发送多个任务领奖请求
func QuestReceiveRewardsRpc(user user_data.User, questIDs []int32) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	return user.SendReq(makeQuestReceiveRewardsMessage(user, questIDs))
}

func makeQuestReceiveRewardsMessage(user user_data.User, questIDs []int32) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: questIDs,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.quest_receive_reward", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
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
