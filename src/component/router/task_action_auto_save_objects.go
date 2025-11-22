package atframework_component_router

import (
	"sync/atomic"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type TaskActionAutoSaveObjects struct {
	*cd.TaskActionNoMessageBase

	manager *RouterManagerSet
	status  autoSaveStatus
}

type autoSaveStatus struct {
	successCount            atomic.Int32
	failedCount             atomic.Int32
	actionRemoveObjectCount atomic.Int32
	actionRemoveCacheCount  atomic.Int32
	actionSaveCount         atomic.Int32

	startTime int64
}

func (s *autoSaveStatus) reset(now int64) {
	s.startTime = now
}

func (t *TaskActionAutoSaveObjects) Name() string {
	return "TaskActionAutoSaveObjects"
}

func (t *TaskActionAutoSaveObjects) Run(_startData *cd.DispatcherStartData) error {
	t.status.reset(t.GetSysNow().Unix())
	t.LogInfo("auto save task started")

	left := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetPendingActionMaxCount()
	batchCount := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetPendingActionBatchCount()
	if left == 0 {
		left = uint64(len(t.manager.pendingActionList))
	}

	pendingActionBatchTask := make([]cd.TaskActionImpl, 0, batchCount)
	for left > 0 {
		if len(t.manager.pendingActionList) == 0 {
			break
		}

		pending := t.manager.pendingActionList[0]
		t.manager.pendingActionList = t.manager.pendingActionList[1:]
		left--

		// 批量等待并完成
		taskAction := cd.AsyncInvoke(t.GetRpcContext(), "Execute Pending Action", func(childCtx cd.AwaitableContext) cd.RpcResult {
			result := t.executePendingAction(childCtx, pending)
			t.handleAutoSaveResult(pending, result)
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

	return nil
}

func (t *TaskActionAutoSaveObjects) executePendingAction(ctx cd.AwaitableContext, data PendingActionData) *cd.RpcResult {
	switch data.Action {
	case AutoSaveActionRemoveObject:
		// 有可能在一系列异步流程后又被mutable_object()了，这时候要放弃降级
		if !data.Object.CheckFlag(FlagSchedRemoveObject) {
			return nil
		}
		mgr := t.manager.GetManager(data.TypeID)
		if mgr == nil {
			return nil
		}
		t.status.actionRemoveObjectCount.Add(1)
		result := mgr.RemoveObject(ctx, data.Object.GetKey(), data.Object, nil)
		// 失败且期间未升级或mutable_object()，下次重试的时候也要走降级流程
		if result.IsError() && data.Object.CheckFlag(FlagSchedRemoveObject) {
			data.Object.SetFlag(FlagForceRemoveObject)
		}
		return &result
	case AutoSaveActionSave:
		// 有可能有可能手动触发了保存，导致多一次冗余的auto_save_data_t，就不需要再保存一次了
		if !data.Object.CheckFlag(FlagSchedSaveObject) {
			return nil
		}
		t.status.actionSaveCount.Add(1)
		guard := IoTaskGuard{}
		defer guard.ResumeAwaitTask(ctx)
		result := data.Object.InternalSaveObject(ctx, &guard, nil)
		if result.IsOK() {
			data.Object.RefreshSaveTime(ctx)
		}
		return &result
	case AutoSaveActionRemoveCache:
		// 有可能在一系列异步流程后缓存被续期了，这时候要放弃移除缓存
		if !data.Object.CheckFlag(FlagSchedRemoveCache) {
			return nil
		}
		mgr := t.manager.GetManager(data.TypeID)
		if mgr == nil {
			return nil
		}
		t.status.actionRemoveCacheCount.Add(1)
		result := mgr.RemoveCache(ctx, data.Object.GetKey(), data.Object, nil)
		return &result
	default:
		return nil
	}
}

func (t *TaskActionAutoSaveObjects) handleAutoSaveResult(data PendingActionData, result *cd.RpcResult) {
	actionName := autoSaveActionName(data.Action)
	if result != nil && result.IsError() {
		t.status.failedCount.Add(1)
		t.LogError("auto save action failed", "action", actionName, "object", data.Object,
			"code", result.GetResponseCode(), "error", result.GetStandardError())
		return
	}

	t.status.successCount.Add(1)
	t.LogInfo("auto save action success", "action", actionName, "object", data.Object)
}

func (t *TaskActionAutoSaveObjects) resetAutoSaveTask() {
	if t.manager != nil && t.manager.autoSaveActionTask == t {
		t.manager.autoSaveActionTask = nil
	}
}

func (t *TaskActionAutoSaveObjects) OnSuccess() {
	t.resetAutoSaveTask()
	t.LogWarn("auto save task done", "success_count", t.status.successCount.Load(), "failed_count", t.status.failedCount.Load())
	if t.status.successCount.Load() == 0 && t.status.failedCount.Load() == 0 {
		t.LogWarn("auto save skipped", "reason", "no object requires saving")
	}
}

func (t *TaskActionAutoSaveObjects) OnFailed() {
	t.resetAutoSaveTask()
	t.LogError("auto save task failed", "success_count", t.status.successCount.Load(), "failed_count", t.status.failedCount.Load(),
		"response_code", t.GetResponseCode())
}

func (t *TaskActionAutoSaveObjects) OnTimeout() {
	t.resetAutoSaveTask()
	elapsed := t.GetSysNow().Unix() - t.status.startTime
	if elapsed < 0 {
		elapsed = 0
	}
	t.LogWarn("auto save task timeout, we will continue on next round", "elapsed_seconds", elapsed)
}

func autoSaveActionName(action AutoSaveActionType) string {
	switch action {
	case AutoSaveActionSave:
		return "save"
	case AutoSaveActionRemoveObject:
		return "remove object"
	case AutoSaveActionRemoveCache:
		return "remove cache"
	default:
		return "unknown action name"
	}
}
