package atframework_component_router

import (
	"container/list"
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

// AutoSaveActionType 自动保存操作类型
type AutoSaveActionType int32

const (
	AutoSaveActionSave         AutoSaveActionType = 0 // 保存
	AutoSaveActionRemoveObject AutoSaveActionType = 1 // 移除实体
	AutoSaveActionRemoveCache  AutoSaveActionType = 2 // 移除缓存
)

// PendingActionData 待处理操作数据
type PendingActionData struct {
	Action AutoSaveActionType
	TypeID uint32
	Object RouterObjectImpl
}

// RouterManagerSet 路由管理器集合
type RouterManagerSet struct {
	libatapp.AppModuleBase

	timers       TimerSet
	lastProcTime int64
	mgrs         []RouterManagerBaseImpl

	pendingActionListLock sync.Mutex
	pendingActionList     []PendingActionData

	taskPendingActionListLock sync.Mutex
	taskPendingActionList     *list.List

	autoSaveActionTask lu.AtomicInterface[cd.TaskActionImpl]
	closingTask        lu.AtomicInterface[cd.TaskActionImpl]

	isClosing    bool
	isClosed     bool
	isPreClosing atomic.Bool
}

var routerManagerSetReflectType reflect.Type

func init() {
	var _ libatapp.AppModuleImpl = (*RouterManagerSet)(nil)
	routerManagerSetReflectType = lu.GetStaticReflectType[RouterManagerSet]()
}

// CreateRouterManagerSet 创建路由管理器集合
func CreateRouterManagerSet(app libatapp.AppImpl) *RouterManagerSet {
	ret := &RouterManagerSet{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
		timers: TimerSet{
			DefaultTimerList: NewTimerList(),
			FastTimerList:    NewTimerList(),
		},
		taskPendingActionList: list.New(),
	}

	// 假设最大类型数为256
	ret.mgrs = make([]RouterManagerBaseImpl, 256)
	return ret
}

func (m *RouterManagerSet) GetReflectType() reflect.Type {
	return routerManagerSetReflectType
}

func (m *RouterManagerSet) Init(parent context.Context) error {
	// 注册Task
	return nil
}

func (m *RouterManagerSet) Name() string { return "RouterManagerSet" }

// Tick 定时处理
func (set *RouterManagerSet) Tick(parent context.Context) bool {
	now := set.GetApp().GetSysNow().Unix()

	// 如果不是正在关闭,则每秒只需要判定一次
	if !set.IsClosing() && !set.isPreClosing.Load() && set.lastProcTime == now {
		return false
	}

	set.timers.DefaultTimerList.DoPending()
	set.timers.FastTimerList.DoPending()

	// 每分钟打印一次统计数据
	if set.lastProcTime/60 != now/60 {
		defaultCount := set.timers.DefaultTimerList.list.Len()
		fastCount := set.timers.FastTimerList.list.Len()

		var defaultNext int64
		if defaultCount > 0 {
			defaultNext = set.timers.DefaultTimerList.list.Front().Value.(*RouterTimer).Timeout
		}

		var fastNext int64
		if fastCount > 0 {
			fastNext = set.timers.FastTimerList.list.Front().Value.(*RouterTimer).Timeout
		}

		set.GetApp().GetDefaultLogger().LogWarn(
			fmt.Sprintf("[STATISTICS] router manager set => now: %d, default timer count: %d (next: %d), fast timer count: %d (next: %d)",
				now, defaultCount, defaultNext, fastCount, fastNext),
		)

		// 打印各管理器的缓存数量
		for i := range set.mgrs {
			if set.mgrs[i] != nil {
				set.GetApp().GetDefaultLogger().LogWarn(fmt.Sprintf("\t%s has %d cache(s)", set.mgrs[i].Name(), set.mgrs[i].Size()))
			}
		}
	}
	set.lastProcTime = now

	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](set.GetApp())
	ctx := d.CreateRpcContext()

	// 正在执行closing任务则不需要自动清理/保存了
	if !set.isClosingTaskRunning() {
		cacheExpire := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetCacheFreeTimeout().GetSeconds()
		objectExpire := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetObjectFreeTimeout().GetSeconds()
		objectSave := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetObjectSaveInterval().GetSeconds()

		set.tickTimer(ctx, cacheExpire, objectExpire, objectSave, set.timers.DefaultTimerList, false)
		set.tickTimer(ctx, cacheExpire, objectExpire, objectSave, set.timers.FastTimerList, true)
	}

	// 启动保存任务
	set.pendingActionListLock.Lock()
	if len(set.pendingActionList) > 0 && !set.IsClosed() && !set.isSaveTaskRunning() && !set.isClosingTaskRunning() {
		// 处理列表
		set.taskPendingActionListLock.Lock()
		for _, v := range set.pendingActionList {
			set.taskPendingActionList.PushBack(v)
		}
		set.taskPendingActionListLock.Unlock()
		set.pendingActionList = set.pendingActionList[:0]
		set.pendingActionListLock.Unlock()

		// 创建 AutoSave 任务
		autoSaveTask, startData := cd.CreateNoMessageTaskAction(
			d, ctx, nil,
			func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *TaskActionAutoSaveObjects {
				ta := TaskActionAutoSaveObjects{
					TaskActionNoMessageBase: cd.CreateNoMessageTaskActionBase(rd, actor, timeout),
					manager:                 set,
				}
				return &ta
			},
		)

		err := libatapp.AtappGetModule[*cd.TaskManager](ctx.GetApp()).StartTaskAction(ctx, autoSaveTask, &startData)
		if err != nil {
			set.GetApp().GetDefaultLogger().LogError("TaskActionAutoSaveObjects StartTaskAction failed", "error", err)
		} else {
			set.autoSaveActionTask.Store(autoSaveTask)
		}
	} else {
		set.pendingActionListLock.Unlock()
	}

	if set.IsClosing() && !set.isClosingTaskRunning() {
		set.isClosed = true
	}

	return false
}

