// Copyright 2026 atframework
// @brief 全服邮件操作：发送和删除全服邮件

package atframework_component_mail

import (
	"time"

	config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	uuid "github.com/atframework/atsf4g-go/component/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	// EN_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT 全服邮件泄漏检查超时时间
	EN_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT = 7 * 24 * 3600 // 7天（秒）
	// EN_MAIL_DEFAULT_MAX_RETRY 默认最大重试次数
	EN_MAIL_DEFAULT_MAX_RETRY = 5
)

// compactMails 淘汰老邮件，保持邮件数量在限制内
func compactMails(_ctx cd.AwaitableContext, dbData *private_protocol_pbdesc.DatabaseGlobalMailBlobData) {
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup == nil {
		return
	}
	constIndex := configGroup.GetCustomIndex().GetConstIndex()
	if constIndex == nil {
		return
	}
	maxMailCount := constIndex.GetGlobalMailMaxCountPerMajorType()
	if maxMailCount <= 0 {
		return
	}

	now := time.Now().Unix()
	futureReserveCount := constIndex.GetGlobalMailFutureReserveMaxCountPerMajorType()
	futureMailCount := int32(0)
	for _, record := range dbData.GetMailRecords() {
		if record.GetStartTime() > now {
			futureMailCount++
		}
	}
	if futureReserveCount > futureMailCount {
		futureReserveCount = futureMailCount
	}
	if futureReserveCount < 0 {
		futureReserveCount = 0
	}
	maxMailCount += futureReserveCount

	// 按类型只保留一定数量的邮件
	if int32(len(dbData.GetMailRecords())) <= maxMailCount {
		return
	}

	// 用于检查未来生效的邮件，防止起始生效时间异常过高而无法删除
	deliveryTimeMaxOffset := int64(30 * 24 * 3600) // 默认30天
	sectionConfig := configGroup.GetSectionConfig()
	if sectionConfig != nil && sectionConfig.GetMail() != nil &&
		sectionConfig.GetMail().GetCompactDeliveryTimeMaxOffset() != nil {
		deliveryTimeMaxOffset = int64(sectionConfig.GetMail().GetCompactDeliveryTimeMaxOffset().GetSeconds())
	}
	if deliveryTimeMaxOffset <= 0 {
		deliveryTimeMaxOffset = 30 * 24 * 3600
	}

	rmcnt := int32(len(dbData.GetMailRecords())) - maxMailCount
	for rmcnt > 0 {
		rmcnt--
		if int32(len(dbData.GetMailRecords())) <= maxMailCount {
			return
		}

		selectedIdx := -1
		selectMailCompactTime := int64(-1)

		// 首先移除可历史删除的邮件
		for i := 0; i < len(dbData.GetMailRecords()); i++ {
			record := dbData.GetMailRecords()[i]
			if MailIsHistoryRemovable(record) {
				dirtyRecord := &public_protocol_pbdesc.DMailRecord{}
				proto.Merge(dirtyRecord, record)
				dirtyRecord.Status = dirtyRecord.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)
				// 重设失效时间为当前时间+容忍误差时间,之后可以删除该记录
				removeTime := now + EN_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT
				if dirtyRecord.GetExpiredTime() > removeTime {
					dirtyRecord.ExpiredTime = removeTime
				}
				if dirtyRecord.GetRemoveTime() > removeTime {
					dirtyRecord.RemoveTime = removeTime
				}
				dbData.PendingRemoveList = append(dbData.GetPendingRemoveList(), dirtyRecord)
				// 移除当前记录
				dbData.MailRecords = append(dbData.GetMailRecords()[:i], dbData.GetMailRecords()[i+1:]...)
				i--
				continue
			}

			compactTime := record.GetStartTime()
			if record.GetDeliveryTime() > compactTime {
				compactTime = record.GetDeliveryTime()
			} else if compactTime > record.GetDeliveryTime()+deliveryTimeMaxOffset {
				compactTime = record.GetDeliveryTime() + deliveryTimeMaxOffset
			}

			if selectedIdx == -1 || compactTime < selectMailCompactTime {
				selectedIdx = i
				selectMailCompactTime = compactTime
			}
		}

		// 移除完过期邮件，可能已经不满了
		if int32(len(dbData.GetMailRecords())) <= maxMailCount || selectedIdx < 0 {
			return
		}

		dirtyRecord := &public_protocol_pbdesc.DMailRecord{}
		proto.Merge(dirtyRecord, dbData.GetMailRecords()[selectedIdx])
		dirtyRecord.Status = dirtyRecord.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)
		// 重设失效时间为当前时间+容忍误差时间,之后可以删除该记录
		removeTime := now + EN_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT
		if dirtyRecord.GetExpiredTime() > removeTime {
			dirtyRecord.ExpiredTime = removeTime
		}
		if dirtyRecord.GetRemoveTime() > removeTime {
			dirtyRecord.RemoveTime = removeTime
		}
		dbData.PendingRemoveList = append(dbData.GetPendingRemoveList(), dirtyRecord)
		// 移除选中的记录
		dbData.MailRecords = append(dbData.GetMailRecords()[:selectedIdx], dbData.GetMailRecords()[selectedIdx+1:]...)
	}
}

