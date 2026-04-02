package lobbysvr_logic_global_mail_action

import (
	atframework_component_config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	global_mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/data"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
)

// GlobalMailManagerForSync 定义 GlobalMailManager 所需的接口
type GlobalMailManagerForSync interface {
	UpdateFromDB(ctx cd.RpcContext, zoneId uint32, majorType int32, blobData *private_protocol_pbdesc.DatabaseGlobalMailBlobData, rewriteDbData bool) bool
	FetchAllUnloadedMails(ctx cd.RpcContext) []int64
	SetMailContentLoaded(mailId int64)
	GetMailRaw(mailId int64) *mail_data.MailData
	RemoveGlobalMail(ctx cd.RpcContext, mailId int64)
	SetLastSuccessFetchTimepoint(t int64)
	GetPendingToRemoveContentsList() []int64
	RemovePendingToRemoveContent(mailId int64)
}

// TaskActionGlobalMailSyncObjects 全局邮件同步任务
type TaskActionGlobalMailSyncObjects struct {
	cd.TaskActionNoMessageBase
	manager GlobalMailManagerForSync

	fetchMailNumber  int
	removeMailNumber int
	fetchTimepoint   int64
}

// CreateTaskActionGlobalMailSyncObjects 创建全局邮件同步任务
func CreateTaskActionGlobalMailSyncObjects(base cd.TaskActionNoMessageBase, mgr GlobalMailManagerForSync) *TaskActionGlobalMailSyncObjects {
	return &TaskActionGlobalMailSyncObjects{
		TaskActionNoMessageBase: base,
		manager:                 mgr,
	}
}

func (t *TaskActionGlobalMailSyncObjects) Name() string {
	return "TaskActionGlobalMailSyncObjects"
}

func (t *TaskActionGlobalMailSyncObjects) Run(_startData *cd.DispatcherStartData) error {
	t.fetchMailNumber = 0
	t.removeMailNumber = 0

	if t.manager == nil {
		return nil
	}

	ctx := t.GetAwaitableContext()
	now := ctx.GetNow().Unix()
	t.fetchTimepoint = now

	// TODO: 服务发现功能尚未实现，暂时假设当前节点为主节点
	isZoneMaster := true   // TODO: 从服务发现获取
	isGlobalMaster := true // TODO: 从服务发现获取

	localZoneId := atframework_component_config.GetConfigManager().GetLogicId()

	// 拉取邮件记录
	// zone_id = 0 表示全服邮件，local_zone_id 表示本区邮件
	// zoneIds := []uint32{0, localZoneId}
	// majorTypes := atframework_component_config.GetAllGlobalMailMajorTypesCurrent()
	zoneIds := []uint32{1001}
	majorTypes := []int32{2}
	for _, zoneId := range zoneIds {
		for _, majorType := range majorTypes {
			dbData, retResult := db.DatabaseTableGlobalMailLoadWithZoneIdMajorType(ctx, zoneId, majorType)
			if retResult.IsError() {
				if retResult.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
					ctx.LogError("TaskActionGlobalMailSyncObjects load global mail failed",
						"zone_id", zoneId,
						"major_type", majorType,
						"error", retResult.GetErrorString())
					continue
				}
				dbData = &private_protocol_pbdesc.DatabaseTableGlobalMail{
					ZoneId:    zoneId,
					MajorType: majorType,
				}
			}

			// 判断是否需要写回数据库
			rewriteDbData := (zoneId == 0 && isGlobalMaster) || (zoneId == localZoneId && isZoneMaster)
			needUpdate := t.manager.UpdateFromDB(ctx, zoneId, majorType, dbData.MutableJobData(), rewriteDbData)

			// 主节点更新删除无效数据
			if needUpdate && rewriteDbData {
				retResult = db.DatabaseTableGlobalMailUpdateZoneIdMajorType(ctx, dbData)
				if retResult.IsError() {
					ctx.LogInfo("TaskActionGlobalMailSyncObjects replace global mail failed, will retry on next round",
						"zone_id", zoneId,
						"major_type", majorType,
						"error", retResult.GetErrorString())
				} else {
					ctx.LogInfo("TaskActionGlobalMailSyncObjects replace global mail success",
						"zone_id", zoneId,
						"major_type", majorType)
				}
			}
		}
	}

	fetchResult := t.fetchMailContents()
	if fetchResult >= 0 {
		t.fetchMailNumber += fetchResult
	}

	removeResult := t.cleanupRemovedMails()
	if removeResult >= 0 {
		t.removeMailNumber += removeResult
	}

	return nil
}

