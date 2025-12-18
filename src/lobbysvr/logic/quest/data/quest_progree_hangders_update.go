package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

// 通用模板：从参数中提取值并更新进度
func updateProgressFromParams(
	valueGetter func(*private_protocol_pbdesc.QuestTriggerParams) int64,
) updateProgressWithCountTypeFunc {
	return func(_ cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
		_ *public_protocol_config.Readonly_DQuestConditionProgress,
		questData *public_protocol_pbdesc.DUserQuestProgressData,
		countType public_protocol_common.EnQuestProgressCountType) cd.RpcResult {
		if questData == nil {
			return cd.CreateRpcResultOk()
		}
		questData.Value = calcValeWithCountType(
			countType,
			questData.Value,
			valueGetter(params),
		)
		return cd.CreateRpcResultOk()
	}
}

var (
	updateProgressByPlayerLevel = updateProgressFromParams(
		func(params *private_protocol_pbdesc.QuestTriggerParams) int64 {
			return params.GetPlayerLevel().GetCurLevel()
		})

	updateProgressByHasItem = updateProgressFromParams(
		func(params *private_protocol_pbdesc.QuestTriggerParams) int64 {
			return params.GetHasItem().GetValue()
		})
)
