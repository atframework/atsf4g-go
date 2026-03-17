package lobbysvr_logic_mail_action

import (
	"time"

	"google.golang.org/protobuf/proto"

	db "github.com/atframework/atsf4g-go/component/db"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component/router"
	uc "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
)

// UserMailManagerForAsyncJobs 定义异步任务所需的 UserMailManager 接口
// 避免循环引用
type UserMailManagerForAsyncJobs interface {
	MergeGlobalMails(ctx cd.RpcContext) (cd.RpcResult, int32)

	FetchAllUnloadedMails(ctx cd.RpcContext) []int64

	SetMailContentLoaded(ctx cd.RpcContext, mailId int64)

	GetMailRaw(mailId int64) *mail_data.MailData

	RemoveUserMail(ctx cd.RpcContext, mailId int64)

	RemoveExpiredMails(ctx cd.RpcContext)

	GetPendingRemoveList() []int64

	RemovePendingRemoveItem(mailId int64)

	GetLazySaveCounter() int

	IncrementLazySaveCounter() int

	ResetLazySaveCounter()

	TryToStartAsyncJobs(ctx cd.RpcContext)

	SendAllSyncData(ctx cd.RpcContext)
}

// 懒保存计数阈值
const (
	EN_SL_LAZY_SAVE_COUNTER_L1 = 5
)

// TaskActionMailAsyncJobs 用户邮件异步任务
type TaskActionMailAsyncJobs struct {
	cd.TaskActionNoMessageBase
	owner   *data.User
	manager UserMailManagerForAsyncJobs

	needRestart           bool
	isWritable            bool
	fetchMailNumber       int
	removeMailNumber      int
	mergeGlobalMailNumber int
	timeoutDuration       time.Duration
	timeoutTimepoint      time.Time
}

func (t *TaskActionMailAsyncJobs) Name() string {
	return "TaskActionMailAsyncJobs"
}

// GetOwner 获取任务所属的用户
func (t *TaskActionMailAsyncJobs) GetOwner() *data.User {
	return t.owner
}

// CreateTaskActionMailAsyncJobs 创建用户邮件异步任务
func CreateTaskActionMailAsyncJobs(
	dispatcher cd.DispatcherImpl,
	actor *cd.ActorExecutor,
	owner *data.User,
	manager UserMailManagerForAsyncJobs,
	timeoutDuration time.Duration,
) *TaskActionMailAsyncJobs {
	t := &TaskActionMailAsyncJobs{
		TaskActionNoMessageBase: cd.CreateNoMessageTaskActionBase(dispatcher, actor, timeoutDuration),
		owner:                   owner,
		manager:                 manager,
		needRestart:             false,
		isWritable:              false,
		fetchMailNumber:         0,
		removeMailNumber:        0,
		mergeGlobalMailNumber:   0,
		timeoutDuration:         timeoutDuration,
		timeoutTimepoint:        time.Now().Add(timeoutDuration),
	}
	return t
}

