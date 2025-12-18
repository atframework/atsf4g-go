package lobbysvr_logic_quest_data

import (
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

func GetProgressCfgParam1Value(progressCfg *public_protocol_config.Readonly_DQuestConditionProgress) int32 {
	switch progressCfg.GetProgressParamOneofCase() {
	case public_protocol_config.DQuestConditionProgress_EnProgressParamID_ItemHas:
		return progressCfg.GetItemHas()
	}
	return 0
}

func GetProgressCfgByUniqueId(unique_id int32, progress []*public_protocol_config.Readonly_DQuestConditionProgress) *public_protocol_config.Readonly_DQuestConditionProgress {
	for _, progressCfg := range progress {
		if progressCfg.GetUniqueId() != unique_id {
			continue
		}
		return progressCfg
	}
	return nil
}
