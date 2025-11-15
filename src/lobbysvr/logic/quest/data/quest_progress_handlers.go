package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type UpdateProgressConditionFunc = func(ctx *cd.RpcContext, params TriggerParams,
	progressCfg *public_protocol_config.DQuestConditionProgress, questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult

type InitProgressConditionFunc = func(ctx *cd.RpcContext, progressCfg *public_protocol_config.DQuestConditionProgress, questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult

type QuestProgressStruct struct {
	UpdateHandler UpdateProgressConditionFunc
	InitHandler   InitProgressConditionFunc
}

func buildQuestProgressCheckers() map[public_protocol_common.EnQuestProgressType]*QuestProgressStruct {
	ret := map[public_protocol_common.EnQuestProgressType]*QuestProgressStruct{}
	return ret
}

var conditionRuleCheckers = buildQuestProgressCheckers()

func initQuestProgressHandler() {
	addHandler(public_protocol_common.EnQuestProgressType_EN_QUEST_PROGRESS_TYPE_PLAYER_LEVEL, updateProgressByPlayerLevel, initProgressByPlayerLevel)
}

func addHandler(progressType public_protocol_common.EnQuestProgressType, update_f UpdateProgressConditionFunc, init_f InitProgressConditionFunc) {
	conditionRuleCheckers[progressType].UpdateHandler = update_f
	conditionRuleCheckers[progressType].InitHandler = init_f
}

func GetQuestProgressHandler(progressType public_protocol_common.EnQuestProgressType) QuestProgressStruct {
	if len(conditionRuleCheckers) == 0 {
		initQuestProgressHandler()
	}
	return *conditionRuleCheckers[progressType]
}

func updateProgressByPlayerLevel(ctx *cd.RpcContext, params TriggerParams,
	progressCfg *public_protocol_config.DQuestConditionProgress, questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult {
	// TODO: 从配置中获取所需的玩家等级
	return cd.CreateRpcResultOk()
}

func initProgressByPlayerLevel(ctx *cd.RpcContext, progressCfg *public_protocol_config.DQuestConditionProgress, questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult {
	// TODO: 从配置中获取所需的玩家等级
	return cd.CreateRpcResultOk()
}
