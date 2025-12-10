package atframework_component_dispatcher

import (
	"container/list"
	"log/slog"
	"sync"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	libatapp "github.com/atframework/libatapp-go"
)

type ActorExecutorStatus int8

const (
	ActorExecutorStatusFree ActorExecutorStatus = iota // 0
	ActorExecutorStatusPending
)

type ActorExecutor struct {
	currentRunningAction lu.AtomicInterface[TaskActionImpl]
	currentRunningLock   sync.Mutex

	actionStatus   ActorExecutorStatus
	actionLock     sync.Mutex
	pendingActions list.List

	Instance libatapp.LogAttr
}

func CreateActorExecutor(actorInstance libatapp.LogAttr) *ActorExecutor {
	return &ActorExecutor{
		actionStatus:   ActorExecutorStatusFree,
		pendingActions: list.List{},
		Instance:       actorInstance,
	}
}

func (actor *ActorExecutor) getCurrentRunningAction() TaskActionImpl {
	return actor.currentRunningAction.Load()
}

func (actor *ActorExecutor) takeCurrentRunningAction(action TaskActionImpl) {
	if lu.IsNil(action) {
		return
	}

	actor.currentRunningLock.Lock()
	actor.currentRunningAction.Store(action)
}

func (actor *ActorExecutor) releaseCurrentRunningAction(app libatapp.AppImpl, expectAction TaskActionImpl, spawnNewGoroutine bool) {
	if lu.IsNil(expectAction) {
		return
	}

	if !actor.currentRunningAction.CompareAndSwap(expectAction, nil) {
		return
	}

	actor.currentRunningLock.Unlock()

	// 释放令牌后允许其他协程并发拉起
	if !spawnNewGoroutine {
		return
	}

	appendActorTaskAction(app, actor, nil, nil)
}

func (actor *ActorExecutor) CheckActorExecutor(ctx RpcContext) bool {
	if actor == nil || lu.IsNil(ctx) {
		ctx.LogError("CheckActorExecutor failed: actor or ctx is nil")
		return false
	}

	if lu.IsNil(ctx.GetAction()) {
		// 没有在任务内
		ctx.LogError("CheckActorExecutor failed: action is nil")
		return false
	}

	if lu.IsNil(ctx.GetAction().GetActorExecutor()) {
		// 没有Actor
		ctx.LogError("CheckActorExecutor failed: action actor is nil")
		return false
	}

	if actor.Instance != ctx.GetAction().GetActorExecutor().Instance {
		ctx.LogError("CheckActorExecutor failed: actor instance mismatch")
		return false
	}

	return true
}

func (actor *ActorExecutor) TryTakeCurrentRunningAction(action TaskActionImpl) bool {
	if lu.IsNil(action) {
		return false
	}

	if action.GetActorExecutor() != nil && actor.Instance == action.GetActorExecutor().Instance {
		// 已经是当前执行者
		return true
	}

	actor.currentRunningLock.Lock()
	actor.currentRunningAction.Store(action)
	action.SetActorExecutor(actor)
	return true
}

func (actor *ActorExecutor) LogAttr() []slog.Attr {
	if actor == nil || lu.IsNil(actor.Instance) {
		return nil
	}
	return actor.Instance.LogAttr()
}