// AddGlobalMail 添加全服邮件
// @param ctx 上下文
// @param mail 邮件内容
// @param zoneId 区服ID，0则是跨区服全服邮件
// @param channel 来源渠道
// @param channelParam 渠道参数
func AddGlobalMail(
	ctx cd.AwaitableContext,
	mail *public_protocol_pbdesc.DMailContent,
	zoneId uint32,
	channel int32,
	channelParam int64,
	reason *public_protocol_pbdesc.DMailFlowReason,
) cd.RpcResult {
	if !config.IsValidGlobalMail(mail.GetMajorType()) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if (mail.GetTitle() == "" || mail.GetContent() == "") && mail.GetMailTemplateId() == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if mail.GetMailId() == 0 {
		guid, result := uuid.GenerateGlobalUniqueID(ctx,
			private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_MAIL,
			private_protocol_pbdesc.EnGlobalUUIDMinorType_EN_GLOBAL_UUID_MIT_DEFAULT,
			private_protocol_pbdesc.EnGlobalUUIDPatchType_EN_GLOBAL_UUID_PT_DEFAULT)
		if result.IsError() {
			return result
		}
		mail.MailId = int64(guid)
	}

	mail.Channel = channel
	mail.ChannelParam = channelParam

	now := time.Now().Unix()

	if mail.Reason == nil {
		mail.Reason = &public_protocol_pbdesc.DMailFlowReason{}
	}
	mail.Reason.MajorReason = reason.MajorReason
	mail.Reason.MinorReason = reason.MinorReason
	mail.Reason.Parameter = reason.Parameter

	// setup delivery_time, show_time, start_time, expired_time
	if mail.GetDeliveryTime() <= 0 {
		mail.DeliveryTime = now
	}
	if mail.GetStartTime() <= 0 {
		mail.StartTime = mail.GetDeliveryTime()
	}
	if mail.GetShowTime() <= 0 {
		mail.ShowTime = mail.GetStartTime()
	}

	// 默认过期时间
	if mail.GetExpiredTime() <= 0 {
		defaultExpire := int64(30 * 24 * 3600) // 默认30天
		configGroup := config.GetConfigManager().GetCurrentConfigGroup()
		if configGroup != nil {
			constIndex := configGroup.GetCustomIndex().GetConstIndex()
			if constIndex != nil && constIndex.GetMailDefaultExpire() != nil {
				defaultExpire = int64(constIndex.GetMailDefaultExpire().GetSeconds())
			}
		}
		mail.ExpiredTime = mail.GetStartTime() + defaultExpire
	}
	if mail.GetRemoveTime() <= 0 {
		if mail.GetResolveExpiredTime() > 0 {
			mail.RemoveTime = MailPaddingDayTime(mail.GetExpiredTime() + mail.GetResolveExpiredTime())
		} else {
			mail.RemoveTime = mail.GetExpiredTime()
		}
	}

	// DB: add mail content
	mailTable := &private_protocol_pbdesc.DatabaseTableMailContent{
		MailId: mail.GetMailId(),
	}
	if mailTable.JobData == nil {
		mailTable.JobData = &private_protocol_pbdesc.DatabaseMailContentBlobData{}
	}
	mailTable.JobData.MailContent = mail.Clone()

	ret := db.DatabaseTableMailContentUpdateMailId(ctx, mailTable)
	if ret.IsError() {
		ctx.LogError("add_mail_content for global failed",
			"mail_id", mail.GetMailId(),
			"error", ret,
		)
		return ret
	}
	ctx.LogInfo("add_mail_content for global success",
		"mail_id", mail.GetMailId(),
	)

	// DB: retry and add mail record
	var lastErr cd.RpcResult
	for leftRetry := EN_MAIL_DEFAULT_MAX_RETRY; leftRetry > 0; leftRetry-- {
		dbTable, loadResult := db.DatabaseTableGlobalMailLoadWithZoneIdMajorType(ctx, zoneId, mail.GetMajorType())
		if loadResult.IsError() &&
			loadResult.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			ctx.LogError("get_global_mail failed",
				"zone_id", zoneId,
				"major_type", mail.GetMajorType(),
				"error", loadResult,
			)
			lastErr = loadResult
			continue
		}

		// 可能是新增数据，强制赋值一下
		if dbTable == nil {
			dbTable = &private_protocol_pbdesc.DatabaseTableGlobalMail{}
		}
		dbTable.ZoneId = zoneId
		dbTable.MajorType = mail.GetMajorType()
		if dbTable.JobData == nil {
			dbTable.JobData = &private_protocol_pbdesc.DatabaseGlobalMailBlobData{}
		}

		mailRecord := &public_protocol_pbdesc.DMailRecord{
			MailId:             mail.GetMailId(),
			MajorType:          mail.GetMajorType(),
			MinorType:          mail.GetMinorType(),
			Status:             int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_NONE),
			DeliveryTime:       mail.GetDeliveryTime(),
			StartTime:          mail.GetStartTime(),
			ShowTime:           mail.GetShowTime(),
			ExpiredTime:        mail.GetExpiredTime(),
			RemoveTime:         mail.GetRemoveTime(),
			ResolveExpiredTime: mail.GetResolveExpiredTime(),
			IsGlobalMail:       true,
		}
		dbTable.JobData.MailRecords = append(dbTable.GetJobData().GetMailRecords(), mailRecord)

		// 如果邮件太多需要淘汰老的
		compactMails(ctx, dbTable.GetJobData())

		updateResult := db.DatabaseTableGlobalMailUpdateZoneIdMajorType(ctx, dbTable)
		if updateResult.IsError() {
			ctx.LogError("replace_global_mail failed",
				"zone_id", zoneId,
				"major_type", mail.GetMajorType(),
				"error", updateResult,
			)
			lastErr = updateResult
			continue
		}

		// 成功
		ctx.LogInfo("add_global_mail success",
			"zone_id", zoneId,
			"mail_id", mail.GetMailId(),
		)
		return cd.CreateRpcResultOk()
	}

	return lastErr
}

