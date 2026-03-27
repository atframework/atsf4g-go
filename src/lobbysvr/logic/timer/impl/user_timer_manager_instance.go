package lobbysvr_logic_timer_internal

import (
	"time"

	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	libatapp "github.com/atframework/libatapp-go"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_timer "github.com/atframework/atsf4g-go/service-lobbysvr/logic/timer"
)

// 确保实现接口
func init() {
	var _ logic_timer.UserTimerManager = (*UserTimerManager)(nil)
	data.RegisterUserModuleManagerCreator[logic_timer.UserTimerManager](func(ctx cd.RpcContext, owner *data.User) data.UserModuleManagerImpl {
		return CreateUserTimerManager(ctx.GetApp(), owner)
	})
}

// UserTimerManager 用户定时器管理器实现
type UserTimerManager struct {
	data.UserModuleManagerBase
	timer *time.Timer

	nextTriggerTimestamp int64
	triggerTimestamps    []int64
}

func CreateUserTimerManager(app libatapp.AppImpl, owner *data.User) *UserTimerManager {
	return &UserTimerManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
		nextTriggerTimestamp:  0,
	}
}

// SetTimer 设置/更新用户定时器，传入触发时间戳（Unix 秒）。
// 仅当新时间比当前注册时间更早（或当前无 timer）时才会重新创建 time.Timer。
func (m *UserTimerManager) SetTimer(ctx cd.RpcContext, triggerTimestamp int64) {
	if triggerTimestamp <= 0 {
		return
	}

	owner := m.GetOwner()
	if owner == nil {
		return
	}

	if triggerTimestamp == m.nextTriggerTimestamp {
		return
	}

	// 插入 但是不更新 timer
	if m.nextTriggerTimestamp != 0 && triggerTimestamp > m.nextTriggerTimestamp {
		m.triggerTimestamps = append(m.triggerTimestamps, triggerTimestamp)
		return
	}

	m.nextTriggerTimestamp = triggerTimestamp

	// 停止旧 timer
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}

	delay := time.Until(time.Unix(triggerTimestamp, 0))
	if delay < 0 {
		delay = 0
	}

	m.timer = time.AfterFunc(delay, func() {
		if !owner.IsWriteable() {
			return
		}
		cd.AsyncInvoke(ctx, "Timer Tick", owner.GetActorExecutor(), func(childCtx cd.AwaitableContext) cd.RpcResult {
			timerMgr := data.UserGetModuleManager[logic_timer.UserTimerManager](owner)
			if timerMgr == nil {
				return cd.CreateRpcResultOk()
			}
			timerMgr.Tick(ctx)
			return cd.CreateRpcResultOk()
		})
	})
}

// cancelTimer 停止并清除用户定时器
func (m *UserTimerManager) cancelTimer() {
	m.nextTriggerTimestamp = 0
	m.triggerTimestamps = m.triggerTimestamps[:0]
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

func (m *UserTimerManager) Tick(ctx cd.RpcContext) {
	// timer 已触发，清除记录
	m.timer = nil

	if m.nextTriggerTimestamp <= 0 {
		return
	}

	owner := m.GetOwner()
	if owner == nil {
		return
	}

	now := ctx.GetNow()
	if now.Unix() < m.nextTriggerTimestamp {
		m.SetTimer(ctx, m.nextTriggerTimestamp)
		return
	}

	m.nextTriggerTimestamp = 0

	// 遍历定时器 清除过期定时器
	validTimestamps := m.triggerTimestamps[:0]
	for _, ts := range m.triggerTimestamps {
		if ts > now.Unix() {
			validTimestamps = append(validTimestamps, ts)
			if m.nextTriggerTimestamp == 0 || ts < m.nextTriggerTimestamp {
				m.nextTriggerTimestamp = ts
			}
		}
	}
	m.triggerTimestamps = validTimestamps

	// 调用 User.RefreshLimit 触发所有模块的 RefreshLimit / RefreshLimitSecond / RefreshLimitMinute
	owner.RefreshLimit(ctx, now)
}

// OnLogout 用户下线时取消定时器
func (m *UserTimerManager) OnLogout(_ cd.RpcContext) {
	m.cancelTimer()
}
