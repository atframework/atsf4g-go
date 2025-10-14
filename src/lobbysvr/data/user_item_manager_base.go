package lobbysvr_data

import (
	"slices"
	"sort"

	ppc "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	ppp "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type ItemFlowReason struct {
	MajorReason int32
	MinorReason int32
	Parameter   int64
}

type userItemTypeIdRange struct {
	beginTypeId int32
	endTypeId   int32
}

type UserItemManagerImpl interface {
	GetOwner() *User

	AddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance, reason *ItemFlowReason) Result
	SubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic, reason *ItemFlowReason) Result

	CheckAddItem(ctx *cd.RpcContext, itemOffset []ppc.DItemInstance) Result
	CheckSubItem(ctx *cd.RpcContext, itemOffset []ppc.DItemBasic) Result

	GetItemFromBasic(itemBasic *ppc.DItemBasic) (*ppc.DItemInstance, Result)
	GetNotEnoughErrorCode(typeId int32) int32

	CheckTypeIdValid(typeId int32) bool
}

type userItemManagerCreator struct {
	userItemTypeIdRanges []userItemTypeIdRange
	fn                   func(*User) UserItemManagerImpl
}

var userItemManagerCreators = make([]userItemManagerCreator, 0)

func RegisterUserItemManagerCreator[ManagerType any](typeIdRanges []userItemTypeIdRange, creator func(*User) UserItemManagerImpl) {
	if creator == nil {
		panic("nil user item manager creator")
	}

	slices.SortFunc(typeIdRanges, func(a, b userItemTypeIdRange) int {
		if a.beginTypeId != b.beginTypeId {
			return int(a.beginTypeId - b.beginTypeId)
		}
		return int(a.endTypeId - b.endTypeId)
	})

	userItemManagerCreators = append(userItemManagerCreators, userItemManagerCreator{
		userItemTypeIdRanges: typeIdRanges,
		fn:                   creator,
	})
}

type UserItemManagerBase struct {
	_ noCopy

	owner *User

	userItemTypeIdRanges []userItemTypeIdRange
}

func CreateUserItemManagerBase(owner *User, userItemTypeIdRanges []userItemTypeIdRange) *UserItemManagerBase {
	ret := &UserItemManagerBase{
		owner:                owner,
		userItemTypeIdRanges: userItemTypeIdRanges,
	}

	sort.Slice(ret.userItemTypeIdRanges, func(i, j int) bool {
		if ret.userItemTypeIdRanges[i].beginTypeId != ret.userItemTypeIdRanges[j].beginTypeId {
			return ret.userItemTypeIdRanges[i].beginTypeId < ret.userItemTypeIdRanges[j].beginTypeId
		}

		return ret.userItemTypeIdRanges[i].endTypeId < ret.userItemTypeIdRanges[j].endTypeId
	})

	return ret
}

func (umb *UserItemManagerBase) GetOwner() *User {
	return umb.owner
}

func (umb *UserItemManagerBase) GetNotEnoughErrorCode(_typeId int32) int32 {
	return int32(ppp.EnErrorCode_EN_ERR_ITEM_NOT_ENOUGH)
}

func (umb *UserItemManagerBase) CheckTypeIdValid(typeId int32) bool {
	_, found := slices.BinarySearchFunc(umb.userItemTypeIdRanges, typeId, func(a userItemTypeIdRange, b int32) int {
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
