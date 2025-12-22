package lobbysvr_data

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"sync"
	"time"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	private_protocol_log "github.com/atframework/atsf4g-go/component-protocol-private/log/protocol/log"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

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

	moduleManagerMap map[reflect.Type]UserModuleManagerImpl
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
		moduleManagerMap: make(map[reflect.Type]UserModuleManagerImpl),
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
				"prev_manager", prev.manager.GetReflectType().String(),
				"curr_manager", curr.manager.GetReflectType().String(),
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
		config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetLoginCodeValidSec().GetSeconds()
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

func (u *User) GetModuleManagerByName(name string) UserModuleManagerImpl {
	if u.moduleManagerMap == nil {
		return nil
	}

	for typeInst, mgr := range u.moduleManagerMap {
		if typeInst.Name() == name {
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

func (u *User) AddItem(ctx cd.RpcContext, itemOffset []*ItemAddGuard, reason *ItemFlowReason) Result {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*ItemAddGuard
	})
	for _, offset := range itemOffset {
		if offset == nil {
			continue
		}
		typeId := offset.Item.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*ItemAddGuard
			}{
				data: make([]*ItemAddGuard, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	result := cd.CreateRpcResultOk()

	ossLog := &private_protocol_log.OperationSupportSystemLog{}
	for mgr, group := range splitByMgr {
		subResult := mgr.AddItem(ctx, group.data, reason)
		if subResult.IsError() {
			subResult.LogError(ctx, "user add item failed")
			result = subResult
		}
		for _, itemGuard := range group.data {
			for _, addedItem := range itemGuard.GetAddedItems() {
				ossLog.Detail = nil
				data := ossLog.MutableItemFlow()
				data.OperationType = private_protocol_log.OSSItemFlow_EN_OSS_ITEM_FLOW_OPERATION_TYPE_ADD
				data.OperationCount = addedItem.GetItemBasic().GetCount()
				data.ItemId = addedItem.GetItemBasic().GetTypeId()
				data.AfterCount = mgr.GetTypeStatistics(ctx, addedItem.GetItemBasic().GetTypeId()).GetTotalCount()
				data.MajorReason = public_protocol_common.EnItemFlowReasonMajorType(reason.GetMajorReason())
				data.MinorReason = public_protocol_common.EnItemFlowReasonMinorType(reason.GetMinorReason())
				data.Parameter = reason.GetParameter()
				data.Result = subResult.GetResponseCode()
				u.SendUserOssLog(ctx, ossLog)
			}
		}
	}

	return result
}

func (u *User) SubItem(ctx cd.RpcContext, itemOffset []*ItemSubGuard, reason *ItemFlowReason) Result {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*ItemSubGuard
	})
	for _, offset := range itemOffset {
		if offset == nil {
			continue
		}
		typeId := offset.Item.GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*ItemSubGuard
			}{
				data: make([]*ItemSubGuard, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	result := cd.CreateRpcResultOk()

	ossLog := &private_protocol_log.OperationSupportSystemLog{}
	for mgr, group := range splitByMgr {
		subResult := mgr.SubItem(ctx, group.data, reason)
		if subResult.IsError() {
			subResult.LogError(ctx, "user sub item failed")
			result = subResult
		}
		for _, itemGuard := range group.data {
			ossLog.Detail = nil
			data := ossLog.MutableItemFlow()
			data.OperationType = private_protocol_log.OSSItemFlow_EN_OSS_ITEM_FLOW_OPERATION_TYPE_SUB
			data.OperationCount = itemGuard.Item.GetCount()
			data.ItemId = itemGuard.Item.GetTypeId()
			data.AfterCount = mgr.GetTypeStatistics(ctx, itemGuard.Item.GetTypeId()).GetTotalCount()
			data.MajorReason = public_protocol_common.EnItemFlowReasonMajorType(reason.GetMajorReason())
			data.MinorReason = public_protocol_common.EnItemFlowReasonMinorType(reason.GetMinorReason())
			data.Parameter = reason.GetParameter()
			data.Result = subResult.GetResponseCode()
			u.SendUserOssLog(ctx, ossLog)
		}
	}

	return result
}

func (u *User) UseItem(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result) {
	typeId := itemBasic.GetTypeId()
	if itemBasic.GetCount() <= 0 {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}
	row := config.GetConfigManager().GetCurrentConfigGroup().GetExcelItemByItemId(typeId)
	if row == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}
	if row.GetUseAction().GetActionTypeOneofCase() == 0 {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_CANNOT_USE)
	}
	result := u.checkUseItem(ctx, itemBasic, row.GetUseAction(), useParam)
	if result.IsError() {
		return nil, result
	}

	guard, result := u.CheckSubItem(ctx, []*public_protocol_common.DItemBasic{itemBasic})
	if result.IsError() {
		return nil, result
	}

	u.SubItem(ctx, guard, reason)

	return u.useItemInner(ctx, itemBasic, row.GetUseAction(), useParam, reason)
}

func (u *User) checkUseItem(_ cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useAction *public_protocol_common.Readonly_DItemUseAction, useParam *public_protocol_common.DItemUseParam) Result {
	switch useAction.GetActionTypeOneofCase() {
	case public_protocol_common.DItemUseAction_EnActionTypeID_RandomPool:
		ret, _ := config.RandomWithPool(useAction.GetRandomPool(), itemBasic.GetCount(), useParam.GetRandomPoolIndex())
		if ret != 0 {
			return cd.CreateRpcResultError(nil, ret)
		}
	}
	return cd.CreateRpcResultOk()
}

func (u *User) useItemInner(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic,
	useAction *public_protocol_common.Readonly_DItemUseAction, useParam *public_protocol_common.DItemUseParam, reason *ItemFlowReason) ([]*public_protocol_common.DItemInstance, Result) {
	switch useAction.GetActionTypeOneofCase() {
	case public_protocol_common.DItemUseAction_EnActionTypeID_RandomPool:
		poolId := useAction.GetRandomPool()
		poolCfg := config.GetConfigManager().GetCurrentConfigGroup().GetExcelRandomPoolByPoolId(poolId)
		if poolCfg == nil {
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_FOUND)
		}

		ret, items := config.RandomWithPool(poolId, itemBasic.GetCount(), useParam.GetRandomPoolIndex())
		if ret != 0 {
			return nil, cd.CreateRpcResultError(nil, ret)
		}

		itemInsts, result := u.GenerateMultipleItemInstancesFromOffset(ctx, items, true)
		if result.IsError() {
			return nil, result
		}
		guard, result := u.CheckAddItem(ctx, itemInsts)
		if result.IsError() {
			return nil, result
		}
		u.AddItem(ctx, guard, reason)
		gainItem := []*public_protocol_common.DItemInstance{}
		for _, item := range guard {
			gainItem = append(gainItem, item.GetAddedItems()...)
		}
		return gainItem, cd.CreateRpcResultOk()
	}
	return nil, cd.CreateRpcResultOk()
}

func (u *User) GenerateItemInstanceFromCfgOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.Readonly_DItemOffset) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemOffset.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromCfgOffset(ctx, itemOffset)
}

func (u *User) GenerateItemInstanceFromOffset(ctx cd.RpcContext, itemOffset *public_protocol_common.DItemOffset) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemOffset.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromOffset(ctx, itemOffset)
}

func (u *User) GenerateMultipleItemInstancesFromCfgOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.Readonly_DItemOffset, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemOffset))
	for _, offset := range itemOffset {
		itemInst, result := u.GenerateItemInstanceFromCfgOffset(ctx, offset)
		if result.IsError() {
			if ignoreError {
				ctx.LogError("generate item instance from item offset failed",
					"error", result.Error, "resoponse_code", result.ResponseCode,

					"item_type_id", offset.GetTypeId(), "item_count", offset.GetCount(),
				)
				continue
			}
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) GenerateMultipleItemInstancesFromOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemOffset, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemOffset))
	for _, offset := range itemOffset {
		itemInst, result := u.GenerateItemInstanceFromOffset(ctx, offset)
		if result.IsError() {
			if ignoreError {
				ctx.LogError("generate item instance from item offset failed",
					"error", result.Error, "resoponse_code", result.ResponseCode,

					"item_type_id", offset.GetTypeId(), "item_count", offset.GetCount(),
				)
				continue
			}
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) GenerateItemInstanceFromBasic(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, Result) {
	typeId := itemBasic.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GenerateItemInstanceFromBasic(ctx, itemBasic)
}

func (u *User) GenerateMultipleItemInstancesFromBasic(ctx cd.RpcContext, itemBasic []*public_protocol_common.DItemBasic, ignoreError bool) ([]*public_protocol_common.DItemInstance, Result) {
	ret := make([]*public_protocol_common.DItemInstance, 0, len(itemBasic))
	for _, basic := range itemBasic {
		itemInst, result := u.GenerateItemInstanceFromBasic(ctx, basic)
		if result.IsError() {
			ctx.LogError("generate item instance from item basic failed",
				"error", result.Error, "resoponse_code", result.ResponseCode,

				"item_type_id", basic.GetTypeId(), "item_count", basic.GetCount(),
			)
			return nil, result
		}
		ret = append(ret, itemInst)
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) unpackMergeItemOffset(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]*public_protocol_common.DItemInstance, Result) {
	if len(itemOffset) == 0 {
		return nil, cd.CreateRpcResultOk()
	}

	mergeItemInstan := make(map[int32]map[int64]*public_protocol_common.DItemInstance)
	itemOffsetSize := 0
	for _, offset := range itemOffset {
		// 解包合并ItemOffset
		typeId := offset.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}
		items, result := mgr.UnpackItem(ctx, offset)
		if result.IsError() {
			return nil, result
		}

		for _, item := range items {
			if _, exists := mergeItemInstan[item.GetItemBasic().GetTypeId()]; !exists {
				mergeItemInstan[item.GetItemBasic().GetTypeId()] = make(map[int64]*public_protocol_common.DItemInstance)
			}
			v := mergeItemInstan[item.GetItemBasic().GetTypeId()]

			existItem, exists := v[item.GetItemBasic().GetGuid()]
			if exists {
				existItem.GetItemBasic().Count += item.GetItemBasic().GetCount()
			} else {
				v[item.GetItemBasic().GetGuid()] = item.Clone()
				itemOffsetSize++
			}
		}
	}

	// 输出
	ret := make([]*public_protocol_common.DItemInstance, 0, itemOffsetSize)
	for _, guidMap := range mergeItemInstan {
		for _, item := range guidMap {
			ret = append(ret, item)
		}
	}

	return ret, cd.CreateRpcResultOk()
}

