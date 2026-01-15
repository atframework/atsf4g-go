package atsf4g_go_robot_protocol

import (
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	user_data "github.com/atframework/robot-go/data"
)

// QuestReceiveRewardRpc 发送任务领奖请求
func QuestReceiveRewardRpc(action *user_data.TaskActionUser, questID int32) (int32, *lobysvr_protocol_pbdesc.SCQuestReceiveRewardRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: []int32{questID},
	}
	return SendQuestReceiveReward(action, csBody, true)
}

// QuestReceiveRewardsRpc 发送多个任务领奖请求
func QuestReceiveRewardsRpc(action *user_data.TaskActionUser, questIDs []int32) (int32, *lobysvr_protocol_pbdesc.SCQuestReceiveRewardRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSQuestReceiveRewardReq{
		QuestIds: questIDs,
	}
	return SendQuestReceiveReward(action, csBody, true)
}

func QuestActivateRpc(action *user_data.TaskActionUser, activateID int32) (int32, *lobysvr_protocol_pbdesc.SCUserActivateRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSUserActivateReq{
		ActivateId: activateID,
	}
	return SendUserActivate(action, csBody, true)
}
