package atframework_component_dispatcher

import (
	"container/list"
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
	currentRunningAction TaskActionImpl
	currentRunningLock   sync.Mutex

	actionStatus   ActorExecutorStatus
	actionLock     sync.Mutex
	pendingActions list.List

	Instance interface{}
}

func CreateActorExecutor(actorInstance interface{}) *ActorExecutor {
	return &ActorExecutor{
		actionStatus:   ActorExecutorStatusFree,
		pendingActions: list.List{},
		Instance:       actorInstance,
	}
}

func (actor *ActorExecutor) getCurrentRunningAction() TaskActionImpl {
	return actor.currentRunningAction
}

func (actor *ActorExecutor) takeCurrentRunningAction(action TaskActionImpl) {
	if lu.IsNil(action) {
		return
	}

	actor.currentRunningLock.Lock()
	actor.currentRunningAction = action
}

func (actor *ActorExecutor) releaseCurrentRunningAction(app libatapp.AppImpl, expectAction TaskActionImpl, spawnNewGoroutine bool) {
	if lu.IsNil(expectAction) {
		return
	}

	if actor.currentRunningAction != expectAction {
		return
	}

	actor.currentRunningAction = nil

	actor.currentRunningLock.Unlock()

	// 释放令牌后允许其他协程并发拉起
	if !spawnNewGoroutine {
		return
	}

	appendActorTaskAction(app, actor, nil, nil)
}
