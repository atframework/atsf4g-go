package lobbysvr_data

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"time"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	private_protocol_log "github.com/atframework/atsf4g-go/component-protocol-private/log/protocol/log"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"

	config "github.com/atframework/atsf4g-go/component-config"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"

	lobbysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	lobbysvr_client_rpc "github.com/atframework/atsf4g-go/service-lobbysvr/rpc/lobbyclientservice"
)

type noCopy struct{}

type Result = cd.RpcResult

type userItemManagerWrapper struct {
	idRange UserItemTypeIdRange
	manager UserItemManagerImpl
}

type UserDirtyData struct {
	dirtyChangeSync *lobbysvr_protocol_pbdesc.SCUserDirtyChgSync
}

func (d *UserDirtyData) MutableNormalDirtyChangeMessage() *lobbysvr_protocol_pbdesc.SCUserDirtyChgSync {
	if d.dirtyChangeSync == nil {
		d.dirtyChangeSync = &lobbysvr_protocol_pbdesc.SCUserDirtyChgSync{}
	}

	return d.dirtyChangeSync
}

type userDirtyHandles struct {
	dumpDirty  func(cd.RpcContext, *UserDirtyData) bool
	clearCache func(cd.RpcContext)
}

type UserLazyEvalationPriority int32

const (
	// 缓式评估优先级
	// 默认优先级约定为不依赖其他模块
	UserLazyEvalationPriority_Default UserLazyEvalationPriority = 1

	// 统计类的优先级较低
	UserLazyEvalationPriority_Statistic UserLazyEvalationPriority = 299999

	// 任务的放最后，因为可能很多模块的数值计算会触发任务变化
	UserLazyEvalationPriority_Quest UserLazyEvalationPriority = 999999
)

type UserLazyEvalationHandle func(cd.RpcContext, *User)

type UserLazyEvalationData struct {
	Handle UserLazyEvalationHandle
	Name   string
}

type User struct {
	uc.UserCache

	loginTaskLock                 sync.Mutex
	loginTaskId                   uint64
	isLoginInited                 bool
	refreshLimitSecondChenckpoint int64
	refreshLimitMinuteChenckpoint int64

	moduleManagerMap map[lu.TypeID]UserModuleManagerImpl
	itemManagerList  []userItemManagerWrapper

	dirtyHandles                   map[unsafe.Pointer]userDirtyHandles
	lazyEvalationHandles           map[UserLazyEvalationPriority]map[string]UserLazyEvalationData
	lazyEvalationRunningPriority   int32
	lazyEvalationRunningName       string
	lazyEvalationRunningReverseReg map[string]map[string]int64
}

func init() {
	var _ uc.UserImpl = (*User)(nil)
}

func (u *User) IsWriteable() bool {
	return u.isLoginInited
}

func (u *User) CanBeWriteable() bool {
	return true
}

