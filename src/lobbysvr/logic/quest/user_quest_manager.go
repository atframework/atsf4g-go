package lobbysvr_logic_quest

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// DaySeconds 一天的秒数
const DaySeconds int64 = 24 * 3600

// TriggerParams 对应 C++ 的 trigger_params_t。
// 字段命名采用驼峰且导出（首字母大写），以便模块内外访问。
type TriggerParams struct {
	X                 int64
	HasX              bool
	Y                 int64
	HasY              bool
	StrVal            string // 用于放 battle id 等信息
	HasStrVal         bool
	SpecifyQuestID    int32
	HasSpecifyQuestID bool
}

type UserQuestManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
	// DumpQuestInfo(to *lobbysvr_protocol_pbdesc.SCUserGetInfoRsp_NormalizeItemData)

	QueryQuestStatus(questID int32) public_protocol_common.EnQuestStatus
	QueryQuestIsFinish(questID int32) bool
	// AddItem(questID int32) cd.RpcResult
	ReceivedQuestSReward(_ctx *cd.RpcContext, questId []int32, autoReceived bool) cd.RpcResult
	ReceivedQuestReward(_ctx *cd.RpcContext, questId int32, autoReceived bool) cd.RpcResult
}
