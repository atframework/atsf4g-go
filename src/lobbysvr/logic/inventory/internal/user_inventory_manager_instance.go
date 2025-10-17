package lobbysvr_logic_item

import (
	"math"

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
	if !ok {
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

func (g *UserInventoryItemGroup) subGroupCount(item *ppc.DItemInstance, count int64) {
	if item == nil || count <= 0 || g == nil {
		return
	}

	if count > item.GetItemBasic().GetCount() {
		count = item.GetItemBasic().GetCount()
	}
	resetStats := false
	if count > g.statistics.TotalCount {
		count = g.statistics.TotalCount
		resetStats = true
	}

	item.ItemBasic.Count -= count
	if item.GetItemBasic().GetCount() <= 0 {
		delete(g.items, item.GetItemBasic().GetGuid())
	}

	g.statistics.TotalCount -= count

	if resetStats {
		g.recalcStatistics()
	}
}

type UserInventoryManager struct {
	owner *data.User

	data.UserModuleManagerBase
	data.UserItemManagerBase

	// type_id -> UserInventoryItemGroup
	itemGroups map[int32]*UserInventoryItemGroup
}

func CreateUserInventoryManager(owner *data.User) *UserInventoryManager {
	return &UserInventoryManager{
		owner:                 owner,
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		UserItemManagerBase:   *data.CreateUserItemManagerBase(owner, nil),
	}
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
	if !ok {
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

func (m *UserInventoryManager) GetOwner() *data.User { return m.owner }

func (m *UserInventoryManager) RefreshLimitSecond(_ctx *cd.RpcContext) {
	// TODO: 限时道具移除
}

func (m *UserInventoryManager) AddItem(ctx *cd.RpcContext, itemOffset []data.ItemAddGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) SubItem(ctx *cd.RpcContext, itemOffset []data.ItemSubGuard, reason *data.ItemFlowReason) data.Result {
	return cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) CheckAddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance) ([]data.ItemAddGuard, data.Result) {
	return m.CreateItemAddGuard(itemOffset)
}

func (m *UserInventoryManager) CheckSubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic) ([]data.ItemSubGuard, data.Result) {
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

		if group.statistics.TotalCount < sub.Item.GetCount() {
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode(m.GetNotEnoughErrorCode(sub.Item.GetTypeId())))
		}
	}

	return guard, cd.CreateRpcResultOk()
}

func (m *UserInventoryManager) GetTypeStatistics(typeId int32) *data.ItemTypeStatistics {
	group, ok := m.itemGroups[typeId]
	if !ok {
		return nil
	}

	return &group.statistics
}

func (m *UserInventoryManager) GetItemFromBasic(itemBasic *ppc.DItemBasic) (*ppc.DItemInstance, data.Result) {
	return nil, cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserInventoryManager) ForeachItem(fn func(item *ppc.DItemInstance) bool) {
	if fn == nil {
		return
	}

	for _, group := range m.itemGroups {
		for _, item := range group.items {
			if !fn(item) {
				return
			}
		}
	}
}
