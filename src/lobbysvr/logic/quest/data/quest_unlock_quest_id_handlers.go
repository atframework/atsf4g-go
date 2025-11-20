package lobbysvr_logic_quest_data

import (
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

type GetUnlockQuestIDFunc = func(ctx cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams) []int32

func UnlockQuestIDCheckers() map[int32]*GetUnlockQuestIDFunc {
	ret := map[int32]*GetUnlockQuestIDFunc{}
	return ret
}

var unlockQuestIDHandlers = UnlockQuestIDCheckers()

func initUnlockQuestIDHandlers() {
	addUnlockQuestIDHanlder(int32(public_protocol_common.EnQuestUnlockConditionType_EN_QUEST_UNLOCK_CONDITION_TYPE_PLAYER_LEVEL), triggerByPlayerLevel)
}

func addUnlockQuestIDHanlder(t int32, f GetUnlockQuestIDFunc) {
	unlockQuestIDHandlers[t] = &f
}

// GetQuestUnlockHandle 调用所有注册的处理器。如果任一处理器返回错误，则停止并返回该错误。
// 注意：如果多个 goroutine 同时修改 handlers（AddHandler、RegisterDefaultUnlockHandler 等），
// 需要在外部加锁或在内部加入并发保护（例如使用 sync.RWMutex）。
func GetQuestUnlockIDHandler(unlockType int32) *GetUnlockQuestIDFunc {
	if len(unlockQuestIDHandlers) == 0 {
		initUnlockQuestIDHandlers()
	}
	return unlockQuestIDHandlers[unlockType]
}

func triggerByPlayerLevel(_ cd.RpcContext, params *private_protocol_pbdesc.QuestTriggerParams) []int32 {
	cfgGroup := config.GetConfigManager().GetCurrentConfigGroup()
	return config.GetBoundUnlockQuestIds(cfgGroup, public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID_PlayerLevel,
		params.GetPlayerLevel().PreLevel, params.GetPlayerLevel().CurLevel)
}
