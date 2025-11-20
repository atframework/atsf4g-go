package lobbysvr_logic_random_pool_impl

import (
	"fmt"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	config "github.com/atframework/atsf4g-go/component-config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	logic_random_pool "github.com/atframework/atsf4g-go/service-lobbysvr/logic/random_pool"
)

func init() {
	var _ logic_random_pool.UserRandomPoolManager = (*UserRandomPoolManager)(nil)

	data.RegisterUserModuleManagerCreator[logic_random_pool.UserRandomPoolManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return CreateUserRandomPoolManager(owner)
	})

	data.RegisterUserItemManagerCreator([]data.UserItemTypeIdRange{
		data.MakeUserItemTypeIdRange(
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_BEGIN),
			int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_END)),
	}, func(ctx cd.RpcContext, owner *data.User) data.UserItemManagerImpl {
		mgr := data.UserGetModuleManager[logic_random_pool.UserRandomPoolManager](owner)
		if mgr == nil {
			ctx.LogError("can not find user RandomPool manager", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		convert, ok := mgr.(data.UserItemManagerImpl)
		if !ok || convert == nil {
			ctx.LogError("user RandomPool manager does not implement UserItemManagerImpl", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		return convert
	})
}

type UserRandomPoolManager struct {
	owner *data.User
	data.UserModuleManagerBase
	data.UserItemManagerBase
}

func CreateUserRandomPoolManager(owner *data.User) *UserRandomPoolManager {
	return &UserRandomPoolManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
	}
}

func (m *UserRandomPoolManager) GetOwner() *data.User {
	return m.owner
}

/////////////////////// ITEM MANAGER //////////////////////////////

func (m *UserRandomPoolManager) UnpackItem(ctx cd.RpcContext, itemOffset *public_protocol_common.DItemInstance) ([]*public_protocol_common.DItemInstance, data.Result) {
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	result, unpackItem := config.RandomWithPool(itemOffset.GetItemBasic().GetTypeId(), itemOffset.GetItemBasic().GetCount(), nil)
	if result != 0 {
		ctx.LogError("unpack random pool item failed", "user_id", m.owner.GetUserId(), "zone_id", m.owner.GetZoneId(), "type_id", itemOffset.GetItemBasic().GetTypeId(), "error_code", result)
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(result))
	}

	itemInst, genResult := m.GetOwner().GenerateMultipleItemInstancesFromOffset(ctx, unpackItem, true)
	if genResult.IsError() {
		ctx.LogError("user cUnpackItem GenerateMultipleItemInstancesFromOffset failed", "error", genResult.Error,
			"user_id", m.GetOwner().GetUserId(), "zone_id", m.GetOwner().GetZoneId())
		return nil, genResult
	}
	return itemInst, cd.CreateRpcResultOk()
}

/////////////////////// NOT_IMPLEMENTED //////////////////////////////

func (m *UserRandomPoolManager) GenerateItemInstanceFromCfgOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.Readonly_DItemOffset) (*public_protocol_common.DItemInstance, data.Result) {
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return &public_protocol_common.DItemInstance{
		ItemBasic: &public_protocol_common.DItemBasic{
			TypeId: itemOffset.GetTypeId(),
			Count:  itemOffset.GetCount(),
		},
	}, cd.CreateRpcResultOk()
}

func (m *UserRandomPoolManager) GenerateItemInstanceFromOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.DItemOffset) (*public_protocol_common.DItemInstance, data.Result) {
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return &public_protocol_common.DItemInstance{
		ItemBasic: &public_protocol_common.DItemBasic{
			TypeId: itemOffset.GetTypeId(),
			Count:  itemOffset.GetCount(),
		},
	}, cd.CreateRpcResultOk()
}

func (m *UserRandomPoolManager) GenerateItemInstanceFromBasic(ctx cd.RpcContext, itemOffset *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) CheckAddItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]data.ItemAddGuard, data.Result) {
	return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) AddItem(ctx cd.RpcContext, itemOffset []data.ItemAddGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) CheckSubItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemBasic) ([]data.ItemSubGuard, data.Result) {
	return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) SubItem(ctx cd.RpcContext, itemOffset []data.ItemSubGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) GetTypeStatistics(typeId int32) *data.ItemTypeStatistics {
	return nil
}

func (m *UserRandomPoolManager) GetItemFromBasic(itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, data.Result) {
	return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_IMPLEMENTED)
}

func (m *UserRandomPoolManager) ForeachItem(fn func(item *public_protocol_common.DItemInstance) bool) {
}