func (u *User) CheckAddItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemInstance) ([]*ItemAddGuard, Result) {
	unpackMergeItemOffset, result := u.unpackMergeItemOffset(ctx, itemOffset)
	if result.IsError() {
		return nil, result
	}

	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*public_protocol_common.DItemInstance
	})
	for _, offset := range unpackMergeItemOffset {
		typeId := offset.GetItemBasic().GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user add item failed, item manager not found", "type_id", typeId, "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*public_protocol_common.DItemInstance
			}{
				data: make([]*public_protocol_common.DItemInstance, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	ret := make([]*ItemAddGuard, 0, len(unpackMergeItemOffset))
	for mgr, group := range splitByMgr {
		subRet, subResult := mgr.CheckAddItem(ctx, group.data)
		if subResult.IsError() {
			return nil, subResult
		}
		ret = append(ret, subRet...)
	}
	return ret, cd.CreateRpcResultOk()
}

func (u *User) CheckSubItem(ctx cd.RpcContext, itemOffset []*public_protocol_common.DItemBasic) ([]*ItemSubGuard, Result) {
	splitByMgr := make(map[UserItemManagerImpl]*struct {
		data []*public_protocol_common.DItemBasic
	})
	for _, offset := range itemOffset {
		typeId := offset.GetTypeId()
		mgr := u.GetItemManager(typeId)
		if mgr == nil {
			ctx.LogWarn("user sub item failed, item manager not found", "type_id", typeId)
			return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
		}

		group, exists := splitByMgr[mgr]
		if !exists || group == nil {
			group = &struct {
				data []*public_protocol_common.DItemBasic
			}{
				data: make([]*public_protocol_common.DItemBasic, 0, 2),
			}
			splitByMgr[mgr] = group
		}
		group.data = append(group.data, offset)
	}

	ret := make([]*ItemSubGuard, 0, len(itemOffset))
	for mgr, group := range splitByMgr {
		subRet, subResult := mgr.CheckSubItem(ctx, group.data)
		if subResult.IsError() {
			return nil, subResult
		}
		ret = append(ret, subRet...)
	}
	return ret, cd.CreateRpcResultOk()
}

func (u *User) GetItemTypeStatistics(ctx cd.RpcContext, typeId int32) *ItemTypeStatistics {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil
	}

	return mgr.GetTypeStatistics(ctx, typeId)
}

