// Copyright 2026 atframework
// Translated from task_action_player_remote_patch_jobs.h/.cpp

package lobbysvr_logic_async_jobs_action

import (
	"time"

	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_async_jobs "github.com/atframework/atsf4g-go/service-lobbysvr/logic/async_jobs"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
)

// UserAsyncJobsManagerForTask 异步任务需要的 UserAsyncJobsManager 接口
type UserAsyncJobsManagerForTask interface {
	ClearJobUuids(jobType int32)
	AddJobUuid(jobType int32, uuid string)
	IsJobUuidExists(jobType int32, uuid string) bool
	AddRetryJob(jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData)
	RemoveRetryJob(jobType int32, uuid string)
	GetRetryJobs(jobType int32) []*private_protocol_pbdesc.UserAsyncJobsBlobData
	ResetAsyncJobsProtect()
}

// SyncCallback 同步回调函数类型
type SyncCallback func(ctx cd.RpcContext, user *data.User, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) int32

// AsyncCallback 异步回调函数类型
type AsyncCallback func(ctx cd.RpcContext, user *data.User, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) cd.RpcResult

// TaskActionPlayerRemotePatchJobs 远程补丁任务
type TaskActionPlayerRemotePatchJobs struct {
	cd.TaskActionNoMessageBase

	user             *data.User
	manager          UserAsyncJobsManagerForTask
	asyncJobTypes    map[int32]struct{}
	timeoutDuration  time.Duration
	timeoutTimepoint time.Time

	needRestart      bool
	isWritable       bool
	patchedJobNumber int

	syncCallbacks  map[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID]SyncCallback
	asyncCallbacks map[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID]AsyncCallback
}

// CreateTaskActionPlayerRemotePatchJobs 创建远程补丁任务
func CreateTaskActionPlayerRemotePatchJobs(
	base cd.TaskActionNoMessageBase,
	user *data.User,
	manager UserAsyncJobsManagerForTask,
	asyncJobTypes map[int32]struct{},
	timeoutDuration time.Duration,
) *TaskActionPlayerRemotePatchJobs {
	t := &TaskActionPlayerRemotePatchJobs{
		TaskActionNoMessageBase: base,
		user:                    user,
		manager:                 manager,
		asyncJobTypes:           asyncJobTypes,
		timeoutDuration:         timeoutDuration,
		timeoutTimepoint:        time.Now().Add(timeoutDuration),
		needRestart:             false,
		isWritable:              false,
		patchedJobNumber:        0,
		syncCallbacks:           make(map[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID]SyncCallback),
		asyncCallbacks:          make(map[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID]AsyncCallback),
	}
	t.registerCallbacks()
	return t
}

func (t *TaskActionPlayerRemotePatchJobs) Name() string {
	return "TaskActionPlayerRemotePatchJobs"
}

func (t *TaskActionPlayerRemotePatchJobs) Run(_startData *cd.DispatcherStartData) error {
	t.needRestart = false
	t.isWritable = false
	t.patchedJobNumber = 0

	if t.user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return nil
	}

	ctx := t.GetAwaitableContext()

	if !t.user.IsWriteable() {
		ctx.LogDebug("TaskActionPlayerRemotePatchJobs: user is not writable, skip")
		return nil
	}

	t.isWritable = true

	jobTypes := t.getAllAsyncJobTypes()

	for _, jobType := range jobTypes {
		if !t.isWritable {
			break
		}

		if len(t.asyncJobTypes) > 0 {
			if _, ok := t.asyncJobTypes[jobType]; !ok {
				continue
			}
		}

		ctx.LogDebug("TaskActionPlayerRemotePatchJobs: processing job type", "job_type", jobType)

		batchJobNumber := t.processJobsFromDB(ctx, jobType)
		t.patchedJobNumber += batchJobNumber

		retryJobNumber := t.processRetryJobs(ctx, jobType)
		t.patchedJobNumber += retryJobNumber

		t.manager.ClearJobUuids(jobType)

		if time.Until(t.timeoutTimepoint) < t.timeoutDuration/2 {
			t.needRestart = true
			break
		}

		t.isWritable = t.user.IsWriteable()
	}

	return nil
}

