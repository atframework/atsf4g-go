package lobbysvr_logic_item

import (
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/inventory/internal"
)

type UserInventoryManager interface {
	data.UserItemManagerImpl
	data.UserModuleManagerImpl
}

func init() {
	data.RegisterUserModuleManagerCreator[UserInventoryManager](func(_ctx *cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return impl.CreateUserInventoryManager(owner)
	})

	data.RegisterUserItemManagerCreator([]data.UserItemTypeIdRange{
		data.MakeUserItemTypeIdRange(
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN),
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END)),
		data.MakeUserItemTypeIdRange(
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_PROP_BEGIN),
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_PROP_END)),
		data.MakeUserItemTypeIdRange(
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_MISC_BEGIN),
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_MISC_END)),
		data.MakeUserItemTypeIdRange(
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_CHARACTER_PROP_BEGIN),
			int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_CHARACTER_PROP_END)),
	}, func(ctx *cd.RpcContext, owner *data.User, descriptor *data.UserItemManagerDescriptor) data.UserItemManagerImpl {
		mgr := data.UserGetModuleManager[UserInventoryManager](owner)
		if mgr == nil {
			ctx.LogError("can not find user inventory manager", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		convert, ok := mgr.(data.UserItemManagerImpl)
		if !ok || convert == nil {
			ctx.LogError("user inventory manager does not implement UserItemManagerImpl", "zone_id", owner.GetZoneId(), "user_id", owner.GetUserId())
			return nil
		}
		return convert
	})
}
