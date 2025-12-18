package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// 内部使用的带 countType 的 update 函数签名
type updateProgressWithCountTypeFunc = func(ctx cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
	countType public_protocol_common.EnQuestProgressCountType) cd.RpcResult

type UpdateProgressConditionFunc = func(ctx cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData) cd.RpcResult

type InitProgressConditionFunc = func(ctx cd.RpcContext,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData, owner *data.User) cd.RpcResult

type GetProgressKeyFunc = func(ctx cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams, index UserProgreesKeyIndex) map[int32]*ProgressKey

type QuestProgressStruct struct {
	UpdateHandler      UpdateProgressConditionFunc
	InitHandler        InitProgressConditionFunc
	ProgreesKeyHandler GetProgressKeyFunc
}

func buildQuestProgressCheckers() map[int32]*QuestProgressStruct {
	ret := map[int32]*QuestProgressStruct{}
	return ret
}

var conditionRuleCheckers = buildQuestProgressCheckers()

func addHandlerWithCountType(progressType int32,
	updateF updateProgressWithCountTypeFunc, initF InitProgressConditionFunc, getKeyF GetProgressKeyFunc,
	countType public_protocol_common.EnQuestProgressCountType) {
	if conditionRuleCheckers[progressType] == nil {
		conditionRuleCheckers[progressType] = &QuestProgressStruct{}
	}

	// 创建闭包，将 countType 绑定到 updateF
	conditionRuleCheckers[progressType].UpdateHandler = func(ctx cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
		progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
		questData *public_protocol_pbdesc.DUserQuestProgressData) cd.RpcResult {
		return updateF(ctx, params, progressCfg, questData, countType)
	}
	conditionRuleCheckers[progressType].InitHandler = initF
	conditionRuleCheckers[progressType].ProgreesKeyHandler = getKeyF
}

func GetQuestProgressHandler(progressType int32) *QuestProgressStruct {
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

func initQuestProgressHandler() {
	// 玩家等级达成
	addHandlerWithCountType(int32(public_protocol_config.DQuestConditionProgress_EnProgressParamID_PlayerLevelReach),
		updateProgressByPlayerLevel, initProgressByPlayerLevel, getProgressKeyDefault,
		public_protocol_common.EnQuestProgressCountType_EN_QUEST_PROGRESS_COUNT_TYPE_UNDEFINE)

	// 拥有物品
	addHandlerWithCountType(int32(public_protocol_config.DQuestConditionProgress_EnProgressParamID_ItemHas),
		updateProgressByHasItem, initProgressByHasItem, getProgressKeyByHasItem,
		public_protocol_common.EnQuestProgressCountType_EN_QUEST_PROGRESS_COUNT_TYPE_ADD_UP)
}
