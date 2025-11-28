package atframework_component_router

import (
	"sync/atomic"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type TaskActionRouterCloseManagerSet struct {
	cd.TaskActionNoMessageBase

	manager     *RouterManagerSet
	pendingList []RouterObjectImpl
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

	batchCount := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetClosingActionBatchCount()
	pendingActionBatchTask := make([]cd.TaskActionImpl, 0, batchCount)

	for t.status.currentIndex < len(t.pendingList) {
		obj := t.pendingList[t.status.currentIndex]
		t.status.currentIndex++

		// 批量等待并完成
		taskAction := cd.AsyncInvoke(t.GetRpcContext(), "TaskActionRouterCloseManagerSet Execute Closing Action", obj.GetActorExecutor(), func(childCtx cd.AwaitableContext) cd.RpcResult {
			t.processClosingObject(childCtx, obj)
			return cd.CreateRpcResultOk()
		})

		if !lu.IsNil(taskAction) && !taskAction.IsExiting() {
			pendingActionBatchTask = append(pendingActionBatchTask, taskAction)
		}

		if len(pendingActionBatchTask) >= int(batchCount) {
			result := cd.AwaitTasks(t.GetAwaitableContext(), pendingActionBatchTask)
			clear(pendingActionBatchTask)
			if result.IsError() {
				t.LogError("Wait sub tasks to failed", "result", result)
			}
		}
	}

	if len(pendingActionBatchTask) != 0 {
		result := cd.AwaitTasks(t.GetAwaitableContext(), pendingActionBatchTask)
		clear(pendingActionBatchTask)
		if result.IsError() {
			t.LogError("Wait sub tasks to failed", "result", result)
		}
	}

	// 如果超时了可能被强杀，这时候要强制触发保存
	if t.IsExiting() {
		t.saveFallback(t.GetAwaitableContext())
	}

	return nil
}

func (t *TaskActionRouterCloseManagerSet) processClosingObject(ctx cd.AwaitableContext, obj RouterObjectImpl) {
	// 已降级或不是实体，不需要保存
	if !obj.CheckFlag(FlagIsObject) {
		return
	}
	mgr := t.manager.GetManager(obj.GetKey().TypeID)
	if mgr == nil {
		t.status.failedCount.Add(1)
		t.LogError("router close task missing manager", "object", obj)
		return
	}
	// 管理器中的对象已被替换或移除则跳过
	if mgr.GetBaseCache(obj.GetKey()) != obj {
		return
	}
	// 降级的时候会保存
	result := mgr.InnerRemoveObject(ctx, obj.GetKey(), obj, nil)
	if t.IsFault() || t.IsTimeout() {
		t.status.failedCount.Add(1)
		t.LogError("router close task save router object failed",
			"object", obj, "code", result.GetResponseCode(), "error", result.GetStandardError())
		return
	}

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
		// 已降级或不是实体，不需要保存
		if !obj.CheckFlag(FlagIsObject) {
			continue
		}
		mgr := t.manager.GetManager(obj.GetKey().TypeID)
		if mgr == nil {
			t.status.failedCount.Add(1)
			t.LogError("router close task fallback missing manager", "object", obj)
			continue
		}
		// 管理器中的对象已被替换或移除则跳过
		if mgr.GetBaseCache(obj.GetKey()) != obj {
			continue
		}
		mgr.InnerRemoveObject(ctx, obj.GetKey(), obj, nil)
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
	t.resetClosingTask()
	t.LogError("router close task failed", "success_count", t.status.successCount.Load(), "failed_count", t.status.failedCount.Load(),
		"response_code", t.GetResponseCode())
}

func (t *TaskActionRouterCloseManagerSet) OnTimeout() {
	t.resetClosingTask()
	t.LogWarn("router close task timeout, we will continue on next round")
}
