package lobbysvr_data

import (
	"slices"

	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	ppcfg "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	ppp "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cc "github.com/atframework/atsf4g-go/component-config"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type ItemFlowReason struct {
	MajorReason int32
	MinorReason int32
	Parameter   int64
}

type UserItemTypeIdRange struct {
	beginTypeId int32
	endTypeId   int32
}

type ItemTypeStatistics struct {
	TotalCount int64
}

type ItemAddGuard struct {
	Configure *ppcfg.ExcelItem
	Item      *ppc.DItemInstance
}

type ItemSubGuard struct {
	Item *ppc.DItemBasic
}

func CreateItemTypeStatistics() ItemTypeStatistics {
	return ItemTypeStatistics{
		TotalCount: 0,
	}
}

type UserItemManagerImpl interface {
	GetOwner() *User

	BindDescriptor(descriptor *UserItemManagerDescriptor)

	AddItem(ctx cd.RpcContext, itemOffset []ItemAddGuard, reason *ItemFlowReason) Result
	SubItem(ctx cd.RpcContext, itemOffset []ItemSubGuard, reason *ItemFlowReason) Result

	GenerateItemInstanceFromOffset(ctx cd.RpcContext, itemOffset *ppc.DItemOffset) (*ppc.DItemInstance, Result)
	GenerateItemInstanceFromBasic(ctx cd.RpcContext, itemOffset *ppc.DItemBasic) (*ppc.DItemInstance, Result)

	// 含默认实现 需要转换Item时实现
	UnpackItem(ctx cd.RpcContext, itemOffset *ppc.DItemInstance) ([]*ppc.DItemInstance, Result)

	CheckAddItem(ctx cd.RpcContext, itemOffset []*ppc.DItemInstance) ([]ItemAddGuard, Result)
	CheckSubItem(ctx cd.RpcContext, itemOffset []*ppc.DItemBasic) ([]ItemSubGuard, Result)

	GetTypeStatistics(typeId int32) *ItemTypeStatistics
	GetItemFromBasic(itemBasic *ppc.DItemBasic) (*ppc.DItemInstance, Result)
	ForeachItem(fn func(item *ppc.DItemInstance) bool)
	GetNotEnoughErrorCode(typeId int32) int32

	CheckTypeIdValid(typeId int32) bool
}

type UserItemManagerDescriptor struct {
	typeIdRanges []UserItemTypeIdRange
}

func (d *UserItemManagerDescriptor) GetTypeIdRanges() []UserItemTypeIdRange {
	return d.typeIdRanges
}

func MakeUserItemTypeIdRange(beginTypeId int32, endTypeId int32) UserItemTypeIdRange {
	return UserItemTypeIdRange{
		beginTypeId: beginTypeId,
		endTypeId:   endTypeId,
	}
}

type userItemManagerCreator struct {
	descriptor *UserItemManagerDescriptor
	fn         func(cd.RpcContext, *User) UserItemManagerImpl
}

var userItemManagerCreators = make([]userItemManagerCreator, 0)

func RegisterUserItemManagerCreator(typeIdRanges []UserItemTypeIdRange, creator func(cd.RpcContext, *User) UserItemManagerImpl) {
	if creator == nil {
		panic("nil user item manager creator")
	}

	slices.SortFunc(typeIdRanges, func(a, b UserItemTypeIdRange) int {
		if a.beginTypeId != b.beginTypeId {
			return int(a.beginTypeId - b.beginTypeId)
		}
		return int(a.endTypeId - b.endTypeId)
	})

	descriptor := &UserItemManagerDescriptor{
		typeIdRanges: typeIdRanges,
	}

	userItemManagerCreators = append(userItemManagerCreators, userItemManagerCreator{
		descriptor: descriptor,
		fn:         creator,
	})
}

type UserItemManagerBase struct {
	_ noCopy

	owner *User

	descriptor *UserItemManagerDescriptor
}

func CreateUserItemManagerBase(owner *User, descriptor *UserItemManagerDescriptor) *UserItemManagerBase {
	ret := &UserItemManagerBase{
		owner:      owner,
		descriptor: descriptor,
	}

	return ret
}

func (umb *UserItemManagerBase) BindDescriptor(descriptor *UserItemManagerDescriptor) {
	umb.descriptor = descriptor
}

func (umb *UserItemManagerBase) GetOwner() *User {
	return umb.owner
}

func (umb *UserItemManagerBase) GetNotEnoughErrorCode(_typeId int32) int32 {
	return int32(ppp.EnErrorCode_EN_ERR_ITEM_NOT_ENOUGH)
}

func (umb *UserItemManagerBase) CheckTypeIdValid(typeId int32) bool {
	_, found := slices.BinarySearchFunc(umb.descriptor.typeIdRanges, typeId, func(a UserItemTypeIdRange, b int32) int {
		if a.beginTypeId > b {
			return 1
		}

		if a.endTypeId <= b {
			return -1
		}

		return 0
	})

	return found
}

func (umb *UserItemManagerBase) HasRepeatedItemInstance(itemOffset []*ppc.DItemInstance) bool {
	if len(itemOffset) <= 1 {
		return false
	}

	// 快速查重
	if len(itemOffset) <= 8 {
		for i := 0; i < len(itemOffset); i++ {
			for j := i + 1; j < len(itemOffset); j++ {
				ib := itemOffset[i].GetItemBasic()
				jb := itemOffset[j].GetItemBasic()
				if ib.GetTypeId() == jb.GetTypeId() && ib.GetGuid() == jb.GetGuid() {
					return true
				}
			}
		}
		return false
	}

	mapTypeId := make(map[int32]map[int64]struct{})
	for i := 0; i < len(itemOffset); i++ {
		ib := itemOffset[i].GetItemBasic()
		if ib == nil {
			continue
		}

		mapGuid, ok := mapTypeId[ib.GetTypeId()]
		if !ok {
			mapGuid = make(map[int64]struct{})
			mapTypeId[ib.GetTypeId()] = mapGuid
		}

		if _, found := mapGuid[ib.GetGuid()]; found {
			return true
		}
		mapGuid[ib.GetGuid()] = struct{}{}
	}

	return false
}

func (umb *UserItemManagerBase) HasRepeatedItemBasic(itemOffset []*ppc.DItemBasic) bool {
	if len(itemOffset) <= 1 {
		return false
	}

	// 快速查重
	if len(itemOffset) <= 8 {
		for i := 0; i < len(itemOffset); i++ {
			for j := i + 1; j < len(itemOffset); j++ {
				ib := itemOffset[i]
				jb := itemOffset[j]
				if ib.GetTypeId() == jb.GetTypeId() && ib.GetGuid() == jb.GetGuid() {
					return true
				}
			}
		}
		return false
	}

	mapTypeId := make(map[int32]map[int64]struct{})
	for i := 0; i < len(itemOffset); i++ {
		ib := itemOffset[i]
		mapGuid, ok := mapTypeId[ib.GetTypeId()]
		if !ok {
			newMapGuid := make(map[int64]struct{})
			mapTypeId[ib.GetTypeId()] = newMapGuid
		}

		if _, found := mapGuid[ib.GetGuid()]; found {
			return true
		}
		mapGuid[ib.GetGuid()] = struct{}{}
	}

	return false
}

func (umb *UserItemManagerBase) GetItemCongiure(typeId int32) *ppcfg.ExcelItem {
	return cc.GetConfigManager().GetCurrentConfigGroup().GetExcelItemByItemId(typeId)
}

func (umb *UserItemManagerBase) CreateItemAddGuard(itemOffset []*ppc.DItemInstance) ([]ItemAddGuard, Result) {
	// AddItem 调用方保证不重复
	// if umb.HasRepeatedItemInstance(itemOffset) {
	// 	return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode_EN_ERR_INVALID_PARAM)
	// }

	ret := make([]ItemAddGuard, 0, len(itemOffset))
	for i := 0; i < len(itemOffset); i++ {
		ib := itemOffset[i].GetItemBasic()
		if ib == nil {
			continue
		}

		if ib.Count == 0 {
			continue
		}
		if ib.Count < 0 {
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode_EN_ERR_INVALID_PARAM)
		}

		cfg := umb.GetItemCongiure(ib.GetTypeId())
		if cfg == nil {
			// TODO: EN_ERR_ITEM_INVALID_TYPE_ID
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode_EN_ERR_ITEM_NOT_FOUND)
		}

		ret = append(ret, ItemAddGuard{
			Configure: cfg,
			Item:      itemOffset[i],
		})
	}

	return ret, cd.CreateRpcResultOk()
}

func (umb *UserItemManagerBase) CreateItemSubGuard(itemOffset []*ppc.DItemBasic) ([]ItemSubGuard, Result) {
	if umb.HasRepeatedItemBasic(itemOffset) {
		return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	ret := make([]ItemSubGuard, 0, len(itemOffset))
	for i := 0; i < len(itemOffset); i++ {
		ib := itemOffset[i]
		if ib.Count == 0 {
			continue
		}
		if ib.Count < 0 {
			return nil, cd.CreateRpcResultError(nil, ppp.EnErrorCode_EN_ERR_INVALID_PARAM)
		}
		ret = append(ret, ItemSubGuard{
			Item: ib,
		})
	}

	return ret, cd.CreateRpcResultOk()
}

func (umb *UserItemManagerBase) UnpackItem(ctx cd.RpcContext, itemOffset *ppc.DItemInstance) ([]*ppc.DItemInstance, Result) {
	return []*ppc.DItemInstance{itemOffset}, cd.CreateRpcResultOk()
}
