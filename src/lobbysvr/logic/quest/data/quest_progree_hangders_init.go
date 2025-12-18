package lobbysvr_logic_quest_data

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func initProgressFromManager[M data.UserModuleManagerImpl](
	valueGetter func(M, cd.RpcContext, *public_protocol_config.Readonly_DQuestConditionProgress) int64,
) InitProgressConditionFunc {
	return func(ctx cd.RpcContext,
		cfg *public_protocol_config.Readonly_DQuestConditionProgress,
		questData *public_protocol_pbdesc.DUserQuestProgressData, owner *data.User) cd.RpcResult {
		if questData == nil || owner == nil {
			return cd.CreateRpcResultOk()
		}
		mgr := data.UserGetModuleManager[M](owner)
		// 将泛型接口转为 any 类型后再与 nil 比较
		if any(mgr) == nil {
			questData.Value = 0
			return cd.CreateRpcResultOk()
		}
		questData.Value = valueGetter(mgr, ctx, cfg)
		return cd.CreateRpcResultOk()
	}
}

var (
	initProgressNone = func(_ cd.RpcContext, _ *public_protocol_config.Readonly_DQuestConditionProgress,
		_ *public_protocol_pbdesc.DUserQuestProgressData, _ *data.User) cd.RpcResult {
		return cd.CreateRpcResultOk()
	}

	initProgressByPlayerLevel = initProgressFromManager(
		func(mgr logic_user.UserBasicManager, ctx cd.RpcContext, _cfg *public_protocol_config.Readonly_DQuestConditionProgress) int64 {
			return int64(mgr.GetUserLevel())
		})
)

var (
	initProgressByHasItem = func(ctx cd.RpcContext,
		cfg *public_protocol_config.Readonly_DQuestConditionProgress,
		questData *public_protocol_pbdesc.DUserQuestProgressData, owner *data.User) cd.RpcResult {
		if questData == nil || owner == nil {
			return cd.CreateRpcResultOk()
		}
		stattistics := owner.GetItemTypeStatistics(ctx, cfg.GetItemHas())
		if stattistics == nil {
			return cd.CreateRpcResultOk()
		}
		questData.Value = stattistics.TotalCount
		return cd.CreateRpcResultOk()
	}
)
