package lobbysvr_logic_quest

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_unlock "github.com/atframework/atsf4g-go/service-lobbysvr/logic/unlock"
)

// DaySeconds 一天的秒数.
const DaySeconds int64 = 24 * 3600

const InitLoginDays int32 = 1

const DeleteCacheKeepSeconds int64 = DaySeconds * 7

const DQuestNoPeogressValue = 1

type UserQuestManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
	logic_unlock.UserUnlockListener
	// 查询任务状态
	QueryQuestStatus(questID int32) public_protocol_common.EnQuestStatus
	// 任务是否完成
	QueryQuestIsFinish(questID int32) bool

	// 领取任务奖励
	ReceivedQuestsReward(ctx cd.RpcContext, questIDs []int32) (rewards []*public_protocol_pbdesc.DuserQuestRewardData, result cd.RpcResult)
	ReceivedQuestReward(ctx cd.RpcContext, questID int32, autoReceived bool) (rewards []*public_protocol_common.DItemBasic, result cd.RpcResult)

	// 触发任务事件
	QuestTriggerEvent(ctx cd.RpcContext, triggerType private_protocol_pbdesc.QuestTriggerParams_EnParamID,
		param *private_protocol_pbdesc.QuestTriggerParams)

	// 导出任务信息
	DumpQuestInfo(questData *public_protocol_pbdesc.DUserQuestsData)

	// 激活任务 如果任务未解锁||已过期才可以激活
	ActivateQuest(ctx cd.RpcContext, questID int32) cd.RpcResult

	// 客户端请求任务变化
	ClientQueryQuestUpdateStatus(ctx cd.RpcContext)

	GMForceFinishQuest(ctx cd.RpcContext, questID int32) cd.RpcResult
	GMForceUnlockQuest(ctx cd.RpcContext, questID int32) cd.RpcResult
}
