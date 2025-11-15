package lobbysvr_logic_quest_data

import (
	"reflect"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
)

type CheckProgressFilterConditionFunc = func(ctx *cd.RpcContext, params logic_quest.TriggerParams,
	progress_config *public_protocol_config.DQuestConditionProgress, user_quest_data *public_protocol_pbdesc.DUserQuestData) cd.RpcResult

func buildQuestProgressFilterCheckers() map[reflect.Type]*CheckProgressFilterConditionFunc {
	ret := map[reflect.Type]*CheckProgressFilterConditionFunc{}
	return ret
}

var ProgressFilterCheckers = buildQuestProgressFilterCheckers()

func initQuestProgressFilterHandler() {
	addProgressFilterHandler(reflect.TypeOf(public_protocol_common.QuestProgressConditionData_RoleLevel{}), updateProgressFilterByPlayerLevel)
}

func addProgressFilterHandler(ProgressFilterType reflect.Type, update_f CheckProgressFilterConditionFunc) {
	ProgressFilterCheckers[ProgressFilterType] = &update_f
}

func GetQuestProgressFilterUpdateHandler(t reflect.Type) CheckProgressFilterConditionFunc {
	if len(ProgressFilterCheckers) == 0 {
		initQuestProgressFilterHandler()
	}
	return *ProgressFilterCheckers[t]
}

func updateProgressFilterByPlayerLevel(ctx *cd.RpcContext, params logic_quest.TriggerParams,
	progress_config *public_protocol_config.DQuestConditionProgress, user_quest_data *public_protocol_pbdesc.DUserQuestData) cd.RpcResult {
	// TODO: 从配置中获取所需的玩家等级
	return cd.CreateRpcResultOk()
}
