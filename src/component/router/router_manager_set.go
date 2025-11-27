package atframework_component_router

import (
	"container/list"
	"context"
	"fmt"
	"reflect"
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

	timers            TimerSet
	lastProcTime      int64
	mgrs              []RouterManagerBaseImpl
	pendingActionList []PendingActionData

	autoSaveActionTask cd.TaskActionImpl
	closingTask        cd.TaskActionImpl

	isClosing    bool
	isClosed     bool
	isPreClosing atomic.Bool
}

var routerManagerSetReflectType reflect.Type

func init() {
	routerManagerSetReflectType = reflect.TypeOf((*RouterManagerSet)(nil)).Elem()
	var _ libatapp.AppModuleImpl = (*RouterManagerSet)(nil)
}

func GetReflectTypeRouterManagerSet() reflect.Type {
	return routerManagerSetReflectType
}

// CreateRouterManagerSet 创建路由管理器集合
func CreateRouterManagerSet(app libatapp.AppImpl) *RouterManagerSet {
	ret := &RouterManagerSet{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
		timers: TimerSet{
			DefaultTimerList: list.New(),
			FastTimerList:    list.New(),
		},
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
	ret := 0
	now := set.GetApp().GetSysNow().Unix()

	// 如果不是正在关闭,则每秒只需要判定一次
	if !set.IsClosing() && !set.isPreClosing.Load() && set.lastProcTime == now {
		return false
	}

	// 每分钟打印一次统计数据
	if set.lastProcTime/60 != now/60 {
		defaultCount := set.timers.DefaultTimerList.Len()
		fastCount := set.timers.FastTimerList.Len()

		var defaultNext int64
		if defaultCount > 0 {
			defaultNext = set.timers.DefaultTimerList.Front().Value.(*RouterTimer).Timeout
		}

		var fastNext int64
		if fastCount > 0 {
			fastNext = set.timers.FastTimerList.Front().Value.(*RouterTimer).Timeout
		}

		set.GetApp().GetDefaultLogger().Warn(
			fmt.Sprintf("[STATISTICS] router manager set => now: %d, default timer count: %d (next: %d), fast timer count: %d (next: %d)",
				now, defaultCount, defaultNext, fastCount, fastNext),
		)

		// 打印各管理器的缓存数量
		for i := range set.mgrs {
			if set.mgrs[i] != nil {
				set.GetApp().GetDefaultLogger().Warn(fmt.Sprintf("\t%s has %d cache(s)", set.mgrs[i].Name(), set.mgrs[i].Size()))
			}
		}
	}
	set.lastProcTime = now

	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](cd.GetReflectTypeNoMessageDispatcher(), set.GetApp())
	ctx := d.CreateRpcContext()

	// 正在执行closing任务则不需要自动清理/保存了
	if !set.isClosingTaskRunning() {
		cacheExpire := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetCacheFreeTimeout().GetSeconds()
		objectExpire := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetObjectFreeTimeout().GetSeconds()
		objectSave := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetObjectSaveInterval().GetSeconds()

		ret += set.tickTimer(ctx, cacheExpire, objectExpire, objectSave, set.timers.DefaultTimerList, false)
		ret += set.tickTimer(ctx, cacheExpire, objectExpire, objectSave, set.timers.FastTimerList, true)
	}

	if ret != 0 {
		set.GetApp().GetDefaultLogger().Debug("RouterManagerSet Tick processed timers", "count", ret)
	}

	if len(set.pendingActionList) > 0 && !set.IsClosed() && !set.isSaveTaskRunning() && !set.isClosingTaskRunning() {
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

		err := libatapp.AtappGetModule[*cd.TaskManager](cd.GetReflectTypeTaskManager(), ctx.GetApp()).StartTaskAction(ctx, autoSaveTask, &startData)
		if err != nil {
			set.GetApp().GetDefaultLogger().Error("TaskActionAutoSaveObjects StartTaskAction failed", "error", err)
		} else {
			set.autoSaveActionTask = autoSaveTask
		}
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

	timerLists := []*list.List{set.timers.DefaultTimerList, set.timers.FastTimerList}
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

	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](cd.GetReflectTypeNoMessageDispatcher(), set.GetApp())
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
		set.closingTask = closingTask
		cd.AsyncThenStartTask(ctx, nil, set.autoSaveActionTask, set.closingTask, &startData)
	} else {
		err := libatapp.AtappGetModule[*cd.TaskManager](cd.GetReflectTypeTaskManager(), ctx.GetApp()).StartTaskAction(ctx, closingTask, &startData)
		if err != nil {
			set.GetApp().GetDefaultLogger().Error("TaskActionRouterCloseManagerSet StartTaskAction failed", "error", err)
		} else {
			set.closingTask = closingTask
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
		cd.KillTaskAction(ctx, set.closingTask, &result)
	}

	set.closingTask = nil
}