// getAllAsyncJobTypes 获取所有异步任务类型
func (t *TaskActionPlayerRemotePatchJobs) getAllAsyncJobTypes() []int32 {
	// TODO: 从 EnPlayerAsyncJobsType 枚举获取所有类型
	return []int32{
		int32(private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_PAY),
		int32(private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_NORMAL),
		int32(private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_MAIL_IMPORTANT),
		int32(private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_MAIL_SYSTEM),
	}
}

// processJobsFromDB 从数据库处理任务
func (t *TaskActionPlayerRemotePatchJobs) processJobsFromDB(ctx cd.AwaitableContext, jobType int32) int {
	if t.user == nil {
		return 0
	}

	userId := t.user.GetUserId()
	zoneId := t.user.GetZoneId()

	indexMessages, result := db.DatabaseTableUserAsyncJobsLoadAllWithJobTypeUserIdZoneId(
		ctx,
		jobType,
		userId,
		zoneId,
	)
	if result.IsError() {
		if result.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			ctx.LogError("processJobsFromDB: load async jobs failed",
				"job_type", jobType,
				"user_id", userId,
				"zone_id", zoneId,
				"error", result)
		}
		return 0
	}

	if len(indexMessages) == 0 {
		return 0
	}

	processedCount := 0
	var processedIndexes []uint64

	for _, indexMsg := range indexMessages {
		if indexMsg.Table == nil {
			continue
		}

		if !t.user.IsWriteable() {
			t.isWritable = false
			break
		}

		table, ok := indexMsg.Table.(*private_protocol_pbdesc.DatabaseTableUserAsyncJobs)
		if !ok || table == nil {
			continue
		}

		jobData := table.GetJobData()
		if jobData == nil {
			processedIndexes = append(processedIndexes, indexMsg.ListIndex)
			continue
		}

		actionUuid := jobData.GetActionUuid()

		if t.manager.IsJobUuidExists(jobType, actionUuid) {
			ctx.LogDebug("processJobsFromDB: job already processed, skip",
				"job_type", jobType,
				"action_uuid", actionUuid)
			processedIndexes = append(processedIndexes, indexMsg.ListIndex)
			continue
		}

		execResult := t.doJob(ctx, jobType, jobData)
		if execResult < 0 {
			ctx.LogError("processJobsFromDB: do async action failed",
				"action_case", jobData.GetAction(),
				"result", execResult,
				"action_uuid", actionUuid)

			if jobData.GetLeftRetryTimes() > 0 {
				t.manager.AddRetryJob(jobType, jobData)
			}
		} else {
			t.manager.AddJobUuid(jobType, actionUuid)
			processedCount++
		}
		processedIndexes = append(processedIndexes, indexMsg.ListIndex)

		if time.Until(t.timeoutTimepoint) < t.timeoutDuration/2 {
			t.needRestart = true
			break
		}
	}

	if len(processedIndexes) > 0 {
		delResult := db.DatabaseTableUserAsyncJobsDelIndexWithJobTypeUserIdZoneId(
			ctx,
			processedIndexes,
			jobType,
			userId,
			zoneId,
		)
		if delResult.IsError() {
			ctx.LogError("processJobsFromDB: delete processed jobs failed",
				"job_type", jobType,
				"user_id", userId,
				"zone_id", zoneId,
				"indexes", processedIndexes,
				"error", delResult)
		}
	}

	return processedCount
}

// processRetryJobs 处理重试队列中的任务
func (t *TaskActionPlayerRemotePatchJobs) processRetryJobs(ctx cd.RpcContext, jobType int32) int {
	retryJobs := t.manager.GetRetryJobs(jobType)
	if len(retryJobs) == 0 {
		return 0
	}

	processedCount := 0
	for _, jobData := range retryJobs {
		if jobData == nil {
			continue
		}

		if !t.user.IsWriteable() {
			t.isWritable = false
			break
		}

		result := t.doJob(ctx, jobType, jobData)
		if result < 0 {
			ctx.LogError("do async action failed",
				"action_case", jobData.GetAction(),
				"result", result)

			if jobData.GetLeftRetryTimes() > 0 {
				jobData.LeftRetryTimes = jobData.GetLeftRetryTimes() - 1
				t.manager.AddRetryJob(jobType, jobData)
			} else {
				t.manager.RemoveRetryJob(jobType, jobData.GetActionUuid())
			}
		} else {
			t.manager.RemoveRetryJob(jobType, jobData.GetActionUuid())
			processedCount++
		}
	}

	return processedCount
}

