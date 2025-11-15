package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

type UpdateProgressConditionFunc = func(ctx *cd.RpcContext, params TriggerParams,
	progressCfg *public_protocol_config.DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult

type InitProgressConditionFunc = func(ctx *cd.RpcContext,
	progressCfg *public_protocol_config.DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData, owner *data.User) cd.RpcResult

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
	addHandler(public_protocol_common.EnQuestProgressType_EN_QUEST_PROGRESS_TYPE_PLAYER_LEVEL,
		updateProgressByPlayerLevel, initProgressByPlayerLevel)
}

func addHandler(progressType public_protocol_common.EnQuestProgressType,
	updateF UpdateProgressConditionFunc, initF InitProgressConditionFunc) {
	conditionRuleCheckers[progressType].UpdateHandler = updateF
	conditionRuleCheckers[progressType].InitHandler = initF
}

func GetQuestProgressHandler(progressType public_protocol_common.EnQuestProgressType) QuestProgressStruct {
	if len(conditionRuleCheckers) == 0 {
		initQuestProgressHandler()
	}
	return *conditionRuleCheckers[progressType]
}

func updateProgressByPlayerLevel(_ *cd.RpcContext, params TriggerParams,
	_ *public_protocol_config.DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult {
	// params.Y 是玩家当前等级，更新任务进度为当前等级
	if questData == nil {
		return cd.CreateRpcResultOk()
	}

	questData.Value = params.Y

	return cd.CreateRpcResultOk()
}

func initProgressByPlayerLevel(_ *cd.RpcContext,
	_ *public_protocol_config.DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData, owner *data.User) cd.RpcResult {
	// 获取玩家当前等级，初始化任务进度
	if questData == nil {
		return cd.CreateRpcResultOk()
	}

	if owner == nil {
		questData.Value = 0
		return cd.CreateRpcResultOk()
	}

	// 获取玩家当前等级
	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](owner)
	if mgr == nil {
		questData.Value = 0
		return cd.CreateRpcResultOk()
	}

	userLevel := mgr.GetUserLevel()
	questData.Value = int64(userLevel)

	return cd.CreateRpcResultOk()
}