func (set *RouterManagerSet) Stop() (bool, error) {
	if set.IsClosing() {
		return true, nil
	}
	set.isClosing = true

	// 准备启动清理任务
	// 收集所有待保存的对象
	recheckSet := make(map[RouterObjectKey]struct{})
	pendingList := make([]RouterObjectImpl, 0)

	set.timers.DefaultTimerList.DoPending()
	set.timers.FastTimerList.DoPending()

	timerLists := []*list.List{set.timers.DefaultTimerList.list, set.timers.FastTimerList.list}
	for _, curList := range timerLists {
		for e := curList.Front(); e != nil; e = e.Next() {
			timer := e.Value.(*RouterTimer)

			obj := timer.ObjWatcher
			if obj == nil {
				continue
			}

			if !obj.CheckFlag(FlagIsObject) {
				continue
			}

			if _, exists := recheckSet[obj.GetKey()]; exists {
				continue
			}
			recheckSet[obj.GetKey()] = struct{}{}
			pendingList = append(pendingList, obj)
		}
	}

	// 清理所有管理器
	for i := range set.mgrs {
		if set.mgrs[i] != nil {
			set.mgrs[i].OnStop()
		}
	}

	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](set.GetApp())
	ctx := d.CreateRpcContext()

	// 创建并启动closing_task
	closingTask, startData := cd.CreateNoMessageTaskAction(
		d, ctx, nil,
		func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *TaskActionRouterCloseManagerSet {
			ta := TaskActionRouterCloseManagerSet{
				TaskActionNoMessageBase: cd.CreateNoMessageTaskActionBase(rd, actor, timeout),
				manager:                 set,
				pendingList:             pendingList,
			}
			return &ta
		},
	)

	if set.isSaveTaskRunning() {
		// 需要等待Save结束后再启动
		set.closingTask.Store(closingTask)
		cd.AsyncThenStartTask(ctx, nil, set.autoSaveActionTask.Load(), closingTask, &startData)
	} else {
		err := libatapp.AtappGetModule[*cd.TaskManager](ctx.GetApp()).StartTaskAction(ctx, closingTask, &startData)
		if err != nil {
			set.GetApp().GetDefaultLogger().LogError("TaskActionRouterCloseManagerSet StartTaskAction failed", "error", err)
		} else {
			set.closingTask.Store(closingTask)
		}
	}

	return false, nil
}

// ForceClose 强制关闭
func (set *RouterManagerSet) ForceClose(ctx cd.RpcContext) {
	if !set.IsClosing() || set.IsClosed() {
		return
	}

	// 强制停止清理任务
	if set.isClosingTaskRunning() {
		result := cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
		cd.KillTaskAction(ctx, set.closingTask.Load(), &result)
	}

	set.closingTask.Store(nil)
}