func (u *User) GetItemFromBasic(ctx cd.RpcContext, itemBasic *public_protocol_common.DItemBasic) (*public_protocol_common.DItemInstance, Result) {
	if itemBasic == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	typeId := itemBasic.GetTypeId()
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GetItemFromBasic(ctx, itemBasic)
}

func (u *User) GetNotEnoughErrorCode(typeId int32) int32 {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_INVALID_TYPE_ID)
	}

	return mgr.GetNotEnoughErrorCode(typeId)
}

func (u *User) CheckTypeIdValid(typeId int32) bool {
	mgr := u.GetItemManager(typeId)
	if mgr == nil {
		return false
	}

	return mgr.CheckTypeIdValid(typeId)
}

// 检查期望消耗是否满足配置要求.
func (u *User) CheckCostItemCfg(ctx cd.RpcContext,
	realCost []*public_protocol_common.DItemBasic,
	expectCost []*public_protocol_common.Readonly_DItemOffset,
) Result {
	if len(expectCost) == 0 {
		return cd.CreateRpcResultOk()
	}

	countByTypeId := make(map[int32]int64)
	for _, cost := range realCost {
		typeId := cost.GetTypeId()
		if typeId == 0 || cost.GetCount() <= 0 {
			continue
		}

		countByTypeId[typeId] += cost.GetCount()
	}

	for _, expect := range expectCost {
		typeId := expect.GetTypeId()
		expectCount := expect.GetCount()
		actualCount, exists := countByTypeId[typeId]
		if !exists || actualCount < expectCount {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(u.GetNotEnoughErrorCode(typeId)))
		}
	}

	return cd.CreateRpcResultOk()
}

func (u *User) CheckCostItem(ctx cd.RpcContext,
	realCost []*public_protocol_common.DItemBasic,
	expectCost []*public_protocol_common.DItemOffset,
) Result {
	if len(expectCost) == 0 {
		return cd.CreateRpcResultOk()
	}

	countByTypeId := make(map[int32]int64)
	for _, cost := range realCost {
		typeId := cost.GetTypeId()
		if typeId == 0 || cost.GetCount() <= 0 {
			continue
		}

		countByTypeId[typeId] += cost.GetCount()
	}

	for _, expect := range expectCost {
		typeId := expect.GetTypeId()
		expectCount := expect.GetCount()
		actualCount, exists := countByTypeId[typeId]
		if !exists || actualCount < expectCount {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode(u.GetNotEnoughErrorCode(typeId)))
		}
	}

	return cd.CreateRpcResultOk()
}

// 检查期望消耗是否满足配置要求.
func (u *User) MergeCostItem(expectCost ...[]*public_protocol_common.Readonly_DItemOffset) []*public_protocol_common.Readonly_DItemOffset {
	if len(expectCost) == 0 {
		return nil
	}

	if len(expectCost) == 1 {
		return expectCost[0]
	}

	countByTypeId := make(map[int32]int64)
	for _, costList := range expectCost {
		for _, cost := range costList {
			typeId := cost.GetTypeId()
			if countByTypeId[typeId] <= 0 {
				countByTypeId[typeId] = 0
			}

			countByTypeId[typeId] += cost.GetCount()
		}
	}

	ret := make([]*public_protocol_common.Readonly_DItemOffset, 0, len(countByTypeId))
	for typeId, count := range countByTypeId {
		o := &public_protocol_common.DItemOffset{
			TypeId: typeId,
			Count:  count,
		}
		ret = append(ret, o.ToReadonly())
	}

	return ret
}
