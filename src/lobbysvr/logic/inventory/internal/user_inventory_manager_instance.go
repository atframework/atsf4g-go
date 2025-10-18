package lobbysvr_logic_item

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/proto"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	ppp "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type UserInventoryItemGroup struct {
	typeId int32

	// group guid -> item instance
	items map[int64]*ppc.DItemInstance

	// 统计总数
	statistics data.ItemTypeStatistics
}

func (g *UserInventoryItemGroup) recalcStatistics() {
	var totalCount int64 = 0
	for _, item := range g.items {
		totalCount += item.GetItemBasic().GetCount()
	}

	g.statistics.TotalCount = totalCount
}

func (g *UserInventoryItemGroup) MutableGroup(groupId int64) *ppc.DItemInstance {
	ret, ok := g.items[groupId]
	if !ok || ret == nil {
		ret = &ppc.DItemInstance{
			ItemBasic: &ppc.DItemBasic{
				TypeId: g.typeId,
				Count:  0,
				Guid:   groupId,
			},
		}
		g.items[groupId] = ret
		return ret
	}

	return ret
}

func (g *UserInventoryItemGroup) GetGroup(groupId int64) *ppc.DItemInstance {
	if ret, ok := g.items[groupId]; !ok {
		return nil
	} else {
		return ret
	}
}

func (g *UserInventoryItemGroup) addGroupCount(item *ppc.DItemInstance, count int64, maxCount int64) {
	if item == nil || count <= 0 || g == nil {
		return
	}

	if maxCount <= 0 {
		maxCount = math.MaxInt64
	}

	if maxCount-g.statistics.TotalCount < count {
		count = maxCount - g.statistics.TotalCount
	}

	resetStats := false
	if maxCount-item.GetItemBasic().GetCount() < count {
		count = maxCount - item.GetItemBasic().GetCount()
		resetStats = true
	}

	item.MutableItemBasic().Count += count
	g.statistics.TotalCount += count

	if resetStats {
		g.recalcStatistics()
	}
}

func (g *UserInventoryItemGroup) subGroupCount(item *ppc.DItemBasic, count int64) {
	if item == nil || count <= 0 || g == nil {
		return
	}

	if count > item.GetCount() {
		count = item.GetCount()
	}
	resetStats := false
	if count > g.statistics.TotalCount {
		count = g.statistics.TotalCount
		resetStats = true
	}

	item.Count -= count
	if item.GetCount() <= 0 {
		delete(g.items, item.GetGuid())
	}

	g.statistics.TotalCount -= count

	if resetStats {
		g.recalcStatistics()
	}
}

func (g *UserInventoryItemGroup) empty() bool {
	return len(g.items) == 0
}

type UserInventoryManager struct {
	data.UserModuleManagerBase
	data.UserItemManagerBase

	virtualItemManager *UserVirtualItemManager

	// type_id -> UserInventoryItemGroup
	itemGroups map[int32]*UserInventoryItemGroup

	dirtyItems map[int32]*map[int64]struct{}
}

func CreateUserInventoryManager(owner *data.User) *UserInventoryManager {
	ret := &UserInventoryManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),

		itemGroups: make(map[int32]*UserInventoryItemGroup),
		dirtyItems: make(map[int32]*map[int64]struct{}),
	}

	ret.virtualItemManager = createVirtualItemManager(ret)

	return ret
}

func (m *UserInventoryManager) getItemGroup(typeId int32) *UserInventoryItemGroup {
	group, ok := m.itemGroups[typeId]
	if !ok {
		return nil
	}
	return group
}

func (m *UserInventoryManager) mutableItemGroup(typeId int32) *UserInventoryItemGroup {
	group, ok := m.itemGroups[typeId]
	if !ok || group == nil {
		group = &UserInventoryItemGroup{
			typeId:     typeId,
			items:      make(map[int64]*ppc.DItemInstance),
			statistics: data.CreateItemTypeStatistics(),
		}
		m.itemGroups[typeId] = group

		return group
	}
	return group
}