// insertTimer 插入定时器
func (set *RouterManagerSet) insertTimer(ctx cd.RpcContext, mgr RouterManagerBaseImpl, obj RouterObjectImpl, isFast bool) bool {
	if set.lastProcTime <= 0 {
		return false
	}
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

	var tmTimer *list.List
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

	timer.TimerElement = tmTimer.PushBack(timer)
	obj.ResetTimerRef(tmTimer, timer.TimerElement)

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
func (set *RouterManagerSet) AddSaveSchedule(ctx cd.RpcContext, obj RouterObjectImpl) bool {
	if lu.IsNil(obj) {
		return false
	}

	if obj.CheckFlag(FlagSchedSaveObject) {
		return false
	}

	if !obj.IsWritable() {
		return false
	}

	set.pendingActionList = append(set.pendingActionList, PendingActionData{
		Action: AutoSaveActionSave,
		TypeID: obj.GetKey().TypeID,
		Object: obj,
	})
	obj.RefreshSaveTime(ctx)
	obj.SetFlag(FlagSchedSaveObject)
	return true
}

// AddDowngradeSchedule 添加降级计划
func (set *RouterManagerSet) AddDowngradeSchedule(ctx cd.RpcContext, obj RouterObjectImpl) bool {
	if lu.IsNil(obj) {
		return false
	}

	if obj.CheckFlag(FlagSchedRemoveObject) {
		return false
	}

	if !obj.IsWritable() {
		return false
	}

	set.pendingActionList = append(set.pendingActionList, PendingActionData{
		Action: AutoSaveActionRemoveObject,
		TypeID: obj.GetKey().TypeID,
		Object: obj,
	})
	obj.RefreshSaveTime(ctx)
	obj.SetFlag(FlagSchedRemoveObject)
	obj.UnsetFlag(FlagSchedSaveObject)
	return true
}

// MarkFastSave 标记快速保存
func (set *RouterManagerSet) MarkFastSave(ctx cd.RpcContext, mgr RouterManagerBaseImpl, obj RouterObjectImpl) bool {
	if lu.IsNil(obj) || lu.IsNil(mgr) {
		return false
	}

	if !obj.IsWritable() {
		return false
	}

	if obj.CheckFlag(FlagSchedSaveObject) {
		return false
	}

	obj.SetFlag(FlagForceSaveObject)
	if obj.GetTimerList() == set.timers.FastTimerList {
		return false
	}

	return set.insertTimer(ctx, mgr, obj, true)
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
	if lu.IsNil(set.autoSaveActionTask) {
		return false
	}
	if set.autoSaveActionTask.IsExiting() {
		set.autoSaveActionTask = nil
		return false
	}
	return true
}

func (set *RouterManagerSet) isClosingTaskRunning() bool {
	if lu.IsNil(set.closingTask) {
		return false
	}
	if set.closingTask.IsExiting() {
		set.closingTask = nil
		return false
	}
	return true
}

func (set *RouterManagerSet) tickTimer(ctx cd.RpcContext, cacheExpire, objectExpire, objectSave int64, timerList *list.List, isFast bool) int {
	ret := 0
	for {
		if timerList.Len() == 0 {
			break
		}

		timerElem := timerList.Front()
		timer := timerElem.Value.(*RouterTimer)

		// 如果没到时间，后面的全没到时间
		if set.lastProcTime <= timer.Timeout {
			break
		}

		// 如果已下线并且缓存失效则跳过
		obj := timer.ObjWatcher
		if lu.IsNil(obj) {
			timerList.Remove(timerElem)
			continue
		}

		// 如果操作序列失效则跳过
		if !obj.CheckTimerSequence(timer.TimerSequence) {
			obj.CheckAndRemoveTimerRef(timerList, timerElem)
			timerList.Remove(timerElem)
			continue
		}

		// 已销毁则跳过
		mgr := set.GetManager(timer.TypeID)
		if mgr == nil {
			obj.CheckAndRemoveTimerRef(timerList, timerElem)
			timerList.Remove(timerElem)
			continue
		}

		// 管理器中的对象已被替换或移除则跳过
		if mgr.GetBaseCache(obj.GetKey()) != obj {
			obj.CheckAndRemoveTimerRef(timerList, timerElem)
			timerList.Remove(timerElem)
			continue
		}

		isNextTimerFast := isFast // 快队列定时器只能进入快队列
		// 正在执行IO任务则不需要任何流程,因为IO任务结束后可能改变状态
		if !obj.IsIORunning() {
			if obj.CheckFlag(FlagIsObject) {
				// 实体过期
				if obj.GetLastVisitTime()+objectExpire < set.lastProcTime || obj.CheckFlag(FlagForceRemoveObject) {
					set.pendingActionList = append(set.pendingActionList, PendingActionData{
						Action: AutoSaveActionRemoveObject,
						TypeID: timer.TypeID,
						Object: obj,
					})
					obj.SetFlag(FlagSchedRemoveObject)
					obj.UnsetFlag(FlagForceRemoveObject)
				} else if obj.GetLastSaveTime()+objectSave < set.lastProcTime || obj.CheckFlag(FlagForceSaveObject) {
					// 实体保存
					set.pendingActionList = append(set.pendingActionList, PendingActionData{
						Action: AutoSaveActionSave,
						TypeID: timer.TypeID,
						Object: obj,
					})
					obj.RefreshSaveTime(ctx)
					obj.SetFlag(FlagSchedSaveObject)
					obj.UnsetFlag(FlagForceSaveObject)
				}
			} else {
				// 缓存过期
				if obj.GetLastVisitTime()+cacheExpire < set.lastProcTime {
					set.pendingActionList = append(set.pendingActionList, PendingActionData{
						Action: AutoSaveActionRemoveCache,
						TypeID: timer.TypeID,
						Object: obj,
					})
					obj.SetFlag(FlagSchedRemoveCache)
					isNextTimerFast = true
				}
			}
		} else {
			isNextTimerFast = true
		}

		obj.CheckAndRemoveTimerRef(timerList, timerElem)
		set.insertTimer(ctx, mgr, obj, isNextTimerFast)
		timerList.Remove(timerElem)
		ret++
	}

	return ret
}