func createUser(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) *User {
	ret := &User{
		UserCache: uc.CreateUserCache(ctx, zoneId, userId, openId, actorExecutor),

		loginTaskLock:    sync.Mutex{},
		loginTaskId:      0,
		isLoginInited:    false,
		moduleManagerMap: make(map[lu.TypeID]UserModuleManagerImpl),
		itemManagerList:  make([]userItemManagerWrapper, 0),

		dirtyHandles:                   make(map[unsafe.Pointer]userDirtyHandles),
		lazyEvalationHandles:           make(map[UserLazyEvalationPriority]map[string]UserLazyEvalationData),
		lazyEvalationRunningPriority:   math.MinInt32,
		lazyEvalationRunningName:       "",
		lazyEvalationRunningReverseReg: make(map[string]map[string]int64),
	}
	ret.Impl = ret

	for _, creator := range userModuleManagerCreators {
		mgr := creator.fn(ctx, ret)
		if mgr != nil {
			ret.registerModuleManager(creator.typeInst, mgr)
		}
	}

	for _, creator := range userItemManagerCreators {
		mgr := creator.fn(ctx, ret)
		if mgr != nil {
			for _, idRange := range creator.descriptor.GetTypeIdRanges() {
				ret.itemManagerList = append(ret.itemManagerList, userItemManagerWrapper{
					idRange: idRange,
					manager: mgr,
				})
			}
			mgr.BindDescriptor(creator.descriptor)
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
		if prev.idRange.endTypeId > curr.idRange.beginTypeId {
			ctx.LogError("user item manager type id range conflict",
				"prev_manager", lu.GetTypeID(prev.manager).String(),
				"curr_manager", lu.GetTypeID(curr.manager).String(),
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
	uc.SetCreateUserImplFn(func(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) uc.UserImpl {
		return createUser(ctx, zoneId, userId, openId, actorExecutor)
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

func (u *User) RefreshLimit(ctx cd.RpcContext, now time.Time) {
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

func (u *User) InitFromDB(ctx cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	result := u.UserCache.InitFromDB(ctx, srcTb)
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

func (u *User) DumpToDB(ctx cd.RpcContext, dstDb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	result := u.UserCache.DumpToDB(ctx, dstDb)
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

func (u *User) createInitItemBatch(ctx cd.RpcContext,
	itemInsts []*public_protocol_common.DItemInstance,
) cd.RpcResult {
	addGuard, result := u.CheckAddItem(ctx, itemInsts)
	if result.IsError() {
		return result
	}

	u.AddItem(ctx, addGuard, &ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_USER),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_USER_INIT_ITEM),
		Parameter:   int64(u.GetUserId()),
	})

	return cd.CreateRpcResultOk()
}

func (u *User) createInitItemOneByOne(ctx cd.RpcContext,
	itemInsts []*public_protocol_common.DItemInstance,
) cd.RpcResult {
	if len(itemInsts) == 0 {
		return cd.CreateRpcResultOk()
	}

	for _, itemInst := range itemInsts {
		addGuard, result := u.CheckAddItem(ctx, []*public_protocol_common.DItemInstance{itemInst})
		if result.IsError() {
			ctx.LogError("user create init generate item from offset failed", "error", result.Error,
				"item_type_id", itemInst.GetItemBasic().GetTypeId(), "item_count", itemInst.GetItemBasic().GetCount())
			continue
		}

		u.AddItem(ctx, addGuard, &ItemFlowReason{
			MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_USER),
			MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_USER_INIT_ITEM),
			Parameter:   int64(u.GetUserId()),
		})
	}

	return cd.CreateRpcResultOk()
}

func (u *User) CreateInit(ctx cd.RpcContext, versionType uint32) {
	u.UserCache.CreateInit(ctx, versionType)

	for _, mgr := range u.moduleManagerMap {
		mgr.CreateInit(ctx, versionType)
	}

	// 默认昵称
	u.GetAccountInfo().MutableProfile().NickName = fmt.Sprintf("User-%v-%v", u.GetZoneId(), u.GetUserId())

	// 玩家出身表
	initItemCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelUserInitializeItemsAllOfIndex()
	if initItemCfg != nil {
		var initItems []*public_protocol_common.Readonly_DItemOffset
		for _, itemCfg := range *initItemCfg {
			if itemCfg.GetItem().GetTypeId() == 0 || itemCfg.GetItem().GetCount() <= 0 {
				continue
			}

			initItems = append(initItems, itemCfg.GetItem())
		}

		var itemInsts []*public_protocol_common.DItemInstance

		for _, initItem := range initItems {
			itemInst, result := u.GenerateItemInstanceFromCfgOffset(ctx, initItem)
			if result.IsError() {
				ctx.LogError("user create init generate item from offset failed", "error", result.Error,
					"item_type_id", itemInst.GetItemBasic().GetTypeId(), "item_count", itemInst.GetItemBasic().GetCount())
				continue
			}

			itemInsts = append(itemInsts, itemInst)
		}

		initItemResult := u.createInitItemBatch(ctx, itemInsts)
		if initItemResult.IsError() {
			initItemResult.LogWarn(ctx, "user create init batch add item failed, we will try to add item one by one")
			u.createInitItemOneByOne(ctx, itemInsts)
		}
	}
}

func (u *User) LoginInit(ctx cd.RpcContext) {
	u.UserCache.LoginInit(ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.LoginInit(ctx)
	}

	u.OnLogin(ctx)
}

func (u *User) OnLogin(ctx cd.RpcContext) {
	u.isLoginInited = true

	u.UserCache.OnLogin(ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnLogin(ctx)
	}

	{
		log := private_protocol_log.OperationSupportSystemLog{}
		log.MutableLoginFlow()
		u.SendUserOssLog(ctx, &log)
	}
}

func (u *User) OnLogout(ctx cd.RpcContext) {
	u.UserCache.OnLogout(ctx)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnLogout(ctx)
	}

	{
		log := private_protocol_log.OperationSupportSystemLog{}
		log.MutableLogoutFlow()
		u.SendUserOssLog(ctx, &log)
	}
}

func (u *User) OnSaved(ctx cd.RpcContext, version uint64) {
	u.UserCache.OnSaved(ctx, version)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnSaved(ctx, version)
	}

	if u.GetSession() == nil {
		u.isLoginInited = false
	}
}

func (u *User) OnUpdateSession(ctx cd.RpcContext, from *uc.Session, to *uc.Session) {
	u.UserCache.OnUpdateSession(ctx, from, to)

	for _, mgr := range u.moduleManagerMap {
		mgr.OnUpdateSession(ctx, from, to)
	}
}

func (u *User) SetLazyEvalationHandle(ctx cd.RpcContext, priority UserLazyEvalationPriority, key string, fn UserLazyEvalationHandle) {
	if fn == nil || u == nil || key == "" {
		return
	}

	if u.lazyEvalationHandles == nil {
		u.lazyEvalationHandles = make(map[UserLazyEvalationPriority]map[string]UserLazyEvalationData)
	}

	handles, exists := u.lazyEvalationHandles[priority]
	if !exists || handles == nil {
		handles = make(map[string]UserLazyEvalationData)
		u.lazyEvalationHandles[priority] = handles
	}

	handles[key] = UserLazyEvalationData{
		Handle: fn,
		Name:   key,
	}
	if int32(priority) <= u.lazyEvalationRunningPriority {
		to, exists := u.lazyEvalationRunningReverseReg[u.lazyEvalationRunningName]
		if !exists || to == nil {
			to = make(map[string]int64)
			u.lazyEvalationRunningReverseReg[u.lazyEvalationRunningName] = to
		}
		counter, exists := to[key]
		if !exists {
			to[key] = 1
		} else {
			to[key] = counter + 1
		}
	}
}

func (u *User) selectFirstLazyEvalationPriority(previous int32) (UserLazyEvalationPriority, bool) {
	if u == nil || len(u.lazyEvalationHandles) == 0 {
		return 0, false
	}

	found := false
	for priority := range u.lazyEvalationHandles {
		if !found && int32(priority) <= previous {
			continue
		}
		if !found || int32(priority) < previous {
			previous = int32(priority)
			found = true
		}
	}

	return UserLazyEvalationPriority(previous), found
}

func (u *User) runLazyEvalationHandle(ctx cd.RpcContext) {
	if u == nil {
		return
	}
	if len(u.lazyEvalationHandles) == 0 {
		return
	}

	defer func() {
		u.lazyEvalationRunningPriority = math.MinInt32
		u.lazyEvalationRunningName = ""
		clear(u.lazyEvalationRunningReverseReg)
	}()

	// 最多尝试16轮
	maxLazyEvalationRounds := 16
	lazyEvalationRounds := 0
	for ; lazyEvalationRounds < maxLazyEvalationRounds; lazyEvalationRounds++ {
		if len(u.lazyEvalationHandles) == 0 {
			break
		}

		nextPriority := int32(math.MinInt32)

		for priority, exists := u.selectFirstLazyEvalationPriority(nextPriority); exists; priority, exists = u.selectFirstLazyEvalationPriority(nextPriority) {
			nextPriority = int32(priority)
			u.lazyEvalationRunningPriority = nextPriority

			priorityHandles := u.lazyEvalationHandles[priority]
			delete(u.lazyEvalationHandles, priority)
			if len(priorityHandles) == 0 {
				continue
			}
			for _, handle := range priorityHandles {
				if handle.Handle == nil {
					continue
				}
				u.lazyEvalationRunningName = handle.Name
				handle.Handle(ctx, u)
				u.lazyEvalationRunningName = ""
			}
		}
	}

	// 如果反向注册轮数过多，可能是配置出了事件触发死循环，打印错误日志
	if lazyEvalationRounds >= maxLazyEvalationRounds {
		ctx.LogError("lazy evaluation handle reverse registion too many rounds possibly cause infinite loop",
			"rounds", lazyEvalationRounds,
		)
		for runningName, to := range u.lazyEvalationRunningReverseReg {
			for key, counter := range to {
				if counter > 1 {
					ctx.LogWarn("lazy evaluation handle reverse registion possibly cause infinite loop",
						"source_name", runningName,
						"target_name", key,
						"counter", counter,
					)
				}
			}
		}
	}
}

func (u *User) InsertDirtyHandleIfNotExists(key interface{},
	dumpDataHandle func(cd.RpcContext, *UserDirtyData) bool,
	clearCacheHandle func(cd.RpcContext),
) {
	if lu.IsNil(key) {
		return
	}

	if lu.IsNil(dumpDataHandle) {
		dumpDataHandle = nil
	}

	if lu.IsNil(clearCacheHandle) {
		clearCacheHandle = nil
	}

	if dumpDataHandle == nil && clearCacheHandle == nil {
		return
	}

	mapKey := lu.GetDataPointerOfInterface(key)
	if _, exists := u.dirtyHandles[mapKey]; exists {
		return
	}

	u.dirtyHandles[mapKey] = userDirtyHandles{
		dumpDirty:  dumpDataHandle,
		clearCache: clearCacheHandle,
	}
}

func (u *User) SyncClientDirtyCache(ctx cd.RpcContext) {
	// 先执行缓式评估的Handles，期间可能产生结算数据
	u.runLazyEvalationHandle(ctx)

	// 然后才执行真正的数据下发
	u.UserCache.SyncClientDirtyCache()

	if len(u.dirtyHandles) == 0 {
		return
	}

	session := u.GetSession()
	if session == nil {
		return
	}

	dumpData := UserDirtyData{}

	hasDirty := false

	// 脏数据导出
	for _, handles := range u.dirtyHandles {
		if handles.dumpDirty == nil {
			continue
		}

		if handles.dumpDirty(ctx, &dumpData) {
			hasDirty = true
		}
	}

	// 脏数据推送
	if dumpData.dirtyChangeSync != nil && hasDirty {
		err := lobbysvr_client_rpc.SendUserDirtyChgSync(session, dumpData.dirtyChangeSync, 0)
		if err != nil {
			ctx.LogError("send user dirty change sync failed", "error", err)
		}
	}
}

func (u *User) CleanupClientDirtyCache(ctx cd.RpcContext) {
	u.UserCache.CleanupClientDirtyCache()

	// 清理脏数据推送handle
	for _, handles := range u.dirtyHandles {
		if handles.clearCache == nil {
			continue
		}

		handles.clearCache(ctx)
	}

	clear(u.dirtyHandles)
}

func (u *User) SendAllSyncData(ctx cd.RpcContext) error {
	u.SyncClientDirtyCache(ctx)

	u.CleanupClientDirtyCache(ctx)
	return nil
}

func (u *User) OnSendResponse(ctx cd.RpcContext) error {
	if u == nil {
		return nil
	}

	u.SendAllSyncData(ctx)
	u.UserCache.OnSendResponse(ctx)
	return nil
}

func (u *User) UpdateHeartbeat(ctx cd.RpcContext) {
	// TODO: 加速器检查

	// 续期LoginCode,
	u.GetLoginLockInfo().LoginExpired = ctx.GetSysNow().Unix() +
		config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetLoginCodeValidSec().GetSeconds()
}

func (u *User) GetModuleManager(typeInst lu.TypeID) UserModuleManagerImpl {
	if u.moduleManagerMap == nil {
		return nil
	}

	mgr, ok := u.moduleManagerMap[typeInst]
	if !ok {
		return nil
	}

	return mgr
}

func (u *User) GetModuleManagerByName(name string) UserModuleManagerImpl {
	if u.moduleManagerMap == nil {
		return nil
	}

	for typeInst, mgr := range u.moduleManagerMap {
		if typeInst.String() == name {
			return mgr
		}
	}

	return nil
}

func UserGetModuleManager[ManagerType UserModuleManagerImpl](u *User) ManagerType {
	if u == nil {
		var zero ManagerType
		return zero
	}

	ret := u.GetModuleManager(lu.GetTypeIDOf[ManagerType]())
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

func (u *User) registerModuleManager(typeInst lu.TypeID, mgr UserModuleManagerImpl) {
	if u.moduleManagerMap == nil {
		u.moduleManagerMap = make(map[lu.TypeID]UserModuleManagerImpl)
	}

	u.moduleManagerMap[typeInst] = mgr
}