func (m *UserInventoryManager) GetOwner() *data.User {
	return m.UserItemManagerBase.GetOwner()
}

func (m *UserInventoryManager) InitFromDB(_ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	invalidIds := make(map[int32]*map[int64]struct{})

	for typeId, group := range m.itemGroups {
		invalidIds[typeId] = &map[int64]struct{}{}
		for guid := range group.items {
			(*invalidIds[typeId])[guid] = struct{}{}
		}
	}

	if dbUser.GetInventoryData() == nil {
		clear(m.itemGroups)
		return cd.RpcResult{
			Error:        nil,
			ResponseCode: 0,
		}
	}

	for _, itemData := range dbUser.GetInventoryData().GetItem() {
		if itemData == nil {
			continue
		}

		typeId := itemData.GetItemBasic().GetTypeId()
		guid := itemData.GetItemBasic().GetGuid()

		// 脏数据索引移除
		if group, tok := invalidIds[typeId]; tok {
			delete(*group, guid)
			if len(*group) == 0 {
				delete(invalidIds, typeId)
			}
		}

		// 虚拟道具分发
		if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
			typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
			standalone, result := m.virtualItemManager.InitFromDB(_ctx, dbUser, itemData)
			if result.IsError() {
				return result
			}
			if standalone {
				continue
			}
		}

		// 通用道具管理
		itemGroup := m.mutableItemGroup(typeId)
		itemInstance := itemGroup.MutableGroup(guid)
		proto.Reset(itemInstance)
		proto.Merge(itemInstance, itemData)
	}

	for dirtyTypeId, guidSet := range invalidIds {
		group := m.getItemGroup(dirtyTypeId)
		if group == nil {
			continue
		}
		for guid := range *guidSet {
			delete(group.items, guid)
		}
		if group.empty() {
			delete(m.itemGroups, dirtyTypeId)
		}
	}

	// 重算索引
	for _, group := range m.itemGroups {
		group.recalcStatistics()
	}
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) DumpToDB(_ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	itemDbData := dbUser.MutableInventoryData().MutableItem()

	for _, group := range m.itemGroups {
		for _, item := range group.items {
			itemDbData = append(itemDbData, item)
		}
	}

	dbUser.MutableInventoryData().Item = itemDbData

	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) RefreshLimitSecond(ctx *cd.RpcContext) {
	m.virtualItemManager.RefreshLimitSecond(ctx)

	// TODO: 限时道具移除
}

func (m *UserInventoryManager) markItemDirty(typeId int32, guid int64) {
	if m.dirtyItems == nil {
		m.dirtyItems = make(map[int32]*map[int64]struct{})
	}

	if _, ok := m.dirtyItems[typeId]; !ok {
		m.dirtyItems[typeId] = &map[int64]struct{}{}
	}

	(*m.dirtyItems[typeId])[guid] = struct{}{}

	m.UserModuleManagerBase.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(ctx *cd.RpcContext, dirty *data.UserItemDirtyData) {
			for typeId, guidSet := range m.dirtyItems {
				group := m.getItemGroup(typeId)
				if group == nil {
					for guid := range *guidSet {
						dirty.MutableRemoveItem(typeId, guid)
					}
					continue
				}
				for guid := range *guidSet {
					itemInstance := group.GetGroup(guid)
					if itemInstance == nil {
						dirty.MutableRemoveItem(typeId, guid)
						continue
					}

					dumpDirty := dirty.MutableDirtyItem(typeId, guid)
					proto.Reset(dumpDirty)
					proto.Merge(dumpDirty, itemInstance)
				}
			}
		},
		func(_ctx *cd.RpcContext) {
			clear(m.dirtyItems)
		},
	)
}