func (t *TaskActionMailAsyncJobs) Run(_startData *cd.DispatcherStartData) error {
	t.needRestart = false
	t.isWritable = false
	t.fetchMailNumber = 0
	t.removeMailNumber = 0
	t.mergeGlobalMailNumber = 0

	user := t.GetOwner()
	if user == nil || t.manager == nil {
		return nil
	}

	ctx := t.GetAwaitableContext()

	// 检查用户缓存是否仍可写
	if !user.IsWriteable() {
		return nil
	}

	t.isWritable = true

	cache := uc.GetUserRouterManager(t.GetRpcContext().GetApp()).GetCache(router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   user.GetZoneId(),
		ObjectID: user.GetUserId()})

	// 1. 补全全局邮件内容，移除过期全服邮件（这里没有RPC）
	mergeResult, mergeNumber := t.manager.MergeGlobalMails(ctx)
	if mergeResult.GetResponseCode() == 0 {
		t.mergeGlobalMailNumber += int(mergeNumber) // 显式转换 int32 -> int
	}

	// 2. 拉取和补全邮件内容
	fetchResult := t.fetchMailContents()
	if fetchResult >= 0 {
		t.fetchMailNumber += fetchResult
	} else {
		ctx.LogError("TaskActionMailAsyncJobs fetch_mail_contents failed",
			"result", fetchResult)
	}

	// 3. 删除过期邮件
	t.manager.RemoveExpiredMails(ctx)
	removeResult := t.cleanupRemovedMails()
	if removeResult >= 0 {
		t.removeMailNumber += removeResult
	} else {
		ctx.LogError("TaskActionMailAsyncJobs cleanup_removed_mails failed",
			"result", removeResult)
	}

	// 可能是从中间中断的，需要重新计算一次是否可写
	t.isWritable = user.IsWriteable()

	// 执行时间过长则中断，下一次再启动流程
	if t.isWritable && time.Until(t.timeoutTimepoint) < (t.timeoutDuration/2) {
		t.needRestart = true
	}

	// 如果缓存对象仍然可写，且有变化，则触发保存
	if t.isWritable && (t.fetchMailNumber > 0 || t.removeMailNumber > 0 || t.mergeGlobalMailNumber > 0) {
		lazySaveCounter := t.manager.IncrementLazySaveCounter()
		if lazySaveCounter >= EN_SL_LAZY_SAVE_COUNTER_L1 {
			t.manager.ResetLazySaveCounter()
			cache.SaveObject(t.GetAwaitableContext(), nil)
			t.isWritable = user.IsWriteable()
		} else {
			// TODO: 标记快速保存
			// router_manager_set::me()->mark_fast_save(router_player_manager::me().get(), cache)
		}
	}

	return nil
}

// fetchMailContents 拉取邮件内容
func (t *TaskActionMailAsyncJobs) fetchMailContents() int {
	if t.manager == nil {
		return 0
	}

	ctx := t.GetAwaitableContext()
	mailUnloaded := t.manager.FetchAllUnloadedMails(ctx)
	if len(mailUnloaded) == 0 {
		return 0
	}

	// 记录未完成的邮件
	undoMails := make(map[int64]struct{})
	for _, mailId := range mailUnloaded {
		undoMails[mailId] = struct{}{}
	}

	ret := 0

	// 遍历拉取每封邮件的内容
	for _, mailId := range mailUnloaded {
		mailContent, retResult := db.DatabaseTableMailContentLoadWithMailId(ctx, mailId)
		if retResult.IsError() {
			if retResult.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
				ctx.LogDebug("TaskActionMailAsyncJobs fetch mail content failed",
					"mail_id", mailId,
					"error", retResult.GetErrorString())
			}
			continue
		}

		delete(undoMails, mailId)

		// 获取原始邮件数据并更新内容
		mailData := t.manager.GetMailRaw(mailId)
		// 可能已经被移除，如果被移除了，忽略即可
		if mailData != nil && mailData.Record != nil {
			// 拷贝内容
			if mailData.Content == nil {
				mailData.Content = &public_protocol_pbdesc.DMailContent{}
			}
			jobData := mailContent.GetJobData()
			if jobData != nil {
				proto.Merge(mailData.Content, jobData.GetMailContent())
			}
			mailData.Record.FetchErrorCount = 0

			ctx.LogDebug("TaskActionMailAsyncJobs fetch mail content success",
				"mail_id", mailId)
			ret++
		} else {
			ctx.LogDebug("TaskActionMailAsyncJobs fetch mail content success, but mail may already removed",
				"mail_id", mailId)
		}

		t.manager.SetMailContentLoaded(ctx, mailId)
	}

	// 处理无效的脏数据
	for dirtyMailId := range undoMails {
		t.manager.SetMailContentLoaded(ctx, dirtyMailId)

		mailData := t.manager.GetMailRaw(dirtyMailId)
		// 可能已经被移除，如果被移除了，忽略即可
		// 可能是邮件内容被删除了，这时候要清理一下索引，这里设置错误容忍值是为了适应万一数据库短期故障
		if mailData != nil && mailData.Record != nil {
			mailData.Record.FetchErrorCount = mailData.Record.GetFetchErrorCount() + 1
			if mailData.Record.GetFetchErrorCount() > mail_data.EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT {
				t.manager.RemoveUserMail(ctx, dirtyMailId)
			}
		} else {
			t.manager.RemoveUserMail(ctx, dirtyMailId)
		}
	}

	return ret
}