// RemoveGlobalMail 移除全服邮件
// @param ctx 上下文
// @param zoneId 区服ID，0则是跨区服全服邮件
// @param mailId 邮件ID
func RemoveGlobalMail(
	ctx cd.AwaitableContext,
	zoneId uint32,
	mailId int64,
) cd.RpcResult {
	if mailId == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	ctx.LogDebug("remove_global_mail start",
		"zone_id", zoneId,
		"mail_id", mailId,
	)

	var majorType int32
	var lastErr cd.RpcResult

	for leftRetry := EN_MAIL_DEFAULT_MAX_RETRY; leftRetry > 0; leftRetry-- {
		if majorType == 0 {
			// DB: get mail content
			mailContent, loadResult := db.DatabaseTableMailContentLoadWithMailId(ctx, mailId)
			if loadResult.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
				return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
			}
			if loadResult.IsError() {
				ctx.LogError("batch_get_mail_content failed",
					"mail_id", mailId,
					"error", loadResult,
				)
				lastErr = loadResult
				continue
			}

			if mailContent.GetJobData().GetMailContent().GetMailId() != mailId ||
				mailContent.GetJobData().GetMailContent().GetMajorType() == 0 {
				return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
			}

			majorType = mailContent.GetJobData().GetMailContent().GetMajorType()
		}

		dbTable, loadResult := db.DatabaseTableGlobalMailLoadWithZoneIdMajorType(ctx, zoneId, majorType)
		if loadResult.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
		}
		if loadResult.IsError() {
			ctx.LogError("get_global_mail failed",
				"zone_id", zoneId,
				"major_type", majorType,
				"error", loadResult,
			)
			lastErr = loadResult
			continue
		}

		dbData := dbTable.GetJobData()
		selectedIdx := -1
		for i := 0; i < len(dbData.GetMailRecords()); i++ {
			if dbData.GetMailRecords()[i].GetMailId() == mailId {
				selectedIdx = i
				break
			}
		}

		if selectedIdx < 0 {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
		}

		now := time.Now().Unix()
		dirtyRecord := &public_protocol_pbdesc.DMailRecord{}
		proto.Merge(dirtyRecord, dbData.GetMailRecords()[selectedIdx])
		dirtyRecord.Status = dirtyRecord.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)
		// 重设失效时间为当前时间+容忍误差时间,之后可以删除该记录
		removeTime := now + EN_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT
		if dirtyRecord.GetExpiredTime() > removeTime {
			dirtyRecord.ExpiredTime = removeTime
		}
		if dirtyRecord.GetRemoveTime() > removeTime {
			dirtyRecord.RemoveTime = removeTime
		}
		dbData.PendingRemoveList = append(dbData.GetPendingRemoveList(), dirtyRecord)
		// 移除选中的记录
		dbData.MailRecords = append(dbData.GetMailRecords()[:selectedIdx], dbData.GetMailRecords()[selectedIdx+1:]...)

		// 如果邮件太多需要淘汰老的
		compactMails(ctx, dbData)

		updateResult := db.DatabaseTableGlobalMailUpdateZoneIdMajorType(ctx, dbTable)
		if updateResult.IsError() {
			ctx.LogError("replace_global_mail failed",
				"zone_id", zoneId,
				"major_type", majorType,
				"error", updateResult,
			)
			lastErr = updateResult
			continue
		}

		// 成功
		ctx.LogInfo("remove_global_mail success",
			"zone_id", zoneId,
			"mail_id", mailId,
		)
		return cd.CreateRpcResultOk()
	}

	return lastErr
}

