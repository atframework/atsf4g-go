package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

// func getProgressKeyByLevelRange(_ cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams,
// 	index UserProgreesKeyIndex) map[int32]*ProgressKey {

// 	result := make(map[int32]*ProgressKey)

// 	for i := params.GetPlayerLevel().GetPreLevel() + 1; i <= params.GetPlayerLevel().GetCurLevel(); i++ {
// 		if index[public_protocol_config.DQuestConditionProgress_EnProgressParamID_PlayerLevelReach] == nil {
// 			break
// 		}
// 		if index[public_protocol_config.DQuestConditionProgress_EnProgressParamID_PlayerLevelReach][int32(i)] == nil {
// 			continue
// 		}
// 		for _, data := range index[public_protocol_config.DQuestConditionProgress_EnProgressParamID_PlayerLevelReach][int32(i)] {
// 			result[data.QuestID] = data
// 		}
// 	}
// 	return result
// }

func getProgressKeyDefault(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams,
	index UserProgreesKeyIndex) map[int32]*ProgressKey {
	return index[public_protocol_config.DQuestConditionProgress_EnProgressParamID(progressType)][0]
}

func getProgressKeyByHasItem(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams,
	index UserProgreesKeyIndex) map[int32]*ProgressKey {

	return index[public_protocol_config.DQuestConditionProgress_EnProgressParamID_ItemHas][params.GetHasItem().GetKey()]
}
