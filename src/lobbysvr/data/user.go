package lobbysvr_data

import (
	"reflect"
	"slices"
	"sync"
	"time"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type noCopy struct{}

type Result = cd.RpcResult

type userItemManagerWrapper struct {
	idRange userItemTypeIdRange
	manager UserItemManagerImpl
}

type User struct {
	uc.UserCache

	loginTaskLock                 sync.Mutex
	loginTaskId                   uint64
	isLoginInited                 bool
	refreshLimitSecondChenckpoint int64
	refreshLimitMinuteChenckpoint int64

	client_info_ uc.UserDirtyWrapper[public_protocol_pbdesc.DClientDeviceInfo]

	moduleManagerMap map[reflect.Type]UserModuleManagerImpl
	itemManagerList  []userItemManagerWrapper
}

func (u *User) Init() {
	u.UserCache.Init(u)

	// TODO: 初始化各类Manager
}

func (u *User) IsWriteable() bool {
	return u.isLoginInited
}

func createUser(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) *User {
	ret := &User{
		UserCache: uc.CreateUserCache(zoneId, userId, openId),

		loginTaskLock:    sync.Mutex{},
		loginTaskId:      0,
		isLoginInited:    false,
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
			ctx.GetDefaultLogger().Error("user item manager type id range conflict",
				"prev_manager", reflect.TypeOf(prev.manager).String(),
				"curr_manager", reflect.TypeOf(curr.manager).String(),
				"prev_range", prev.idRange,
				"prev_begin", prev.idRange.beginTypeId,
				"prev_end", prev.idRange.endTypeId,
				"curr_begin", curr.idRange.beginTypeId,
				"curr_end", curr.idRange.endTypeId,
			)
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

func (u *User) TryLockLoginTask(taskId uint64) bool {
	if !u.loginTaskLock.TryLock() {
		return false
	}

	u.loginTaskId = taskId
	return true
}

func (u *User) UnlockLoginTask(taskId uint64) {
	if u.loginTaskId != taskId {
		return
	}

	u.loginTaskId = 0
	u.loginTaskLock.Unlock()
}

func (u *User) GetLoginTaskId() uint64 {
	return u.loginTaskId
}

func (u *User) RefreshLimit(ctx *cd.RpcContext, now time.Time) {
	// Base action
	u.UserCache.RefreshLimit(ctx, now)

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

func (u *User) InitFromDB(self uc.UserImpl, ctx *cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	result := u.UserCache.InitFromDB(self, ctx, srcTb)
	if result.IsError() {
		return result
	}

	for _, mgr := range u.moduleManagerMap {
		result = mgr.InitFromDB(ctx, srcTb)
		if result.IsError() {
			return result
		}
	}

	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (u *User) DumpToDB(self uc.UserImpl, ctx *cd.RpcContext, dstDb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	result := u.UserCache.DumpToDB(self, ctx, dstDb)
	if result.IsError() {
		return result
	}

	for _, mgr := range u.moduleManagerMap {
		result = mgr.DumpToDB(ctx, dstDb)
		if result.IsError() {
			return result
		}
	}

	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (u *User) CreateInit(self uc.UserImpl, ctx *cd.RpcContext, versionType uint32) {
	u.UserCache.CreateInit(self, ctx, versionType)

	for _, mgr := range u.moduleManagerMap {
		mgr.CreateInit(ctx, versionType)
	}
}

func (u *User) LoginInit(self uc.UserImpl, ctx *cd.RpcContext) {
	u.UserCache.LoginInit(self, ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.LoginInit(ctx)
	}

	u.OnLogin(u, ctx)
}

func (u *User) OnLogin(self uc.UserImpl, ctx *cd.RpcContext) {
	u.isLoginInited = true

	u.UserCache.OnLogin(self, ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnLogin(ctx)
	}
}

func (u *User) OnLogout(self uc.UserImpl, ctx *cd.RpcContext) {
	u.UserCache.OnLogout(self, ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnLogout(ctx)
	}

	u.isLoginInited = false
}

func (u *User) OnSaved(self uc.UserImpl, ctx *cd.RpcContext, version uint64) {
	u.UserCache.OnSaved(self, ctx, version)

	u.client_info_.ClearDirty(version)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnSaved(ctx, version)
	}
}

func (u *User) OnUpdateSession(self uc.UserImpl, ctx *cd.RpcContext, from *uc.Session, to *uc.Session) {
	u.UserCache.OnUpdateSession(self, ctx, from, to)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnUpdateSession(ctx, from, to)
	}
}

func (u *User) SyncClientDirtyCache() {
	u.UserCache.SyncClientDirtyCache()

	// TODO: 脏数据推送handle
	for _, mgr := range u.moduleManagerMap {
		mgr.SyncClientDirtyCache()
	}
}

func (u *User) CleanupClientDirtyCache() {
	u.UserCache.CleanupClientDirtyCache()

	// TODO: 脏数据推送handle
	for _, mgr := range u.moduleManagerMap {
		mgr.CleanupClientDirtyCache()
	}
}

func (u *User) SendAllSyncData() error {
	u.SyncClientDirtyCache()

	u.CleanupClientDirtyCache()
	return nil
}

func (u *User) UpdateHeartbeat(ctx *cd.RpcContext) {
	// TODO: 加速器检查

	// 续期LoginCode,
	// TODO: 有效期来自配置
	u.GetLoginInfo().LoginCodeExpired = ctx.GetNow().Unix() + int64(20*60)
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

func GetModuleManager[ManagerType any](u *User) ManagerType {
	if u == nil {
		var zero ManagerType
		return zero
	}

	ret := u.GetModuleManager(reflect.TypeOf((*ManagerType)(nil)).Elem())
	if ret == nil {
		var zero ManagerType
		return zero
	}

	convertRet, ok := ret.(ManagerType)
	if !ok {
		var zero ManagerType
		return zero
	}

	return convertRet
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

func (u *User) GetClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	return u.client_info_.Get()
}

func (u *User) MutableClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	return u.client_info_.Mutable(u.GetCurrentDbDataVersion())
}