func (m *UserInventoryManager) AddItem(ctx *cd.RpcContext, itemOffset []data.ItemAddGuard, reason *data.ItemFlowReason) data.Result {
	for i := 0; i < len(itemOffset); i++ {
		add := &itemOffset[i]

		typeId := add.Item.GetItemBasic().GetTypeId()
		// 虚拟道具分发
		if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
			typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
			standalone, _ := m.virtualItemManager.AddItem(ctx, add, reason)
			if standalone {
				continue
			}
		}

		addCount := add.Item.GetItemBasic().GetCount()
		groupGuid := add.Item.GetItemBasic().GetGuid()

		// 通用道具管理
		group := m.mutableItemGroup(typeId)
		if group == nil {
			ctx.LogError("sub item not enough, should failed in CheckAddItem",
				"zone_id", m.UserModuleManagerBase.GetOwner().GetZoneId(), "user_id", m.UserModuleManagerBase.GetOwner().GetUserId(),
				"item_id", typeId, "item_guid", groupGuid, "add_item_count", addCount,
			)
			continue
		}

		// TODO: 堆叠?
		// maxStacking := add.Configure.GetShowMaxStacking()

		addSet := group.MutableGroup(groupGuid)
		beforeCount := group.statistics.TotalCount
		group.addGroupCount(addSet, addCount, 0 /*maxStacking*/)
		afterCount := group.statistics.TotalCount
		if afterCount-beforeCount < addCount {
			ctx.LogError("add item not overflow, should failed in CheckAddItem",
				"zone_id", m.UserModuleManagerBase.GetOwner().GetZoneId(), "user_id", m.UserModuleManagerBase.GetOwner().GetUserId(),
				"item_id", typeId, "item_guid", groupGuid, "before_item_count", beforeCount, "after_item_count", afterCount, "add_item_count",
				addCount,
			)
		}

		m.markItemDirty(typeId, groupGuid)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) SubItem(ctx *cd.RpcContext, itemOffset []data.ItemSubGuard, reason *data.ItemFlowReason) data.Result {
	for i := 0; i < len(itemOffset); i++ {
		sub := &itemOffset[i]

		typeId := sub.Item.GetTypeId()
		// 虚拟道具分发
		if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
			typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
			standalone, _ := m.virtualItemManager.SubItem(ctx, sub, reason)
			if standalone {
				continue
			}
		}

		// 通用道具管理
		group := m.getItemGroup(typeId)
		if group == nil {
			ctx.LogError("sub item not enough, should failed in CheckSubItem",
				"zone_id", m.UserModuleManagerBase.GetOwner().GetZoneId(), "user_id", m.UserModuleManagerBase.GetOwner().GetUserId(),
				"item_id", sub.Item.GetTypeId(), "item_guid", sub.Item.GetGuid(), "sub_item_count", sub.Item.GetCount(),
			)
			continue
		}

		subSet := group.GetGroup(sub.Item.GetGuid())
		if subSet.GetItemBasic().GetCount() < sub.Item.GetCount() {
			ctx.LogError("sub item not enough, should failed in CheckSubItem",
				"zone_id", m.UserModuleManagerBase.GetOwner().GetZoneId(), "user_id", m.UserModuleManagerBase.GetOwner().GetUserId(),
				"item_id", sub.Item.GetTypeId(), "item_guid", sub.Item.GetGuid(), "has_item_count", subSet.GetItemBasic().GetCount(), "sub_item_count", sub.Item.GetCount(),
			)
		}
		group.subGroupCount(sub.Item, sub.Item.GetCount())
		if group.empty() {
			delete(m.itemGroups, sub.Item.GetTypeId())
		}

		m.markItemDirty(sub.Item.GetTypeId(), sub.Item.GetGuid())
	}

	return cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) CheckAddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance) ([]data.ItemAddGuard, data.Result) {
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), ppp.EnErrorCode(ppp.EnErrorCode_EN_ERR_INVALID_PARAM))
	}

	// 虚拟道具分发
	for i := 0; i < len(itemOffset); i++ {
		item := &itemOffset[i]
		typeId := item.GetItemBasic().GetTypeId()
		if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
			typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
			result := m.virtualItemManager.CheckAddItem(ctx, item)
			if result.IsError() {
				return nil, result
			}
		}
	}

	// 通用道具管理
	return m.CreateItemAddGuard(itemOffset)
}

