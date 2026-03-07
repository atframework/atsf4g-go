package atframework_component_async_jobs

import (
	pu "github.com/atframework/atframe-utils-go/proto_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	uuid "github.com/atframework/atsf4g-go/component-uuid"
	"google.golang.org/protobuf/proto"
)

// ActionOptions 异步任务操作选项
type ActionOptions struct {
	// NotifyPlayer 通知玩家
	NotifyPlayer bool

	// IgnoreRouterCache 忽略玩家路由表缓存
	// 注意如果这一项要设置为true，要评估QPS。不能太高否则可能影响login表负载和稳定性
	IgnoreRouterCache bool
}

// DefaultActionOptions 默认操作选项
func DefaultActionOptions() ActionOptions {
	return ActionOptions{
		NotifyPlayer:      true,
		IgnoreRouterCache: false,
	}
}

// GetJobs 获取用户异步任务表所有数据
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @return []pu.RedisListIndexMessage, cd.RpcResult
func GetJobs(
	ctx cd.AwaitableContext,
	jobsType int32,
	userId uint64,
	zoneId uint32,
) ([]pu.RedisListIndexMessage, cd.RpcResult) {
	if jobsType == 0 || userId == 0 {
		ctx.LogError("GetJobs invalid parameters",
			"jobs_type", jobsType,
			"user_id", userId)
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	ctx.LogDebug("GetJobs unsupported type",
		"jobs_type", jobsType,
		"user_id", userId)

	indexMessages, result := db.DatabaseTableUserAsyncJobsLoadAllWithJobTypeUserIdZoneId(
		ctx, jobsType, userId, zoneId,
	)
	return indexMessages, result
}

// DelJobs 删除用户异步任务表指定任务数据
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @param indexes 要删除的下标列表
// @return cd.RpcResult
func DelJobs(
	ctx cd.AwaitableContext,
	jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType,
	userId uint64,
	zoneId uint32,
	indexes []uint64,
) cd.RpcResult {
	if jobsType == 0 || userId == 0 {
		ctx.LogError("DelJobs invalid parameters",
			"jobs_type", jobsType,
			"user_id", userId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	ctx.LogDebug("DelJobs unsupported type",
		"jobs_type", jobsType,
		"user_id", userId)

	if len(indexes) == 0 {
		return cd.CreateRpcResultOk()
	}

	return db.DatabaseTableUserAsyncJobsDelIndexWithJobTypeUserIdZoneId(
		ctx, indexes, int32(jobsType), userId, zoneId,
	)
}

// AddJobs 添加用户异步任务
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @param jobData 任务数据
// @param options 操作选项
// @return cd.RpcResult, uint64 (新记录的索引)
func AddJobs(
	ctx cd.AwaitableContext,
	jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType,
	userId uint64,
	zoneId uint32,
	jobData *private_protocol_pbdesc.UserAsyncJobsBlobData,
	options ActionOptions,
) (cd.RpcResult, uint64) {
	if jobsType == 0 || userId == 0 {
		ctx.LogError("AddJobs invalid parameters",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), 0
	}

	ctx.LogDebug("AddJobs unsupported type",
		"jobs_type", jobsType,
		"user_id", userId,
		"zone_id", zoneId)

	if jobData == nil || jobData.GetAction() == nil {
		ctx.LogError("AddJobs without a action",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), 0
	}

	if jobData.GetActionUuid() == "" {
		jobData.ActionUuid = uuid.GenerateShortUUID(ctx)
	}

	table := &private_protocol_pbdesc.DatabaseTableUserAsyncJobs{
		JobType: int32(jobsType),
		UserId:  userId,
		ZoneId:  zoneId,
		JobData: proto.Clone(jobData).(*private_protocol_pbdesc.UserAsyncJobsBlobData),
	}

	result, newListIndex := db.DatabaseTableUserAsyncJobsAddJobTypeUserIdZoneId(ctx, table)
	if result.IsError() {
		ctx.LogError("AddJobs db add failed",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId,
			"error", result)
		return result, 0
	}

	if options.NotifyPlayer {
		notifyOnlinePlayer(ctx, jobsType, userId, zoneId, options.IgnoreRouterCache)
	}

	ctx.LogDebug("AddJobs success",
		"jobs_type", jobsType,
		"user_id", userId,
		"zone_id", zoneId,
		"list_index", newListIndex)

	return cd.CreateRpcResultOk(), newListIndex
}

// AddJobsWithRetry 添加用户异步任务（自动补全重试次数）
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @param jobData 任务数据
// @param options 操作选项
// @return cd.RpcResult, uint64 (新记录的索引)
func AddJobsWithRetry(
	ctx cd.AwaitableContext,
	jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType,
	userId uint64,
	zoneId uint32,
	jobData *private_protocol_pbdesc.UserAsyncJobsBlobData,
	options ActionOptions,
) (cd.RpcResult, uint64) {
	if jobData.GetLeftRetryTimes() <= 0 {
		defaultRetryTimes := int32(3)
		cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetUser().GetAsyncJob()
		if cfg != nil && cfg.GetDefaultRetryTimes() > 0 {
			defaultRetryTimes = cfg.GetDefaultRetryTimes()
		}
		jobData.LeftRetryTimes = defaultRetryTimes
	}

	return AddJobs(ctx, jobsType, userId, zoneId, jobData, options)
}

// RemoveAllJobs 删除用户异步任务表所有数据
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @return cd.RpcResult
func RemoveAllJobs(
	ctx cd.AwaitableContext,
	jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType,
	userId uint64,
	zoneId uint32,
) cd.RpcResult {
	if jobsType == 0 || userId == 0 {
		ctx.LogError("RemoveAllJobs invalid parameters",
			"jobs_type", jobsType,
			"user_id", userId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return db.DatabaseTableUserAsyncJobsDelWithJobTypeUserIdZoneId(ctx, int32(jobsType), userId, zoneId)
}

// UpdateJobs 更新用户异步任务表单条记录
// @param ctx 上下文
// @param jobsType 任务类型
// @param userId 用户ID
// @param zoneId 区服ID
// @param jobData 待更新的数据
// @param listIndex 待更新的数据所在的数据库下标
// @param options 操作选项
// @return cd.RpcResult
func UpdateJobs(
	ctx cd.AwaitableContext,
	jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType,
	userId uint64,
	zoneId uint32,
	jobData *private_protocol_pbdesc.UserAsyncJobsBlobData,
	listIndex uint64,
	options ActionOptions,
) cd.RpcResult {
	if jobsType == private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_INVALID || userId == 0 {
		ctx.LogError("UpdateJobs invalid parameters",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if jobData == nil || jobData.GetAction() == nil {
		ctx.LogError("UpdateJobs without a action",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId)
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if jobData.GetActionUuid() == "" {
		jobData.ActionUuid = uuid.GenerateShortUUID(ctx)
	}

	table := &private_protocol_pbdesc.DatabaseTableUserAsyncJobs{
		JobType: int32(jobsType),
		UserId:  userId,
		ZoneId:  zoneId,
		JobData: proto.Clone(jobData).(*private_protocol_pbdesc.UserAsyncJobsBlobData),
	}

	result := db.DatabaseTableUserAsyncJobsUpdateJobTypeUserIdZoneId(ctx, table, listIndex)
	if result.IsError() {
		ctx.LogError("UpdateJobs db update failed",
			"jobs_type", jobsType,
			"user_id", userId,
			"zone_id", zoneId,
			"list_index", listIndex,
			"error", result)
		return result
	}

	if options.NotifyPlayer {
		notifyOnlinePlayer(ctx, jobsType, userId, zoneId, options.IgnoreRouterCache)
	}

	ctx.LogDebug("UpdateJobs success",
		"jobs_type", jobsType,
		"user_id", userId,
		"zone_id", zoneId,
		"list_index", listIndex)

	return cd.CreateRpcResultOk()
}

// notifyOnlinePlayer 尝试通知在线玩家
func notifyOnlinePlayer(ctx cd.AwaitableContext, jobsType private_protocol_pbdesc.EnPlayerAsyncJobsType, userId uint64, zoneId uint32, ignoreCache bool) {
	// 失败则放弃，只是会延迟到账，不影响逻辑
	// TODO: 实现通知在线玩家的逻辑
	// 1. 获取玩家路由信息 (fetch_user_login_cache)
	// 2. 检查玩家是否在线
	// 3. 调用 rpc::game::player_async_jobs_sync 通知玩家
	ctx.LogDebug("notifyOnlinePlayer: not fully implemented",
		"jobs_type", jobsType,
		"user_id", userId,
		"zone_id", zoneId,
		"ignore_cache", ignoreCache)
}