// insertTimer 插入定时器 多线程操作
func (set *RouterManagerSet) insertTimer(ctx cd.RpcContext, mgr RouterManagerBaseImpl, obj RouterObjectImpl, isFast bool) bool {
	if lu.IsNil(obj) || lu.IsNil(mgr) {
		return false
	}

	if set.IsClosing() {
		return false
	}

	checkedMgr := set.GetManager(mgr.GetTypeID())
	if checkedMgr != mgr {
		ctx.LogError("router_manager_set has registered, but try to setup timer", "type_id", mgr.GetTypeID(), "registered_mgr",
			lu.IsNil(checkedMgr), "name", mgr.Name())
		return false
	}

	var tmTimer *TimerList
	var interval int64
	if !isFast {
		tmTimer = set.timers.DefaultTimerList
		interval = config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetDefaultTimerInterval().GetSeconds()
	} else {
		tmTimer = set.timers.FastTimerList
		interval = config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetFastTimerInterval().GetSeconds()
	}

	timer := &RouterTimer{
		ObjWatcher: obj,
		TypeID:     mgr.GetTypeID(),

		Timeout:       ctx.GetSysNow().Unix() + interval,
		TimerSequence: obj.AllocTimerSequence(),
		TimerList:     tmTimer,
	}

	timer.TimerElement.Store(tmTimer.PushBack(timer))
	obj.ResetTimerRef(tmTimer, timer.TimerElement.Load())

	return true
}

// GetManager 获取管理器
func (set *RouterManagerSet) GetManager(typeID uint32) RouterManagerBaseImpl {
	if int(typeID) >= len(set.mgrs) {
		return nil
	}
	return set.mgrs[typeID]
}

// RegisterManager 注册管理器
func (set *RouterManagerSet) RegisterManager(mgr RouterManagerBaseImpl) error {
	if mgr == nil {
		return fmt.Errorf("manager is nil")
	}

	typeID := mgr.GetTypeID()
	if int(typeID) >= len(set.mgrs) {
		return fmt.Errorf("router %s has invalid type id %d", mgr.Name(), typeID)
	}

	if set.mgrs[typeID] != nil {
		return fmt.Errorf("router %s has type conflict with %s", set.mgrs[typeID].Name(), mgr.Name())
	}

	set.mgrs[typeID] = mgr

	if set.IsClosing() {
		mgr.OnStop()
	}

	return nil
}

// Size 获取总缓存数量
func (set *RouterManagerSet) Size() int {
	ret := 0
	for i := range set.mgrs {
		if set.mgrs[i] != nil {
			ret += set.mgrs[i].Size()
		}
	}
	return ret
}

// AddSaveSchedule 添加保存计划
func (set *RouterManagerSet) AddSaveSchedule(ctx cd.RpcContext, obj RouterObjectImpl) {
	if lu.IsNil(obj) {
		return
	}

	obj.PushActorAction(ctx, "RouterManagerSet AddSaveSchedule", func(childCtx cd.AwaitableContext, childObj RouterObjectImpl) {
		if childObj.CheckFlag(FlagSchedSaveObject) {
			return
		}

		if !childObj.IsWritable() {
			return
		}

		set.pendingActionListLock.Lock()
		set.pendingActionList = append(set.pendingActionList, PendingActionData{
			Action: AutoSaveActionSave,
			TypeID: childObj.GetKey().TypeID,
			Object: childObj,
		})
		set.pendingActionListLock.Unlock()
		childObj.RefreshSaveTime(childCtx)
		childObj.SetFlag(FlagSchedSaveObject)
	})
}

// AddDowngradeSchedule 添加降级计划
func (set *RouterManagerSet) AddDowngradeSchedule(ctx cd.RpcContext, obj RouterObjectImpl) {
	if lu.IsNil(obj) {
		return
	}

	obj.PushActorAction(ctx, "RouterManagerSet AddDowngradeSchedule", func(childCtx cd.AwaitableContext, childObj RouterObjectImpl) {
		if childObj.CheckFlag(FlagSchedRemoveObject) {
			return
		}

		if !childObj.IsWritable() {
			return
		}

		set.pendingActionListLock.Lock()
		set.pendingActionList = append(set.pendingActionList, PendingActionData{
			Action: AutoSaveActionRemoveObject,
			TypeID: childObj.GetKey().TypeID,
			Object: childObj,
		})
		set.pendingActionListLock.Unlock()

		childObj.RefreshSaveTime(childCtx)
		childObj.SetFlag(FlagSchedRemoveObject)
		childObj.UnsetFlag(FlagSchedSaveObject)
	})
}

// MarkFastSave 标记快速保存
func (set *RouterManagerSet) MarkFastSave(ctx cd.RpcContext, mgr RouterManagerBaseImpl, obj RouterObjectImpl) {
	if lu.IsNil(obj) || lu.IsNil(mgr) {
		return
	}

	obj.PushActorAction(ctx, "RouterManagerSet MarkFastSave", func(childCtx cd.AwaitableContext, childObj RouterObjectImpl) {
		if !childObj.IsWritable() {
			return
		}

		if childObj.CheckFlag(FlagSchedSaveObject) {
			return
		}

		childObj.SetFlag(FlagForceSaveObject)
		if childObj.GetTimerList() == set.timers.FastTimerList {
			return
		}

		set.insertTimer(childCtx, mgr, childObj, true)
	})

}