func (m *UserInventoryManager) CheckSubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic) ([]data.ItemSubGuard, data.Result) {
	// 虚拟道具分发
	if itemOffset == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemOffset is nil"), ppp.EnErrorCode(ppp.EnErrorCode_EN_ERR_INVALID_PARAM))
	}

	// 虚拟道具分发
	for i := 0; i < len(itemOffset); i++ {
		item := &itemOffset[i]
		typeId := item.GetTypeId()
		if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
			typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
			result := m.virtualItemManager.CheckSubItem(ctx, item)
			if result.IsError() {
				return nil, result
			}
		}
	}

	// 通用道具管理
	guard, result := m.CreateItemSubGuard(itemOffset)
	if result.IsError() {
		return nil, result
	}

	for i := 0; i < len(guard); i++ {
		sub := &guard[i]
		group := m.getItemGroup(sub.Item.GetTypeId())
		if group == nil {
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode(m.GetNotEnoughErrorCode(sub.Item.GetTypeId())))
		}

		subSet := group.GetGroup(sub.Item.GetGuid())
		if subSet == nil || subSet.GetItemBasic().GetCount() < sub.Item.GetCount() {
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode(m.GetNotEnoughErrorCode(sub.Item.GetTypeId())))
		}
	}

	return guard, cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) GetTypeStatistics(typeId int32) *data.ItemTypeStatistics {
	// 虚拟道具分发
	if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
		typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
		standalone, itemStats := m.virtualItemManager.GetTypeStatistics(typeId)
		if standalone {
			return itemStats
		}
	}

	// 通用道具管理
	group, ok := m.itemGroups[typeId]
	if !ok {
		return nil
	}

	return &group.statistics
}

func (m *UserInventoryManager) GetNotEnoughErrorCode(typeId int32) int32 {
	// 虚拟道具分发
	if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
		typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
		return m.virtualItemManager.GetNotEnoughErrorCode(typeId)
	}

	// 通用道具管理
	return int32(ppp.EnErrorCode_EN_ERR_ITEM_NOT_ENOUGH)
}

func (m *UserInventoryManager) GetItemFromBasic(itemBasic *ppc.DItemBasic) (*ppc.DItemInstance, data.Result) {
	if itemBasic == nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("itemBasic is nil"), ppp.EnErrorCode(ppp.EnErrorCode_EN_ERR_INVALID_PARAM))
	}

	typeId := itemBasic.GetTypeId()

	// 虚拟道具分发
	if typeId >= int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_BEGIN) &&
		typeId < int32(ppc.EnItemTypeRange_EN_ITEM_TYPE_RANGE_VIRTUAL_ITEM_END) {
		standalone, ret, result := m.virtualItemManager.GetItemFromBasic(itemBasic)
		if standalone {
			return ret, result
		}
	}

	// 通用道具管理
	group, ok := m.itemGroups[typeId]
	if !ok {
		return nil, cd.RpcResult{
			Error:        nil,
			ResponseCode: m.GetNotEnoughErrorCode(typeId),
		}
	}

	groupItem := group.GetGroup(itemBasic.GetGuid())
	if groupItem == nil {
		return nil, cd.RpcResult{
			Error:        nil,
			ResponseCode: m.GetNotEnoughErrorCode(typeId),
		}
	}

	return groupItem, cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) ForeachItem(fn func(item *ppc.DItemInstance) bool) {
	if fn == nil {
		return
	}

	// 虚拟道具分发
	m.virtualItemManager.ForeachItem(fn)

	// 通用道具管理
	for _, group := range m.itemGroups {
		for _, item := range group.items {
			if !fn(item) {
				return
			}
		}
	}
}
