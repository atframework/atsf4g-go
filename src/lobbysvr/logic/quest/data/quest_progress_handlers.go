package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

type UpdateProgressConditionFunc = func(ctx cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult

type InitProgressConditionFunc = func(ctx cd.RpcContext,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
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
	if conditionRuleCheckers[progressType] == nil {
		conditionRuleCheckers[progressType] = &QuestProgressStruct{}
	}
	conditionRuleCheckers[progressType].UpdateHandler = updateF
	conditionRuleCheckers[progressType].InitHandler = initF
}

func GetQuestProgressHandler(progressType public_protocol_common.EnQuestProgressType) *QuestProgressStruct {
	if len(conditionRuleCheckers) == 0 {
		initQuestProgressHandler()
	}
	handler := conditionRuleCheckers[progressType]
	if handler == nil {
		return &QuestProgressStruct{}
	}
	return handler
}

func calcValeWithCountType(countType public_protocol_common.EnQuestProgressCountType, oldValue, addValue int64) int64 {
	switch countType {
	case public_protocol_common.EnQuestProgressCountType_EN_QUEST_PROGRESS_COUNT_TYPE_SINGLE:
		return max(oldValue, addValue)
	case public_protocol_common.EnQuestProgressCountType_EN_QUEST_PROGRESS_COUNT_TYPE_ADD_UP:
		return oldValue + addValue
	default:
		return addValue
	}
}

func updateProgressByPlayerLevel(_ cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestData) cd.RpcResult {
	// params.Y 是玩家当前等级，更新任务进度为当前等级
	if questData == nil {
		return cd.CreateRpcResultOk()
	}

	questData.Value = calcValeWithCountType(
		progressCfg.GetCountType(),
		questData.Value,
		int64(params.GetPlayerLevel().GetCurLevel()),
	)
	// questData.Value =
	// params.GetPlayerLevel().GetCurLevel()

	return cd.CreateRpcResultOk()
}

func initProgressByPlayerLevel(_ cd.RpcContext,
	_ *public_protocol_config.Readonly_DQuestConditionProgress,
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
