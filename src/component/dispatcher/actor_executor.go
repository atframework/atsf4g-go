package atframework_component_dispatcher

import (
	"container/list"
	"sync"
	"sync/atomic"

	libatapp "github.com/atframework/libatapp-go"
)

type ActorExecutorStatus int8

const (
	ActorExecutorStatusFree ActorExecutorStatus = iota // 0
	ActorExecutorStatusPending
)

type ActorExecutor struct {
	currentRunningAction atomic.Value
	currentRunningLock   sync.Mutex

	actionStatus   ActorExecutorStatus
	actionLock     sync.Mutex
	pendingActions list.List

	Instance interface{}
}

func CreateActorExecutor(actorInstance interface{}) ActorExecutor {
	return ActorExecutor{
		currentRunningAction: atomic.Value{},
		actionStatus:         ActorExecutorStatusFree,
		pendingActions:       list.List{},
		Instance:             actorInstance,
	}
}

func (actor *ActorExecutor) getCurrentRunningAction() TaskActionImpl {
	result := actor.currentRunningAction.Load()
	if result == nil {
		return nil
	}

	return result.(TaskActionImpl)
}

func (actor *ActorExecutor) takeCurrentRunningAction(action TaskActionImpl) {
	if action == nil {
		return
	}

	actor.currentRunningLock.Lock()
	actor.currentRunningAction.Store(action)
}

func (actor *ActorExecutor) releaseCurrentRunningAction(app libatapp.AppImpl, expectAction TaskActionImpl, spawnNewGoroutine bool) {
	if expectAction == nil {
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
