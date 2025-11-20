package lobbysvr_logic_quest_data

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

type CheckUnlockConditionFunc = func(ctx cd.RpcContext,
	rule *public_protocol_common.Readonly_DQuestUnlockConditionItem, owner *data.User) cd.RpcResult

func QuestUnlockRuleCheckers() map[public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID]*CheckUnlockConditionFunc {
	ret := map[public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID]*CheckUnlockConditionFunc{}
	return ret
}

var conditionUnlockCheckers = QuestUnlockRuleCheckers()

func initUnlockHandler() {
	addUnlockHanlder(public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID_PlayerLevel, UnlockByPlayerLevel)
}

func addUnlockHanlder(t public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID, f CheckUnlockConditionFunc) {
	conditionUnlockCheckers[t] = &f
}

// GetQuestUnlockHandle 获取所有已注册的解锁条件处理器
// 注意：如果多个 goroutine 同时修改 handlers（AddHandler、RegisterDefaultUnlockHandler 等），
// 需要在外部加锁或在内部加入并发保护（例如使用 sync.RWMutex）。
func GetQuestUnlockHandle() map[public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID]*CheckUnlockConditionFunc {
	if len(conditionUnlockCheckers) == 0 {
		initUnlockHandler()
	}
	return conditionUnlockCheckers
}

func UnlockByPlayerLevel(_ cd.RpcContext, rule *public_protocol_common.Readonly_DQuestUnlockConditionItem,
	owner *data.User) cd.RpcResult {
	if owner == nil {
		return cd.CreateRpcResultError(fmt.Errorf("owner is nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	mgr := data.UserGetModuleManager[logic_user.UserBasicManager](owner)
	if mgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("UserBasicManager is nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	userLevel := mgr.GetUserLevel()
	requiredLevel := uint32(rule.GetPlayerLevel()) //nolint:gosec
	if userLevel >= requiredLevel {
		return cd.CreateRpcResultOk()
	}

	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_MIN_LEVEL_LIMIT)
}
