package lobbysvr_logic_quest_data

import (
	"reflect"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

type CheckUnlockConditionFunc = func(ctx *cd.RpcContext, rule *public_protocol_common.DQuestUnlockConditionItem) cd.RpcResult

func QuestUnlockRuleCheckers() map[reflect.Type]*CheckUnlockConditionFunc {
	ret := map[reflect.Type]*CheckUnlockConditionFunc{}
	return ret
}

var conditionUnlockCheckers = QuestUnlockRuleCheckers()

func initUnlockHandler() {
	addUnlockHanlder(reflect.TypeOf(public_protocol_common.DQuestUnlockConditionItem_PlayerLevel{}), UnlockByPlayerLevel)
}

func addUnlockHanlder(t reflect.Type, f CheckUnlockConditionFunc) {
	conditionUnlockCheckers[t] = &f
}

// GetQuestUnlockHandle 调用所有注册的处理器。如果任一处理器返回错误，则停止并返回该错误。
// 注意：如果多个 goroutine 同时修改 handlers（AddHandler、RegisterDefaultUnlockHandler 等），
// 需要在外部加锁或在内部加入并发保护（例如使用 sync.RWMutex）。
func GetQuestUnlockHandle() map[reflect.Type]*CheckUnlockConditionFunc {
	if len(conditionUnlockCheckers) == 0 {
		initUnlockHandler()
	}
	return conditionUnlockCheckers
}

func UnlockByPlayerLevel(ctx *cd.RpcContext, rule *public_protocol_common.DQuestUnlockConditionItem) cd.RpcResult {
	// TODO: 从配置中获取所需的玩家等级
	return cd.CreateRpcResultOk()
}