// doJob 执行单个任务
func (t *TaskActionPlayerRemotePatchJobs) doJob(ctx cd.RpcContext, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) int32 {
	if jobData == nil {
		return 0
	}

	actionCase := jobData.GetActionOneofCase()

	if callback, ok := t.syncCallbacks[actionCase]; ok && callback != nil {
		return callback(ctx, t.user, jobType, jobData)
	}

	if callback, ok := t.asyncCallbacks[actionCase]; ok && callback != nil {
		result := callback(ctx, t.user, jobType, jobData)
		return result.GetResponseCode()
	}

	ctx.LogError("do invalid async action", "action_case", actionCase)
	return 0
}

// registerCallbacks 注册回调函数
func (t *TaskActionPlayerRemotePatchJobs) registerCallbacks() {
	t.syncCallbacks[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID_DebugMessage] = func(ctx cd.RpcContext, user *data.User, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) int32 {
		debugMsg := jobData.GetDebugMessage()
		if debugMsg != nil {
			ctx.LogInfo("[TODO] do async action debug_message",
				"title", debugMsg.GetTitle(),
				"content", debugMsg.GetContent())
		}
		return 0
	}

	// 添加邮件
	t.syncCallbacks[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID_AddMail] = func(ctx cd.RpcContext, user *data.User, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) int32 {
		mailMsg := jobData.GetAddMail()
		if mailMsg != nil {
			ctx.LogInfo("do async action add_mail",
				"mail_record", mailMsg.GetMailRecord())
		}

		mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
		if mailMgr == nil {
			ctx.LogError("do async action add_mail failed: user mail manager is nil")
			return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return mailMgr.AddMail(ctx, mailMsg.GetMailRecord(), nil)
	}

	// 删除邮件
	t.syncCallbacks[private_protocol_pbdesc.UserAsyncJobsBlobData_EnActionID_RemoveMail] = func(ctx cd.RpcContext, user *data.User, jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) int32 {
		mailMsg := jobData.GetRemoveMail()
		if mailMsg != nil {
			ctx.LogInfo("do async action remove_mail",
				"mail_record", mailMsg.GetMailRecord())
		}

		mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
		if mailMgr == nil {
			ctx.LogError("do async action remove_mail failed: user mail manager is nil")
			return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		ret := mailMgr.RemoveMail(ctx, mailMsg.GetMailRecord().GetMailId(), nil)
		if ret == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			return 0
		}
		return ret
	}
}

func (t *TaskActionPlayerRemotePatchJobs) OnSuccess() {
	ctx := t.GetRpcContext()
	ctx.LogDebug("TaskActionPlayerRemotePatchJobs success",
		"patched_job_number", t.patchedJobNumber)

	if t.isWritable && t.user != nil {
		// 如果需要重启，重置保护时间
		if t.needRestart {
			t.manager.ResetAsyncJobsProtect()
		}

		mailMgr := data.UserGetModuleManager[logic_async_jobs.UserAsyncJobsManager](t.user)
		if mailMgr == nil {
			ctx.LogError("do async action add_mail failed: user mail manager is nil")
			return
		}
		mailMgr.TryAsyncJobs(t.GetRpcContext())
	}

	if t.patchedJobNumber > 0 {
		t.user.SendAllSyncData(ctx)
	}
}

func (t *TaskActionPlayerRemotePatchJobs) OnFailed() {
	ctx := t.GetRpcContext()
	ctx.LogError("TaskActionPlayerRemotePatchJobs failed",
		"patched_job_number", t.patchedJobNumber,
		"result", t.GetResponseCode())

	if t.isWritable && t.user != nil {
		if t.needRestart {
			t.manager.ResetAsyncJobsProtect()
		}
	}

	if t.patchedJobNumber > 0 {
		t.user.SendAllSyncData(ctx)
	}
}

func (t *TaskActionPlayerRemotePatchJobs) OnTimeout() {
	ctx := t.GetRpcContext()
	ctx.LogError("TaskActionPlayerRemotePatchJobs timeout",
		"patched_job_number", t.patchedJobNumber)
}