// IsClosing 是否正在关闭
func (set *RouterManagerSet) IsClosing() bool {
	return set.isClosing
}

// IsClosed 是否已关闭
func (set *RouterManagerSet) IsClosed() bool {
	return set.isClosed
}

// SetPreClosing 设置预关闭状态
func (set *RouterManagerSet) SetPreClosing() {
	set.isPreClosing.Store(true)
}

// 私有方法
func (set *RouterManagerSet) isSaveTaskRunning() bool {
	if lu.IsNil(set.autoSaveActionTask.Load()) {
		return false
	}
	if set.autoSaveActionTask.Load().IsExiting() {
		set.autoSaveActionTask.Store(nil)
		return false
	}
	return true
}

func (set *RouterManagerSet) isClosingTaskRunning() bool {
	if lu.IsNil(set.closingTask.Load()) {
		return false
	}
	if set.closingTask.Load().IsExiting() {
		set.closingTask.Store(nil)
		return false
	}
	return true
}

func (set *RouterManagerSet) tickTimer(ctx cd.RpcContext, cacheExpire, objectExpire, objectSave int64, timerList *TimerList, isFast bool) {
	for {
		if timerList.list.Len() == 0 {
			break
		}

		timerElem := timerList.list.Front()
		timer := timerElem.Value.(*RouterTimer)

		// 如果没到时间，后面的全没到时间
		if set.lastProcTime <= timer.Timeout {
			break
		}
		timerList.list.Remove(timerElem)

		// 如果已下线并且缓存失效则跳过
		obj := timer.ObjWatcher
		if lu.IsNil(obj) {
			continue
		}

		obj.PushActorAction(ctx, "RouterManagerSet TimerTick", func(childCtx cd.AwaitableContext, childObj RouterObjectImpl) {
			// 如果操作序列失效则跳过
			if !childObj.CheckTimerSequence(timer.TimerSequence) {
				childObj.CheckAndRemoveTimerRef(timerList, timer.TimerElement.Load())
				return
			}

			// 已销毁则跳过
			mgr := set.GetManager(timer.TypeID)
			if mgr == nil {
				childObj.CheckAndRemoveTimerRef(timerList, timer.TimerElement.Load())
				return
			}

			// 管理器中的对象已被替换或移除则跳过
			if mgr.GetBaseCache(childObj.GetKey()) != childObj {
				childObj.CheckAndRemoveTimerRef(timerList, timer.TimerElement.Load())
				return
			}

			isNextTimerFast := isFast // 快队列定时器只能进入快队列
			// 正在执行IO任务则不需要任何流程,因为IO任务结束后可能改变状态
			if !childObj.IsIORunning() {
				if childObj.CheckFlag(FlagIsObject) {
					// 实体过期
					if childObj.GetLastVisitTime()+objectExpire < set.lastProcTime || childObj.CheckFlag(FlagForceRemoveObject) {
						set.pendingActionListLock.Lock()
						set.pendingActionList = append(set.pendingActionList, PendingActionData{
							Action: AutoSaveActionRemoveObject,
							TypeID: timer.TypeID,
							Object: childObj,
						})
						set.pendingActionListLock.Unlock()
						childObj.SetFlag(FlagSchedRemoveObject)
						childObj.UnsetFlag(FlagForceRemoveObject)
					} else if childObj.GetLastSaveTime()+objectSave < set.lastProcTime || childObj.CheckFlag(FlagForceSaveObject) {
						// 实体保存
						set.pendingActionListLock.Lock()
						set.pendingActionList = append(set.pendingActionList, PendingActionData{
							Action: AutoSaveActionSave,
							TypeID: timer.TypeID,
							Object: childObj,
						})
						set.pendingActionListLock.Unlock()
						childObj.RefreshSaveTime(childCtx)
						childObj.SetFlag(FlagSchedSaveObject)
						childObj.UnsetFlag(FlagForceSaveObject)
					}
				} else {
					// 缓存过期
					if childObj.GetLastVisitTime()+cacheExpire < set.lastProcTime {
						set.pendingActionListLock.Lock()
						set.pendingActionList = append(set.pendingActionList, PendingActionData{
							Action: AutoSaveActionRemoveCache,
							TypeID: timer.TypeID,
							Object: childObj,
						})
						set.pendingActionListLock.Unlock()
						childObj.SetFlag(FlagSchedRemoveCache)
						isNextTimerFast = true
					}
				}
			} else {
				isNextTimerFast = true
			}

			childObj.CheckAndRemoveTimerRef(timerList, timer.TimerElement.Load())
			set.insertTimer(childCtx, mgr, childObj, isNextTimerFast)
		})
	}
}
