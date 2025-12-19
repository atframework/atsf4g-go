package lobbysvr_logic_customer

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type UserMallManager interface {
	data.UserModuleManagerImpl

	// 客户端通知
	MallPurchase(ctx cd.RpcContext, productId int32, purchasePriority int32,
		expectCostItems []*public_protocol_common.DItemBasic, rspBody *service_protocol.SCMallPurchaseRsp) int32
	// GetInfo
	FetchData() *public_protocol_pbdesc.DUserMallData

	GetProductCounter(productId int32) *public_protocol_common.DConditionCounterStorage
}