// AddGlobalMailWithTemplate 使用模板发送全服邮件
// @param ctx 上下文
func AddGlobalMailWithTemplate(
	ctx cd.AwaitableContext,
	MailTemplateId int32,
	Sender *public_protocol_pbdesc.DMailUserInfo,
	ZoneId uint32,
	Channel int32,
	ChannelParam int64,
	itemOffset []*public_protocol_common.DItemOffset,
	Extensions map[string]string,
	reason *public_protocol_pbdesc.DMailFlowReason,
) cd.RpcResult {
	if MailTemplateId == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	mail := &public_protocol_pbdesc.DMailContent{}

	// render mail title and content
	ret := MailRenderTemplate(MailTemplateId, mail, Sender, itemOffset, Extensions)
	if ret.IsError() {
		ctx.LogError("mail_render_template failed for global mail",
			"mail_template_id", MailTemplateId,
			"zone_id", ZoneId,
			"error", ret,
		)
		return ret
	}

	if mail.GetMailId() == 0 {
		guid, result := uuid.GenerateGlobalUniqueID(ctx,
			private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_MAIL,
			private_protocol_pbdesc.EnGlobalUUIDMinorType_EN_GLOBAL_UUID_MIT_DEFAULT,
			private_protocol_pbdesc.EnGlobalUUIDPatchType_EN_GLOBAL_UUID_PT_DEFAULT)
		if result.IsError() {
			return result
		}
		mail.MailId = int64(guid)
	}

	// call AddGlobalMail()
	return AddGlobalMail(ctx, mail, ZoneId, Channel, ChannelParam, reason)
}
