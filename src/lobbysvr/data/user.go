package lobbysvr_data

import (
	"log/slog"
	"reflect"
	"slices"
	"time"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type Result = cd.DispatcherErrorResult

type userItemManagerWrapper struct {
	idRange userItemTypeIdRange
	manager UserItemManagerImpl
}

type User struct {
	uc.UserCache

	refreshLimitSecondChenckpoint int64
	refreshLimitMinuteChenckpoint int64

	moduleManagerMap map[reflect.Type]UserModuleManagerImpl
	itemManagerList  []userItemManagerWrapper
}

func (u *User) Init() {
	u.UserCache.Init(u)

	// TODO: 初始化各类Manager
}

func createUser(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) *User {
	ret := &User{
		UserCache:        uc.CreateUserCache(zoneId, userId, openId),
		moduleManagerMap: make(map[reflect.Type]UserModuleManagerImpl),
		itemManagerList:  make([]userItemManagerWrapper, 0),
	}

	for _, creator := range userModuleManagerCreators {
		mgr := creator.fn(ret)
		if mgr != nil {
			ret.registerModuleManager(creator.typeInst, mgr)
		}
	}

	for _, creator := range userItemManagerCreators {
		mgr := creator.fn(ret)
		if mgr != nil {
			for _, idRange := range creator.userItemTypeIdRanges {
				ret.itemManagerList = append(ret.itemManagerList, userItemManagerWrapper{
					idRange: idRange,
					manager: mgr,
				})
			}
		}
	}

	slices.SortFunc(ret.itemManagerList, func(a, b userItemManagerWrapper) int {
		if a.idRange.beginTypeId != b.idRange.beginTypeId {
			return int(a.idRange.beginTypeId - b.idRange.beginTypeId)
		}
		return int(a.idRange.endTypeId - b.idRange.endTypeId)
	})

	// Check item range conflict
	for i := 1; i < len(ret.itemManagerList); i++ {
		prev := ret.itemManagerList[i-1]
		curr := ret.itemManagerList[i]
		if prev.idRange.endTypeId >= curr.idRange.beginTypeId {
			if ctx.Logger != nil {
				ctx.Logger.Error("user item manager type id range conflict",
					"prev_manager", reflect.TypeOf(prev.manager).String(),
					"curr_manager", reflect.TypeOf(curr.manager).String(),
					"prev_range", prev.idRange,
					"prev_begin", prev.idRange.beginTypeId,
					"prev_end", prev.idRange.endTypeId,
					"curr_begin", curr.idRange.beginTypeId,
					"curr_end", curr.idRange.endTypeId,
				)
			} else {
				slog.Error("user item manager type id range conflict",
					"prev_manager", reflect.TypeOf(prev.manager).String(),
					"curr_manager", reflect.TypeOf(curr.manager).String(),
					"prev_range", prev.idRange,
					"prev_begin", prev.idRange.beginTypeId,
					"prev_end", prev.idRange.endTypeId,
					"curr_begin", curr.idRange.beginTypeId,
					"curr_end", curr.idRange.endTypeId)
			}
		}
	}

	return ret
}

func init() {
	uc.GlobalUserManager.SetCreateUserCallback(func(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) uc.UserImpl {
		ret := createUser(ctx, zoneId, userId, openId)
		return ret
	})
}

func (u *User) RefreshLimit(ctx *cd.RpcContext, now time.Time) {
	nowUnix := now.Unix()
	minuteCheckpoint := nowUnix / 60
	refreshSecond := nowUnix != u.refreshLimitSecondChenckpoint
	refreshMinute := minuteCheckpoint != u.refreshLimitMinuteChenckpoint
	if refreshSecond {
		u.refreshLimitSecondChenckpoint = nowUnix
	}
	if refreshMinute {
		u.refreshLimitMinuteChenckpoint = minuteCheckpoint
	}

	for _, mgr := range u.moduleManagerMap {
		mgr.RefreshLimit(ctx)
		if refreshSecond {
			mgr.RefreshLimitSecond(ctx)
		}
		if refreshMinute {
			mgr.RefreshLimitMinute(ctx)
		}
	}
}

func (u *User) GetModuleManager(typeInst reflect.Type) UserModuleManagerImpl {
	if u.moduleManagerMap == nil {
		return nil
	}

	mgr, ok := u.moduleManagerMap[typeInst]
	if !ok {
		return nil
	}

	return mgr
}

func GetModuleManager[ManagerType any](u *User) UserModuleManagerImpl {
	if u == nil {
		return nil
	}

	return u.GetModuleManager(reflect.TypeOf((*ManagerType)(nil)).Elem())
}

func (u *User) registerModuleManager(typeInst reflect.Type, mgr UserModuleManagerImpl) {
	if u.moduleManagerMap == nil {
		u.moduleManagerMap = make(map[reflect.Type]UserModuleManagerImpl)
	}

	u.moduleManagerMap[typeInst] = mgr
}

func (u *User) GetItemManager(typeId int32) UserItemManagerImpl {
	if u.itemManagerList == nil {
		return nil
	}
	// Binary search
	index, found := slices.BinarySearchFunc(u.itemManagerList, typeId, func(a userItemManagerWrapper, b int32) int {
		if a.idRange.endTypeId <= b {
			return -1
		}
		if a.idRange.beginTypeId > b {
			return 1
		}
		return 0
	})

	if index < 0 || index >= len(u.itemManagerList) || !found {
		return nil
	}

	return u.itemManagerList[index].manager
}
