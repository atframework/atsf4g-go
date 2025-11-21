package atframework_component_router

import (
	"sync/atomic"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type TaskActionRouterCloseManagerSet struct {
	cd.TaskActionNoMessageBase

	manager     *RouterManagerSet
	pendingList []RouterObject
	status      routerCloseStatus
}

type routerCloseStatus struct {
	successCount atomic.Int32
	failedCount  atomic.Int32
	currentIndex int
}

func (s *routerCloseStatus) reset() {
	s.currentIndex = 0
}

func (t *TaskActionRouterCloseManagerSet) Name() string {
	return "TaskActionRouterCloseManagerSet"
}

func (t *TaskActionRouterCloseManagerSet) Run(_startData *cd.DispatcherStartData) error {
	t.status.reset()
	t.LogInfo("router close task started")

	for t.status.currentIndex < len(t.pendingList) {
		obj := t.pendingList[t.status.currentIndex]
		t.status.currentIndex++
		// 批量等待并完成 TODO
		t.processClosingObject(t.GetAwaitableContext(), obj)
	}

	return nil
}

func (t *TaskActionRouterCloseManagerSet) processClosingObject(ctx cd.AwaitableContext, obj RouterObject) {
	if lu.IsNil(obj) {
		return
	}
	if !obj.CheckFlag(FlagIsObject) {
		return
	}
	mgr := t.manager.GetManager(obj.GetKey().TypeID)
	if mgr == nil {
		t.status.failedCount.Add(1)
		t.LogError("router close task missing manager", "object", obj)
		return
	}
	if mgr.GetBaseCache(obj.GetKey()) != obj {
		return
	}
	result := mgr.RemoveObject(ctx, obj.GetKey(), obj, nil, nil)
	if result.IsError() {
		t.status.failedCount.Add(1)
		t.LogError("router close task save router object failed",
			"object", obj, "code", result.GetResponseCode(), "error", result.GetStandardError())
		return
	}

	t.status.successCount.Add(1)
	t.LogInfo("router close task save router object success", "object", obj)
}

func (t *TaskActionRouterCloseManagerSet) saveFallback(ctx cd.AwaitableContext) {
	for t.status.currentIndex < len(t.pendingList) {
		obj := t.pendingList[t.status.currentIndex]
		t.status.currentIndex++
		if lu.IsNil(obj) || !obj.CheckFlag(FlagIsObject) {
			continue
		}
		mgr := t.manager.GetManager(obj.GetKey().TypeID)
		if mgr == nil {
			t.status.failedCount.Add(1)
			t.LogError("router close task fallback missing manager", "object", obj)
			continue
		}
		if mgr.GetBaseCache(obj.GetKey()) != obj {
			continue
		}
		_ = mgr.RemoveObject(ctx, obj.GetKey(), obj, nil, nil)
		t.LogWarn("router close task fallback save issued", "object", obj, "result", "unknown")
	}
}

func (t *TaskActionRouterCloseManagerSet) resetClosingTask() {
	if t.manager.closingTask == t {
		t.manager.closingTask = nil
	}
}

func (t *TaskActionRouterCloseManagerSet) OnSuccess() {
	t.resetClosingTask()
	t.LogInfo("router close task done", "success_count", t.status.successCount.Load(), "failed_count", t.status.failedCount.Load())
}

func (t *TaskActionRouterCloseManagerSet) OnFailed() {
	t.saveFallback(nil)
	t.resetClosingTask()
	t.LogError("router close task failed", "success_count", t.status.successCount.Load(), "failed_count", t.status.failedCount.Load(),
		"response_code", t.GetResponseCode())
}

func (t *TaskActionRouterCloseManagerSet) OnTimeout() {
	t.saveFallback(nil)
	t.resetClosingTask()
	t.LogWarn("router close task timeout, we will continue on next round")
}