// cleanupRemovedMails 清理待删除的邮件内容
func (t *TaskActionMailAsyncJobs) cleanupRemovedMails() int {
	if t.manager == nil {
		return 0
	}

	ret := 0
	ctx := t.GetAwaitableContext()
	user := t.GetOwner()

	pendingToRemoveList := t.manager.GetPendingRemoveList()

	for _, delMailId := range pendingToRemoveList {
		// 检查用户是否仍可写
		if user != nil && !user.IsWriteable() {
			break
		}

		if delMailId == 0 {
			t.manager.RemovePendingRemoveItem(delMailId)
			ret++
			continue
		}

		retResult := db.DatabaseTableMailContentDelWithMailId(ctx, delMailId)
		if retResult.IsError() {
			if retResult.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
				ctx.LogWarn("TaskActionMailAsyncJobs delete mail content failed",
					"mail_id", delMailId,
					"error", retResult.GetErrorString())
				break
			}
			// 邮件已经被删除过了
			ctx.LogInfo("TaskActionMailAsyncJobs delete mail content, but mail already removed before",
				"mail_id", delMailId)
		} else {
			ctx.LogDebug("TaskActionMailAsyncJobs delete mail content success",
				"mail_id", delMailId)
		}

		ret++
		t.manager.RemovePendingRemoveItem(delMailId)

		// 执行时间过长则中断，下一次再启动流程
		if time.Until(t.timeoutTimepoint) < (t.timeoutDuration / 2) {
			t.needRestart = true
			break
		}
	}

	return ret
}

func (t *TaskActionMailAsyncJobs) OnSuccess() {
	ctx := t.GetRpcContext()
	user := t.GetOwner()

	if user != nil {
		ctx.LogInfo("TaskActionMailAsyncJobs success",
			"user_id", user.GetUserId(),
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	} else {
		ctx.LogInfo("TaskActionMailAsyncJobs success (unknown user)",
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	}

	// 数据变更推送
	if t.manager != nil {
		t.manager.SendAllSyncData(ctx)
	}

	// 先推送再重启，否则邮件脏数据不会下发
	if t.isWritable && t.needRestart && t.manager != nil {
		t.manager.TryToStartAsyncJobs(ctx)
	}
}

func (t *TaskActionMailAsyncJobs) OnFailed() {
	ctx := t.GetRpcContext()
	user := t.GetOwner()

	if user != nil {
		ctx.LogError("TaskActionMailAsyncJobs failed",
			"user_id", user.GetUserId(),
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	} else {
		ctx.LogError("TaskActionMailAsyncJobs failed (unknown user)",
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	}

	// 数据变更推送
	if t.manager != nil {
		t.manager.SendAllSyncData(ctx)
	}

	// 先推送再重启，否则邮件脏数据不会下发
	if t.isWritable && t.needRestart && t.manager != nil {
		t.manager.TryToStartAsyncJobs(ctx)
	}
}

func (t *TaskActionMailAsyncJobs) OnTimeout() {
	ctx := t.GetRpcContext()
	user := t.GetOwner()

	if user != nil {
		ctx.LogError("TaskActionMailAsyncJobs timeout",
			"user_id", user.GetUserId(),
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	} else {
		ctx.LogError("TaskActionMailAsyncJobs timeout (unknown user)",
			"fetch_mail_number", t.fetchMailNumber,
			"remove_mail_number", t.removeMailNumber,
			"merge_global_mail_number", t.mergeGlobalMailNumber)
	}

	// 数据变更推送
	if t.manager != nil {
		t.manager.SendAllSyncData(ctx)
	}
}
