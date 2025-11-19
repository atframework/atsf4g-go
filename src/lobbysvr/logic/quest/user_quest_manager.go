package lobbysvr_logic_quest

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// DaySeconds 一天的秒数.
const DaySeconds int64 = 24 * 3600

type UserQuestManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl

	// 查询任务状态
	QueryQuestStatus(questID int32) public_protocol_common.EnQuestStatus
	// 任务是否完成
	QueryQuestIsFinish(questID int32) bool

	// 领取任务奖励
	ReceivedQuestSReward(_ctx cd.RpcContext, questIDs []int32, autoReceived bool) cd.RpcResult
	ReceivedQuestReward(_ctx cd.RpcContext, questID int32, autoReceived bool) cd.RpcResult

	// 触发任务事件
	QuestTriggerEvent(_ctx cd.RpcContext, triggerType public_protocol_common.EnQuestTriggerType,
		param *private_protocol_pbdesc.QuestTriggerParams)

	// 导出任务信息
	DumpQuestInfo(questData *public_protocol_pbdesc.DUserQuestsData)
}
