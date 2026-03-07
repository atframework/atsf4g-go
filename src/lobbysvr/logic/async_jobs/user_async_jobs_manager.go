// Copyright 2026 atframework
// Translated from user_async_jobs_manager.h/.cpp

package lobbysvr_logic_async_jobs

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// UserAsyncJobsManager 用户异步任务管理器接口
type UserAsyncJobsManager interface {
	data.UserModuleManagerImpl

	// IsDirty 检查是否有脏数据
	IsDirty() bool

	// ClearDirty 清除脏标记
	ClearDirty()

	// IsAsyncJobsTaskRunning 检查异步任务是否正在运行
	IsAsyncJobsTaskRunning() bool

	// TryAsyncJobs 尝试启动异步任务
	// 返回 true 表示成功启动，false 表示已有任务正在运行或保护时间未到
	TryAsyncJobs(ctx cd.RpcContext) bool

	// WaitForAsyncTask 等待异步任务完成
	WaitForAsyncTask(ctx cd.RpcContext) cd.RpcResult

	// ForceAsyncJob 强制执行指定类型的异步任务
	ForceAsyncJob(jobsType int32)

	// ResetAsyncJobsProtect 重置异步任务保护时间
	ResetAsyncJobsProtect()

	// ClearJobUuids 清理指定类型的任务UUID历史记录
	ClearJobUuids(jobType int32)

	// AddJobUuid 添加任务UUID到历史记录
	AddJobUuid(jobType int32, uuid string)

	// IsJobUuidExists 检查任务UUID是否存在
	IsJobUuidExists(jobType int32, uuid string) bool

	// AddRetryJob 添加重试任务
	AddRetryJob(jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData)

	// RemoveRetryJob 移除重试任务
	RemoveRetryJob(jobType int32, uuid string)

	// GetRetryJobs 获取指定类型的重试任务列表
	GetRetryJobs(jobType int32) []*private_protocol_pbdesc.UserAsyncJobsBlobData

	// GetOwner 获取所属用户
	GetOwner() *data.User
}