// fetchMailContents 拉取邮件内容
func (t *TaskActionGlobalMailSyncObjects) fetchMailContents() int {
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
				ctx.LogDebug("TaskActionGlobalMailSyncObjects fetch mail content failed",
					"mail_id", mailId,
					"error", retResult.GetErrorString())
			}
			continue
		}

		delete(undoMails, mailId)
		t.manager.SetMailContentLoaded(mailId)

		// 获取原始邮件数据并更新内容
		mailData := t.manager.GetMailRaw(mailId)
		if mailData != nil {

			record := mailData.Record
			if record != nil {
				// 拷贝内容
				if mailData.Content == nil {
					mailData.Content = &public_protocol_pbdesc.DMailContent{}
				}
				jobData := mailContent.GetJobData()
				if jobData != nil {
					mailData.Content = jobData.GetMailContent().Clone()
				}
				record.FetchErrorCount = 0

				ctx.LogDebug("TaskActionGlobalMailSyncObjects fetch mail content success",
					"mail_id", mailId)
				ret++
			}

		} else {
			ctx.LogDebug("TaskActionGlobalMailSyncObjects fetch mail content success, but mail may already removed",
				"mail_id", mailId)
		}
	}

	if ret <= 0 {
		return ret
	}

	// 处理无效的脏数据
	for dirtyMailId := range undoMails {
		mailData := t.manager.GetMailRaw(dirtyMailId)
		if mailData != nil {
			// 增加错误计数
			record := mailData.Record
			if record != nil {
				record.FetchErrorCount = record.GetFetchErrorCount() + 1
				if record.GetFetchErrorCount() > global_mail_data.EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT {
					t.manager.RemoveGlobalMail(ctx, dirtyMailId)
				}
			} else {
				t.manager.RemoveGlobalMail(ctx, dirtyMailId)
			}
		} else {
			t.manager.RemoveGlobalMail(ctx, dirtyMailId)
		}
	}

	return ret
}

// cleanupRemovedMails 清理待删除的邮件内容
func (t *TaskActionGlobalMailSyncObjects) cleanupRemovedMails() int {
	if t.manager == nil {
		return 0
	}

	ret := 0
	ctx := t.GetAwaitableContext()

	pendingToRemoveList := t.manager.GetPendingToRemoveContentsList()

	for _, delMailId := range pendingToRemoveList {
		if delMailId == 0 {
			t.manager.RemovePendingToRemoveContent(delMailId)
			ret++
			continue
		}

		retResult := db.DatabaseTableMailContentDelWithMailId(ctx, delMailId)
		if retResult.IsError() {
			if retResult.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
				ctx.LogWarn("TaskActionGlobalMailSyncObjects delete mail content failed",
					"mail_id", delMailId,
					"error", retResult.GetErrorString())
				break
			}
			ctx.LogInfo("TaskActionGlobalMailSyncObjects delete mail content, but mail already removed before",
				"mail_id", delMailId)
		} else {
			ctx.LogDebug("TaskActionGlobalMailSyncObjects delete mail content success",
				"mail_id", delMailId)
		}

		t.manager.RemovePendingToRemoveContent(delMailId)
		ret++
	}

	return ret
}

func (t *TaskActionGlobalMailSyncObjects) OnSuccess() {
	ctx := t.GetRpcContext()
	ctx.LogInfo("global mail async jobs success",
		"fetch_mail_number", t.fetchMailNumber,
		"remove_mail_number", t.removeMailNumber)

	if t.manager != nil && t.fetchTimepoint > 0 {
		t.manager.SetLastSuccessFetchTimepoint(t.fetchTimepoint)
	}
}

func (t *TaskActionGlobalMailSyncObjects) OnFailed() {
	ctx := t.GetRpcContext()
	ctx.LogError("global mail async jobs failed",
		"fetch_mail_number", t.fetchMailNumber,
		"remove_mail_number", t.removeMailNumber)
}

func (t *TaskActionGlobalMailSyncObjects) OnTimeout() {
	ctx := t.GetRpcContext()
	ctx.LogError("global mail async jobs timeout",
		"fetch_mail_number", t.fetchMailNumber,
		"remove_mail_number", t.removeMailNumber)
}
