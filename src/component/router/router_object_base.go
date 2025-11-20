package atframework_component_router

import (
	"container/list"
	"fmt"
	"runtime"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type RouterObjectKey struct {
	TypeID   uint32
	ZoneID   uint32
	ObjectID uint64
}

type FlagType int32

const (
	FlagForcePullObject FlagType = 0x0001 // 下一次mutable_object时是否强制执行数据拉取
	FlagIsObject        FlagType = 0x0002 // 当前对象是否是实体（可写）
	FlagForceSaveObject FlagType = 0x0004 // 下一次触发定时器时是否强制执行数据保存

	FlagCacheRemoved      FlagType = 0x0008 // 当前对象缓存是否已处于实体被移除的状态，缓存被移除意味着已经不在manager的管理中，但是可能临时存在于部分正在进行的任务里
	FlagSaving            FlagType = 0x0010 // 是否正在保存
	FlagTransfering       FlagType = 0x0020 // 是否正在进行数据转移
	FlagPullingCache      FlagType = 0x0040 // 是否正在拉取对象缓存
	FlagPullingObject     FlagType = 0x0080 // 是否正在拉取对象实体
	FlagSchedRemoveObject FlagType = 0x0100 // 定时任务 - 实体降级计划任务是否有效
	FlagSchedRemoveCache  FlagType = 0x0200 // 定时任务 - 移除缓存计划任务是否有效
	FlagSchedSaveObject   FlagType = 0x0400 // 定时任务 - 实体保存计划任务是否有效
	FlagForceRemoveObject FlagType = 0x0800 // 下一次触发定时器时是否强制执行实体降级
	FlagRemovingCache     FlagType = 0x1000 // 是否正在移除对象缓存
	FlagRemovingObject    FlagType = 0x2000 // 是否正在移除对象实体
)

type RouterObjectBase struct {
	impl RouterObject // 实现对象接口

	key   RouterObjectKey // 对象的键
	flags int32           // 标志位

	lastSaveTime  int64  // 最后一次保存时间
	lastVisitTime int64  // 最后一次访问时间
	routerSvrID   uint64 // 路由服务器ID
	routerSvrName string // 路由服务器名称
	routerSvrVer  uint64 // 路由服务器版本

	savingSequence      uint64            // 保存序列
	savedSequence       uint64            // 已保存序列
	awaitTaskActionImpl cd.TaskActionImpl // 正在等待的IO任务ID
	awaitIOTaskList     list.List         // 正在等待IO任务的任务列表

	// 定时器相关
	timerSequence uint64        // 定时器序列号
	timerList     *list.List    // 定时器列表
	timerElement  *list.Element // 定时器元素
}

type FlagGuard struct {
	obj  *RouterObjectBase
	flag FlagType
}

func NewFlagGuard(obj *RouterObjectBase, flag FlagType) *FlagGuard {
	guard := &FlagGuard{obj: obj, flag: flag}
	if obj.CheckFlag(flag) {
		return guard
	}
	obj.SetFlag(flag)
	return guard
}

func (g *FlagGuard) Release() {
	g.obj.UnsetFlag(g.flag)
}

type RouterPrivateData interface {
}

type RouterObjectBaseImpl interface {
	GetRouterObjectBase() *RouterObjectBase
	RefreshVisitTime(ctx cd.RpcContext)
	RefreshSaveTime(ctx cd.RpcContext)
	GetLastVisitTime() int64
	GetLastSaveTime() int64
	UnsetFlag(flag FlagType)
	SetFlag(flag FlagType)
	CheckFlag(flag FlagType) bool
	IsWritable() bool
	IsPullingCache() bool
	IsCacheAvailable(ctx cd.RpcContext) bool
	IsObjectAvailable() bool
	Downgrade(ctx cd.RpcContext)
	Upgrade(ctx cd.RpcContext)
	GetRouterSvrId() uint64
	GetRouterSvrVer() uint64
	GetKey() RouterObjectKey
	GetRouterSvrName() string

	// 任务调度相关
	GetAwaitTaskId() uint64
	SetAwaitTaskAction(taskActionImpl cd.TaskActionImpl)
	AwaitIOTask(ctx cd.AwaitableContext) cd.RpcResult
	ResumeAwaitTask(ctx cd.RpcContext)
	IsIORunning() bool

	RemoveObject(ctx cd.AwaitableContext, transferToSvrId uint64, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult
	InternalPullCache(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult
	InternalPullObject(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult
	InternalSaveObject(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult

	// 定时器相关
	AllocTimerSequence() uint64
	CheckTimerSequence(seq uint64) bool
	ResetTimerRef(timerList *list.List, timerElem *list.Element)
	CheckAndRemoveTimerRef(timerList *list.List, timerElem *list.Element)
	GetTimerList() *list.List
	UnsetTimerRef()
}

type RouterObject interface {
	PullCache(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult // 存在默认实现
	PullObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	SaveObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	RouterObjectBaseImpl
}

type IoTaskGuard struct {
	awaitTaskId uint64
	owner       RouterObject
}

func (g *IoTaskGuard) Take(ctx cd.AwaitableContext, obj RouterObject) cd.RpcResult {
	if ctx.GetAction() == nil {
		return cd.CreateRpcResultError(fmt.Errorf("no task"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	// 已被调用者接管，则忽略子层接管
	if ctx.GetAction().GetTaskId() == obj.GetAwaitTaskId() {
		return cd.CreateRpcResultOk()
	}

	ret := obj.AwaitIOTask(ctx)
	if ret.IsError() {
		return ret
	}

	if 0 == ctx.GetAction().GetTaskId() {
		return cd.CreateRpcResultOk()
	}

	g.owner = obj
	g.awaitTaskId = ctx.GetAction().GetTaskId()
	obj.SetAwaitTaskAction(ctx.GetAction())

	return cd.CreateRpcResultOk()
}

func (g *IoTaskGuard) ResumeAwaitTask(ctx cd.RpcContext) {
	if g.awaitTaskId == 0 {
		return
	}
	g.awaitTaskId = 0

	if lu.IsNil(g.owner) {
		return
	}

	// IO任务被抢占
	if g.awaitTaskId != g.owner.GetAwaitTaskId() {
		return
	}

	g.owner.ResumeAwaitTask(ctx)
}

func CreateRouterObjectBase(ctx cd.RpcContext, key RouterObjectKey) *RouterObjectBase {
	ret := &RouterObjectBase{
		key: key,
	}
	ret.RefreshVisitTime(ctx)
	runtime.SetFinalizer(ret, func(obj *RouterObjectBase) {
		obj.UnsetTimerRef()
	})
	return ret
}

func (obj *RouterObjectBase) GetRouterObjectBase() *RouterObjectBase {
	return obj
}

func (obj *RouterObjectBase) RefreshVisitTime(ctx cd.RpcContext) {
	obj.lastVisitTime = ctx.GetSysNow().Unix()

	// 刷新访问事件要取消移除缓存的计划任务
	obj.UnsetFlag(FlagRemovingCache)
}

func (obj *RouterObjectBase) RefreshSaveTime(ctx cd.RpcContext) {
	obj.lastSaveTime = ctx.GetSysNow().Unix()
}

func (obj *RouterObjectBase) GetLastVisitTime() int64 {
	return obj.lastVisitTime
}

func (obj *RouterObjectBase) GetLastSaveTime() int64 {
	return obj.lastSaveTime
}

func (obj *RouterObjectBase) UnsetFlag(flag FlagType) {
	obj.flags &= ^int32(flag)
}

func (obj *RouterObjectBase) SetFlag(flag FlagType) {
	obj.flags |= int32(flag)
}

func (obj *RouterObjectBase) CheckFlag(flag FlagType) bool {
	return (obj.flags & int32(flag)) == int32(flag)
}

func (obj *RouterObjectBase) IsWritable() bool {
	return obj.CheckFlag(FlagIsObject) && !obj.CheckFlag(FlagForcePullObject) &&
		!obj.CheckFlag(FlagCacheRemoved)
}

func (obj *RouterObjectBase) IsPullingCache() bool {
	return obj.CheckFlag(FlagPullingCache)
}

func (obj *RouterObjectBase) IsCacheAvailable(ctx cd.RpcContext) bool {
	if obj.IsPullingCache() {
		return false
	}

	if obj.IsWritable() {
		return true
	}

	if obj.lastSaveTime+
		config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetCacheUpdateInterval().GetSeconds() < ctx.GetSysNow().Unix() {
		return false
	}

	return true
}

func (obj *RouterObjectBase) IsObjectAvailable() bool {
	if obj.CheckFlag(FlagPullingObject) {
		return false
	}

	return obj.IsWritable()
}

func (obj *RouterObjectBase) Upgrade(ctx cd.RpcContext) {
	if obj.CheckFlag(FlagIsObject) {
		return
	}

	obj.RefreshVisitTime(ctx)
	obj.SetFlag(FlagIsObject)
	obj.UnsetFlag(FlagCacheRemoved)

	// 升级操作要取消移除缓存和降级的计划任务
	obj.UnsetFlag(FlagForceRemoveObject)
	obj.UnsetFlag(FlagSchedRemoveObject)
	obj.UnsetFlag(FlagSchedRemoveCache)
}

func (obj *RouterObjectBase) Downgrade(ctx cd.RpcContext) {
	if !obj.CheckFlag(FlagIsObject) {
		return
	}

	obj.RefreshVisitTime(ctx)
	obj.UnsetFlag(FlagIsObject)
}

func (obj *RouterObjectBase) GetRouterSvrId() uint64 {
	return obj.routerSvrID
}

func (obj *RouterObjectBase) GetKey() RouterObjectKey {
	return obj.key
}

func (obj *RouterObjectBase) GetRouterSvrName() string {
	return obj.routerSvrName
}

func (obj *RouterObjectBase) GetRouterSvrVer() uint64 {
	return obj.routerSvrVer
}

func (obj *RouterObjectBase) GetAwaitTaskId() uint64 {
	if lu.IsNil(obj.awaitTaskActionImpl) {
		return 0
	}
	if obj.awaitTaskActionImpl.IsExiting() {
		obj.awaitTaskActionImpl = nil
		return 0
	}
	return obj.awaitTaskActionImpl.GetTaskId()
}

func (obj *RouterObjectBase) SetAwaitTaskAction(awaitTaskActionImpl cd.TaskActionImpl) {
	obj.awaitTaskActionImpl = awaitTaskActionImpl
}

func (obj *RouterObjectBase) AwaitIOTask(ctx cd.AwaitableContext) cd.RpcResult {
	// 可重入式的等待器
	if obj.GetAwaitTaskId() == 0 || obj.GetAwaitTaskId() == ctx.GetAction().GetTaskId() {
		return cd.CreateRpcResultOk()
	}

	for obj.GetAwaitTaskId() != 0 && obj.GetAwaitTaskId() != ctx.GetAction().GetTaskId() {
		// if
		e := obj.awaitIOTaskList.PushBack(ctx.GetAction())
		_, _ = cd.YieldTaskAction(ctx.GetApp(), ctx.GetAction(), &cd.DispatcherAwaitOptions{
			Type:     uint64(uintptr(unsafe.Pointer(obj))),
			Sequence: ctx.GetAction().GetTaskId(),
			Timeout:  config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetTask().GetCsmsg().LogValue().Duration(),
		}, nil)
		obj.awaitIOTaskList.Remove(e)
	}

	return cd.CreateRpcResultOk()
}

func (obj *RouterObjectBase) ResumeAwaitTask(ctx cd.RpcContext) {
	// 可重入式的等待器
	var failedTask cd.TaskActionImpl
	for obj.awaitIOTaskList.Len() > 0 {
		if obj.GetAwaitTaskId() == 0 {
			break
		}

		e := obj.awaitIOTaskList.Front()
		taskAction := e.Value.(cd.TaskActionImpl)

		if !taskAction.IsExiting() && failedTask != taskAction {
			err := cd.ResumeTaskAction(ctx.GetApp(), taskAction, &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: uint64(uintptr(unsafe.Pointer(obj))),
				},
				Sequence: taskAction.GetTaskId(),
			})
			if err != nil {
				ctx.GetApp().GetDefaultLogger().Error("Resume await IO task action failed", "error", err)
				failedTask = taskAction
			} else {
				failedTask = nil
			}
		} else {
			obj.awaitIOTaskList.Remove(e)
		}
	}
}

func (obj *RouterObjectBase) IsIORunning() bool {
	return obj.GetAwaitTaskId() != 0
}

func (obj *RouterObjectBase) PullCache(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult {
	// 默认实现
	return obj.impl.PullObject(ctx, privateData)
}

func (obj *RouterObjectBase) RemoveObject(ctx cd.AwaitableContext, transferToSvrId uint64, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult {
	ret := guard.Take(ctx, obj.impl)
	if ret.IsError() {
		return ret
	}

	// 在等待其他任务的时候已经完成移除或降级，直接成功即可
	if !obj.IsWritable() {
		return cd.CreateRpcResultOk()
	}

	// 移除实体需要设置路由BUS ID为0并保存一次
	oldRouterServerId := obj.GetRouterSvrId()
	oldRouterVer := obj.GetRouterSvrVer()

	if transferToSvrId != oldRouterServerId {
		obj.routerSvrID = transferToSvrId
		obj.routerSvrVer = oldRouterVer + 1
	}
	obj.RefreshVisitTime(ctx)

	if !obj.IsWritable() {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_WRITABLE)
	}

	ret = obj.InternalSaveObject(ctx, guard, privateData)
	if ret.IsError() {
		// 保存失败则恢复原来的路由信息
		obj.routerSvrID = oldRouterServerId
		obj.routerSvrVer = oldRouterVer
		return ret
	}

	obj.Downgrade(ctx)
	return cd.CreateRpcResultOk()
}

func (obj *RouterObjectBase) InternalPullCache(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult {
	// 触发拉取缓存时要取消移除缓存的计划任务
	obj.UnsetFlag(FlagSchedRemoveCache)

	if lu.IsNil(ctx.GetAction()) || ctx.GetAction().GetTaskId() == 0 {
		ctx.LogError("show in task")
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	ret := guard.Take(ctx, obj.impl)
	if ret.IsError() {
		return ret
	}

	flagGuard := NewFlagGuard(obj, FlagPullingCache)
	defer flagGuard.Release()

	ret = obj.PullCache(ctx, privateData)
	if ret.IsError() {
		return ret
	}

	// 拉取成功要刷新保存时间
	obj.RefreshSaveTime(ctx)
	return cd.CreateRpcResultOk()
}

func (obj *RouterObjectBase) InternalPullObject(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult {
	// 触发拉取实体时要取消移除缓存和降级的计划任务
	obj.UnsetFlag(FlagForceRemoveObject)
	obj.UnsetFlag(FlagSchedRemoveObject)
	obj.UnsetFlag(FlagSchedRemoveCache)

	if lu.IsNil(ctx.GetAction()) || ctx.GetAction().GetTaskId() == 0 {
		ctx.LogError("show in task")
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	ret := guard.Take(ctx, obj.impl)
	if ret.IsError() {
		return ret
	}

	// 其他任务中已经拉取成功并已经升级为实体且未失效，直接视为成功
	if obj.IsWritable() {
		return cd.CreateRpcResultOk()
	}

	flagGuard := NewFlagGuard(obj, FlagPullingObject)
	defer flagGuard.Release()

	// 清除缓存移除和强制拉取标记
	obj.UnsetFlag(FlagCacheRemoved)
	obj.UnsetFlag(FlagForcePullObject)

	// 执行拉取实体
	ret = obj.impl.PullObject(ctx, privateData)
	if ret.IsError() {
		return ret
	}

	// 拉取成功要刷新保存时间
	obj.RefreshSaveTime(ctx)

	// 检查路由服务器ID
	if obj.GetRouterSvrId() != 0 {
		// 从config获取本地服务器ID进行比较
		if uint64(ctx.GetApp().GetLogicId()) != obj.GetRouterSvrId() {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_WRITABLE)
		}
	}

	// 升级为实体
	obj.Upgrade(ctx)
	return cd.CreateRpcResultOk()
}

func (obj *RouterObjectBase) InternalSaveObject(ctx cd.AwaitableContext, guard *IoTaskGuard, privateData RouterPrivateData) cd.RpcResult {
	obj.UnsetFlag(FlagSchedSaveObject)

	// 排队写任务和并发写merge
	obj.savingSequence = obj.savingSequence + 1
	thisSavingSeq := obj.savingSequence

	if lu.IsNil(ctx.GetAction()) || ctx.GetAction().GetTaskId() == 0 {
		ctx.LogError("show in task")
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	ret := guard.Take(ctx, obj.impl)
	if ret.IsError() {
		return ret
	}

	if !obj.IsWritable() {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_WRITABLE)
	}

	// 因为可能叠加和被其他任务抢占，所以这里需要核查一遍保存序号
	// 如果其他任务的保存涵盖了自己的数据变化，则直接成功返回
	if obj.savedSequence >= thisSavingSeq {
		return cd.CreateRpcResultOk()
	}

	flagGuard := NewFlagGuard(obj, FlagSaving)
	defer flagGuard.Release()

	realSavingSeq := obj.savingSequence

	// 执行保存
	result := obj.impl.SaveObject(ctx, privateData)
	if result.IsError() {
		return result
	}

	if realSavingSeq > obj.savedSequence {
		obj.savedSequence = realSavingSeq
	}

	// 刷新保存时间
	obj.RefreshSaveTime(ctx)
	return cd.CreateRpcResultOk()
}

// ResetTimerRef 重置定时器引用
func (obj *RouterObjectBase) ResetTimerRef(timerList *list.List, timerElem *list.Element) {
	if obj.timerList == timerList && obj.timerElement == timerElem {
		return
	}
	obj.UnsetTimerRef()
	obj.timerList = timerList
	obj.timerElement = timerElem
}

// CheckAndRemoveTimerRef 检查并移除定时器引用
func (obj *RouterObjectBase) CheckAndRemoveTimerRef(timerList *list.List, timerElem *list.Element) {
	if obj.timerList == timerList && obj.timerElement == timerElem {
		// 内部接口，会在外层执行timer_list_->erase(timer_iter_);所以这里不执行移除
		obj.timerList = nil
		obj.timerElement = nil
	}
}

func (obj *RouterObjectBase) UnsetTimerRef() {
	if obj.timerList != nil && obj.timerElement != nil {
		obj.timerList.Remove(obj.timerElement)
	}
	obj.timerList = nil
	obj.timerElement = nil
}

// AllocTimerSequence 分配定时器序列号
func (obj *RouterObjectBase) AllocTimerSequence() uint64 {
	obj.timerSequence++
	return obj.timerSequence
}

// CheckTimerSequence 检查定时器序列号
func (obj *RouterObjectBase) CheckTimerSequence(seq uint64) bool {
	return obj.timerSequence == seq
}

// GetTimerList 获取定时器列表
func (obj *RouterObjectBase) GetTimerList() *list.List {
	return obj.timerList
}
