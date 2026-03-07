package lobbysvr_logic_mall_impl

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_mall "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mall"
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
)

func registerQuestProgressHandlers() {

	// 购买过指定商品
	logic_quest.RegisterProgressHandler(public_protocol_config.DQuestConditionProgress_EnProgressParamID_PurchaseSpecifiedProductAccumulative,
		questInitPurchaseSpecifiedProductAccumulative,
		questUpdatePurchaseSpecifiedProductAccumulative,
		questprogressKeyIndexPurchaseSpecifiedProductAccumulative,
	)
	// 购买过指定商城
	logic_quest.RegisterProgressHandler(public_protocol_config.DQuestConditionProgress_EnProgressParamID_PurchaseSpecifiedMallAccumulative,
		questInitPurchaseSpecifiedMallAccumulative,
		questUpdatePurchaseSpecifiedMallAccumulative,
		questprogressKeyIndexPurchaseSpecifiedMallAccumulative,
	)

}

// 购买过指定商品
func questInitPurchaseSpecifiedProductAccumulative(ctx cd.RpcContext,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
	user *data.User,
) cd.RpcResult {
	userMallMgr := data.UserGetModuleManager[logic_mall.UserMallManager](user)
	if userMallMgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if questData != nil {
		versionCount := userMallMgr.GetProductCounter(progressCfg.GetPurchaseSpecifiedProductAccumulative()).GetVersionCounter()
		questData.Value = 0
		if versionCount != nil {
			questData.Value = versionCount.GetSumCounter()
		}
	}

	return cd.CreateRpcResultOk()
}

func questUpdatePurchaseSpecifiedProductAccumulative(ctx cd.RpcContext,
	params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
) cd.RpcResult {
	if questData == nil {
		return cd.CreateRpcResultOk()
	}
	if params.GetMallPurchase().GetProductId() == progressCfg.GetPurchaseSpecifiedProductAccumulative() {
		questData.Value += int64(params.GetMallPurchase().GetCount())
	}
	return cd.CreateRpcResultOk()
}

func questprogressKeyIndexPurchaseSpecifiedProductAccumulative(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams) logic_quest.UserQuestProgressIndexParams {
	return logic_quest.UserQuestProgressIndexParams{
		ParamsOne: params.GetMallPurchase().GetProductId(),
	}
}

// 购买过指定商城
func questInitPurchaseSpecifiedMallAccumulative(ctx cd.RpcContext,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
	user *data.User,
) cd.RpcResult {
	userMallMgr := data.UserGetModuleManager[logic_mall.UserMallManager](user)
	if userMallMgr == nil {
		return cd.CreateRpcResultError(fmt.Errorf("can not get UserMallManager"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if questData != nil {
		questData.Value = userMallMgr.GetMallPurchaseSum(public_protocol_common.EnMallType(progressCfg.GetPurchaseSpecifiedMallAccumulative()))
	}

	return cd.CreateRpcResultOk()
}

func questUpdatePurchaseSpecifiedMallAccumulative(ctx cd.RpcContext,
	params *private_protocol_pbdesc.QuestTriggerParams,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
) cd.RpcResult {
	if questData == nil {
		return cd.CreateRpcResultOk()
	}
	if params.GetMallPurchase().GetMallId() == int32(progressCfg.GetPurchaseSpecifiedMallAccumulative()) {
		questData.Value += int64(params.GetMallPurchase().GetCount())
	}
	return cd.CreateRpcResultOk()
}

func questprogressKeyIndexPurchaseSpecifiedMallAccumulative(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams) logic_quest.UserQuestProgressIndexParams {
	return logic_quest.UserQuestProgressIndexParams{
		ParamsOne: params.GetMallPurchase().GetMallId(),
	}
}
