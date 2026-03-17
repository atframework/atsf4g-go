package lobbysvr_logic_async_jobs_internal

import (
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_async_jobs "github.com/atframework/atsf4g-go/service-lobbysvr/logic/async_jobs"
	async_jobs_action "github.com/atframework/atsf4g-go/service-lobbysvr/logic/async_jobs/action"
	async_jobs_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/async_jobs/data"
)

func init() {
	var _ logic_async_jobs.UserAsyncJobsManager = (*UserAsyncJobsManager)(nil)
	data.RegisterUserModuleManagerCreator[logic_async_jobs.UserAsyncJobsManager](func(_ cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserAsyncJobsManager(owner)
	})
}

// UserAsyncJobsManager 用户异步任务管理器实现
type UserAsyncJobsManager struct {
	data.UserModuleManagerBase

	isDirty bool

	remoteCommandPatchTask              lu.AtomicInterface[cd.TaskActionImpl]
	remoteCommandPatchTaskNextTimepoint time.Time

	historyUuids async_jobs_data.HistoryByJobType

	forceAsyncJobTypes map[int32]struct{}

	retryJobs async_jobs_data.RetryJobsByType
}

func CreateUserAsyncJobsManager(owner *data.User) *UserAsyncJobsManager {
	mgr := &UserAsyncJobsManager{
		UserModuleManagerBase:               *data.CreateUserModuleManagerBase(owner),
		isDirty:                             false,
		remoteCommandPatchTaskNextTimepoint: time.Time{},
		historyUuids:                        make(async_jobs_data.HistoryByJobType),
		forceAsyncJobTypes:                  make(map[int32]struct{}),
		retryJobs:                           make(async_jobs_data.RetryJobsByType),
	}
	return mgr
}

func (m *UserAsyncJobsManager) GetOwner() *data.User {
	return m.UserModuleManagerBase.GetOwner()
}

// ========== UserModuleManagerImpl 接口实现 ==========

func (m *UserAsyncJobsManager) CreateInit(_ctx cd.RpcContext, _versionType uint32) {
}

func (m *UserAsyncJobsManager) LoginInit(_ctx cd.RpcContext) {
	m.ResetAsyncJobsProtect()
}

func (m *UserAsyncJobsManager) RefreshLimitSecond(ctx cd.RpcContext) {
	m.TryAsyncJobs(ctx)
}

func (m *UserAsyncJobsManager) InitFromDB(ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	m.historyUuids = make(async_jobs_data.HistoryByJobType)
	m.retryJobs = make(async_jobs_data.RetryJobsByType)
	m.isDirty = false

	asyncJobBlobData := dbUser.GetAsyncJobBlobData()
	if asyncJobBlobData == nil {
		return cd.RpcResult{Error: nil, ResponseCode: 0}
	}

	asyncJobs := asyncJobBlobData.GetAsyncJobs()
	if asyncJobs != nil {
		// 加载下一次任务激活时间
		nextActiveTime := asyncJobs.GetNextTaskActiveTime()
		if nextActiveTime > 0 {
			m.remoteCommandPatchTaskNextTimepoint = time.Unix(nextActiveTime, 0)
		}

		// 加载历史记录
		for _, history := range asyncJobs.GetHistory() {
			jobType := history.GetJobType()
			if _, ok := m.historyUuids[jobType]; !ok {
				m.historyUuids[jobType] = make(async_jobs_data.HistoryMap)
			}

			item := &async_jobs_data.HistoryItem{
				Uuid:       history.GetActionUuid(),
				Timeout:    time.Unix(history.GetTimeout(), 0),
				InsertTime: time.Now(),
			}
			m.historyUuids[jobType][history.GetActionUuid()] = item
		}
	}

	// 加载重试任务
	for _, retryJob := range asyncJobBlobData.GetRetryJobs() {
		jobType := retryJob.GetJobType()
		if _, ok := m.retryJobs[jobType]; !ok {
			m.retryJobs[jobType] = make(async_jobs_data.RetryJobMap)
		}

		jobData := retryJob.GetJobData()
		if jobData != nil {
			m.retryJobs[jobType][jobData.GetActionUuid()] = jobData.Clone()
		}
	}

	// 清理过期的历史记录
	for jobType := range m.historyUuids {
		m.ClearJobUuids(jobType)
	}

	return cd.RpcResult{Error: nil, ResponseCode: 0}
}

func (m *UserAsyncJobsManager) DumpToDB(ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	asyncJobBlobData := dbUser.MutableAsyncJobBlobData()
	asyncJobs := asyncJobBlobData.MutableAsyncJobs()

	now := time.Now()

	// 保存下一次任务激活时间
	asyncJobs.NextTaskActiveTime = m.remoteCommandPatchTaskNextTimepoint.Unix()

	// 保存历史记录
	for jobType, historyMap := range m.historyUuids {
		for uuid, item := range historyMap {
			if item == nil {
				continue
			}
			// 跳过已过期的记录
			if item.Timeout.Before(now) {
				continue
			}

			historyData := &private_protocol_pbdesc.UserAsyncJobsDataHistoryJobMeta{
				JobType:    jobType,
				ActionUuid: uuid,
				Timeout:    item.Timeout.Unix(),
			}
			asyncJobs.History = append(asyncJobs.History, historyData)
		}
	}

	// 保存重试任务
	for jobType, retryMap := range m.retryJobs {
		for _, jobData := range retryMap {
			if jobData == nil {
				continue
			}

			retryData := &private_protocol_pbdesc.UserAsyncJobsCacheBlobDataRetryJobData{
				JobType: jobType,
				JobData: jobData.Clone(),
			}
			asyncJobBlobData.RetryJobs = append(asyncJobBlobData.RetryJobs, retryData)
		}
	}

	return cd.RpcResult{Error: nil, ResponseCode: 0}
}

