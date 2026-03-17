// Copyright 2026 atframework
// Translated from async_jobs related C++ code

package lobbysvr_logic_async_jobs_data

import (
	"time"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
)

// HistoryItem 历史任务记录项
type HistoryItem struct {
	Uuid       string
	Timeout    time.Time
	InsertTime time.Time // 用于 LRU 排序
}

// HistoryMap 历史任务记录表 (uuid -> HistoryItem)
type HistoryMap map[string]*HistoryItem

// HistoryByJobType 按任务类型的历史记录 (job_type -> HistoryMap)
type HistoryByJobType map[int32]HistoryMap

// RetryJobMap 重试任务表 (uuid -> job_data)
type RetryJobMap map[string]*private_protocol_pbdesc.UserAsyncJobsBlobData

// RetryJobsByType 按任务类型的重试任务表 (job_type -> RetryJobMap)
type RetryJobsByType map[int32]RetryJobMap

// ActionOptions 异步任务操作选项
type ActionOptions struct {
	// NotifyPlayer 是否通知玩家
	NotifyPlayer bool

	// IgnoreRouterCache 忽略玩家路由表缓存
	// 注意: 如果设置为 true，需要评估 QPS，不能太高否则可能影响 login 表负载和稳定性
	IgnoreRouterCache bool
}

// NewActionOptions 创建默认选项
func NewActionOptions() ActionOptions {
	return ActionOptions{
		NotifyPlayer:      true,
		IgnoreRouterCache: false,
	}
}

// NewActionOptionsWithNotify 创建带通知选项
func NewActionOptionsWithNotify(notifyPlayer bool) ActionOptions {
	return ActionOptions{
		NotifyPlayer:      notifyPlayer,
		IgnoreRouterCache: false,
	}
}