func (m *UserAsyncJobsManager) IsDirty() bool {
	return m.isDirty
}

func (m *UserAsyncJobsManager) ClearDirty() {
	m.isDirty = false
}

// IsAsyncJobsTaskRunning 检查异步任务是否正在运行
func (m *UserAsyncJobsManager) IsAsyncJobsTaskRunning() bool {
	task := m.remoteCommandPatchTask.Load()
	if lu.IsNil(task) {
		return false
	}
	if task.IsExiting() {
		m.remoteCommandPatchTask.Store(nil)
		return false
	}
	return true
}

// TryAsyncJobs 尝试启动异步任务
func (m *UserAsyncJobsManager) TryAsyncJobs(ctx cd.RpcContext) bool {
	owner := m.GetOwner()
	if owner == nil {
		return false
	}

	// 保护时间检查
	now := ctx.GetNow()
	if now.Before(m.remoteCommandPatchTaskNextTimepoint) && len(m.forceAsyncJobTypes) == 0 {
		return false
	}

	// 只允许一个任务进行
	if m.IsAsyncJobsTaskRunning() {
		return false
	}

	// 玩家临时性登出，暂时也不需要 patch 数据
	if !owner.IsWriteable() {
		return false
	}

	session := owner.GetSession()
	if session == nil {
		return false
	}

	m.isDirty = true

	// 更新下一次任务时间
	m.forceAsyncJobTypes = make(map[int32]struct{})
	interval := m.getAsyncJobInterval()
	m.remoteCommandPatchTaskNextTimepoint = now.Add(interval)

	// 获取 dispatcher
	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](ctx.GetApp())
	if d == nil {
		ctx.LogError("TryAsyncJobs failed: NoMessageDispatcher not found")
		return false
	}

	// 创建任务
	asyncJobTypes := m.forceAsyncJobTypes
	m.forceAsyncJobTypes = make(map[int32]struct{})

	timeoutDuration := m.getAsyncJobTimeout()

	remotePatchTask, startData := cd.CreateNoMessageTaskAction(
		d, d.CreateRpcContext(), m.GetOwner().GetActorExecutor(),
		func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *async_jobs_action.TaskActionPlayerRemotePatchJobs {
			return async_jobs_action.CreateTaskActionPlayerRemotePatchJobs(
				cd.CreateNoMessageTaskActionBase(rd, actor, timeout),
				owner,
				m,
				asyncJobTypes,
				timeoutDuration,
			)
		},
	)

	// 启动任务
	err := libatapp.AtappGetModule[*cd.TaskManager](ctx.GetApp()).StartTaskAction(ctx, remotePatchTask, &startData)
	if err != nil {
		ctx.LogError("TryAsyncJobs StartTaskAction failed", "error", err, "user_id", owner.GetUserId())
		return false
	}

	m.remoteCommandPatchTask.Store(remotePatchTask)
	ctx.LogDebug("TryAsyncJobs started", "user_id", owner.GetUserId())

	return true
}

// WaitForAsyncTask 等待异步任务完成
func (m *UserAsyncJobsManager) WaitForAsyncTask(ctx cd.RpcContext) cd.RpcResult {
	if !m.IsAsyncJobsTaskRunning() {
		return cd.CreateRpcResultOk()
	}
	task := m.remoteCommandPatchTask.Load()
	result := cd.AwaitTask(m.GetOwner().GetSession().GetDispatcher().CreateAwaitableContext(), task)
	return result
}

// ForceAsyncJob 强制执行指定类型的异步任务
func (m *UserAsyncJobsManager) ForceAsyncJob(jobsType int32) {
	owner := m.GetOwner()
	if owner == nil {
		return
	}

	m.forceAsyncJobTypes[jobsType] = struct{}{}
	m.isDirty = true
}

// ResetAsyncJobsProtect 重置异步任务保护时间
func (m *UserAsyncJobsManager) ResetAsyncJobsProtect() {
	if !m.remoteCommandPatchTaskNextTimepoint.IsZero() {
		m.remoteCommandPatchTaskNextTimepoint = time.Time{}
		m.isDirty = true
	}
}

// ClearJobUuids 清理指定类型的任务 UUID 历史记录
func (m *UserAsyncJobsManager) ClearJobUuids(jobType int32) {
	historyMap, ok := m.historyUuids[jobType]
	if !ok {
		return
	}

	conflictCheckingQueueSize := m.getConflictCheckingQueueSize()
	now := time.Now()

	// 清理过期的和超出队列大小的记录
	var toRemove []string
	for uuid, item := range historyMap {
		if item == nil {
			toRemove = append(toRemove, uuid)
			continue
		}

		if now.After(item.Timeout) && len(historyMap) > int(conflictCheckingQueueSize) {
			toRemove = append(toRemove, uuid)
		}
	}

	for _, uuid := range toRemove {
		delete(historyMap, uuid)
		m.isDirty = true
	}
}

// AddJobUuid 添加任务 UUID 到历史记录
func (m *UserAsyncJobsManager) AddJobUuid(jobType int32, uuid string) {
	if jobType <= 0 || uuid == "" {
		return
	}

	if _, ok := m.historyUuids[jobType]; !ok {
		m.historyUuids[jobType] = make(async_jobs_data.HistoryMap)
	}

	conflictCheckingTimeout := m.getConflictCheckingTimeout()
	now := time.Now()

	item := &async_jobs_data.HistoryItem{
		Uuid:       uuid,
		Timeout:    now.Add(conflictCheckingTimeout),
		InsertTime: now,
	}
	m.historyUuids[jobType][uuid] = item

	// 清理冗余队列
	m.ClearJobUuids(jobType)

	m.isDirty = true
}

// IsJobUuidExists 检查任务 UUID 是否存在
func (m *UserAsyncJobsManager) IsJobUuidExists(jobType int32, uuid string) bool {
	// 检查历史记录
	if historyMap, ok := m.historyUuids[jobType]; ok {
		if _, exists := historyMap[uuid]; exists {
			return true
		}
	}

	// 检查重试队列
	if retryMap, ok := m.retryJobs[jobType]; ok {
		if _, exists := retryMap[uuid]; exists {
			return true
		}
	}

	return false
}

// AddRetryJob 添加重试任务
func (m *UserAsyncJobsManager) AddRetryJob(jobType int32, jobData *private_protocol_pbdesc.UserAsyncJobsBlobData) {
	if jobType <= 0 || jobData == nil {
		return
	}
	if jobData.GetActionUuid() == "" {
		return
	}

	if _, ok := m.retryJobs[jobType]; !ok {
		m.retryJobs[jobType] = make(async_jobs_data.RetryJobMap)
	}

	uuid := jobData.GetActionUuid()
	if existing, ok := m.retryJobs[jobType][uuid]; ok && existing != nil {
		existing.LeftRetryTimes = existing.GetLeftRetryTimes() - 1
	} else {
		m.retryJobs[jobType][uuid] = jobData.Clone()
	}

	m.isDirty = true
}

// RemoveRetryJob 移除重试任务
func (m *UserAsyncJobsManager) RemoveRetryJob(jobType int32, uuid string) {
	if retryMap, ok := m.retryJobs[jobType]; ok {
		delete(retryMap, uuid)
		m.isDirty = true
	}
}

// GetRetryJobs 获取指定类型的重试任务列表
func (m *UserAsyncJobsManager) GetRetryJobs(jobType int32) []*private_protocol_pbdesc.UserAsyncJobsBlobData {
	retryMap, ok := m.retryJobs[jobType]
	if !ok {
		return nil
	}

	result := make([]*private_protocol_pbdesc.UserAsyncJobsBlobData, 0, len(retryMap))
	for _, jobData := range retryMap {
		if jobData != nil {
			result = append(result, jobData)
		}
	}
	return result
}

// ========== 私有方法 ==========

// getAsyncJobInterval 获取异步任务执行间隔
func (m *UserAsyncJobsManager) getAsyncJobInterval() time.Duration {
	// TODO: 从配置获取
	// cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetUser().GetAsyncJob()
	// if cfg != nil && cfg.GetInterval() != nil {
	// 	return cfg.GetInterval().AsDuration()
	// }
	return 60 * time.Second
}

// getAsyncJobTimeout 获取异步任务超时时间
func (m *UserAsyncJobsManager) getAsyncJobTimeout() time.Duration {
	// TODO: 从配置获取
	// cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetUser().GetAsyncJob()
	// if cfg != nil && cfg.GetTimeout() != nil {
	// 	return cfg.GetTimeout().AsDuration()
	// }
	return 30 * time.Second
}

// getConflictCheckingTimeout 获取冲突检查超时时间
func (m *UserAsyncJobsManager) getConflictCheckingTimeout() time.Duration {
	// TODO: 从配置获取
	// cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetUser().GetAsyncJob()
	// if cfg != nil && cfg.GetConflictCheckingTimeout() != nil {
	// 	return cfg.GetConflictCheckingTimeout().AsDuration()
	// }
	return 1800 * time.Second
}

// getConflictCheckingQueueSize 获取冲突检查队列大小
func (m *UserAsyncJobsManager) getConflictCheckingQueueSize() uint32 {
	// TODO: 从配置获取
	// cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetUser().GetAsyncJob()
	// if cfg != nil && cfg.GetConflictCheckingQueueSize() > 0 {
	// 	return cfg.GetConflictCheckingQueueSize()
	// }
	return 1000
}
