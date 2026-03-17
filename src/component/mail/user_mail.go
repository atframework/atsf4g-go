package atframework_component_mail

import (
	"time"

	async_jobs "github.com/atframework/atsf4g-go/component/async_jobs"
	config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	uuid "github.com/atframework/atsf4g-go/component/uuid"
)

// AddUserMailResult 用户邮件添加结果
type AddUserMailResult struct {
	MailId     int64
	MailRecord *public_protocol_pbdesc.DMailRecord
}

// AddUserMail 添加用户邮件
// @param ctx 上下文
// @param userId 用户ID
// @param zoneId 区服ID
// @param mail 邮件内容
// @param channel 来源渠道
// @param channelParam 渠道参数
// @param reason 道具变更原因
// @return cd.RpcResult, *AddUserMailResult
func AddUserMail(
	ctx cd.AwaitableContext,
	userId uint64,
	zoneId uint32,
	mail *public_protocol_pbdesc.DMailContent,
	channel int32,
	channelParam int64,
	reason *public_protocol_pbdesc.DMailFlowReason,
) (cd.RpcResult, *AddUserMailResult) {
	// check user_id
	if userId == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), nil
	}

	// check mail type
	if mail.GetMajorType() <= 0 || !config.IsValidUserMail(mail.GetMajorType()) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), nil
	}

	// 检查附件数量限制
	maxItemDetailSz := int32(15)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	// 注意：GetMailMaxAttachmentDetailCount 方法不存在，使用默认值

	if int32(len(mail.GetAttachmentsOffset())) > maxItemDetailSz {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), nil
	}

	// alloc mail_id first
	if mail.GetMailId() == 0 {
		guid, result := uuid.GenerateGlobalUniqueID(ctx,
			private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_MAIL,
			private_protocol_pbdesc.EnGlobalUUIDMinorType_EN_GLOBAL_UUID_MIT_DEFAULT,
			private_protocol_pbdesc.EnGlobalUUIDPatchType_EN_GLOBAL_UUID_PT_DEFAULT)
		if result.IsError() {
			return result, nil
		}
		mail.MailId = int64(guid)
	}

	mail.Channel = channel
	mail.ChannelParam = channelParam

	now := time.Now().Unix()

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
		if configGroup != nil {
			constIndex := configGroup.GetCustomIndex().GetConstIndex()
			if constIndex != nil && constIndex.GetMailDefaultExpire() != nil {
				defaultExpire = int64(constIndex.GetMailDefaultExpire().GetSeconds())
			}
		}
		mail.ExpiredTime = mail.GetStartTime() + defaultExpire
	}

	mail.ResolveExpiredTime = 0 // 个人邮件不允许重置过期时间
	if mail.Reason == nil {
		mail.Reason = &public_protocol_pbdesc.DMailFlowReason{}
	}
	mail.Reason.MajorReason = reason.MajorReason
	mail.Reason.MinorReason = reason.MinorReason
	mail.Reason.Parameter = reason.Parameter

	// 创建邮件记录
	mailRecord := &public_protocol_pbdesc.DMailRecord{
		MailId:             mail.GetMailId(),
		MajorType:          mail.GetMajorType(),
		MinorType:          mail.GetMinorType(),
		Status:             int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_NONE),
		DeliveryTime:       mail.GetDeliveryTime(),
		StartTime:          mail.GetStartTime(),
		ShowTime:           mail.GetShowTime(),
		ExpiredTime:        mail.GetExpiredTime(),
		ResolveExpiredTime: mail.GetResolveExpiredTime(),
		IsGlobalMail:       false,
	}

	// DB: add mail content
	mailTable := &private_protocol_pbdesc.DatabaseTableMailContent{
		MailId: mail.GetMailId(),
	}
	if mailTable.JobData == nil {
		mailTable.JobData = &private_protocol_pbdesc.DatabaseMailContentBlobData{}
	}
	mailTable.JobData.MailContent = mail.Clone()

	// async add mail to user mail box

	JobData := &private_protocol_pbdesc.UserAsyncJobsBlobData{
		Action: &private_protocol_pbdesc.UserAsyncJobsBlobData_AddMail{
			AddMail: &private_protocol_pbdesc.UserAsyncJobMailMessage{
				MailRecord: mailRecord.Clone(),
			},
		},
	}
	jobsType := private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_INVALID

	switch mail.GetMajorType() {
	case int32(public_protocol_common.EnMailMajorType_EN_MAIL_MAJOR_SYSTEM_PAY):
		jobsType = private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_MAIL_IMPORTANT
	case int32(public_protocol_common.EnMailMajorType_EN_MAIL_MAJOR_SYSTEM_ANNOUNCEMENT):
		jobsType = private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_MAIL_IMPORTANT
	default:
		jobsType = private_protocol_pbdesc.EnPlayerAsyncJobsType_EN_PAJT_MAIL_SYSTEM
	}

	ret := db.DatabaseTableMailContentUpdateMailId(ctx, mailTable)
	if ret.IsError() {
		ctx.LogError("add_mail_content for user failed",
			"mail_id", mail.GetMailId(),
			"user_id", userId,
			"error", ret,
		)
		return ret, nil
	}

	ret, _ = async_jobs.AddJobs(ctx, jobsType, userId, zoneId, JobData, async_jobs.DefaultActionOptions())
	ctx.LogInfo("add_mail_content for user success",
		"mail_id", mail.GetMailId(),
		"user_id", userId,
		"zoneId", zoneId,
		"jobsType", jobsType,
		"ret", ret.GetResponseCode(),
	)

	// 返回邮件记录，由调用方处理异步任务
	result := &AddUserMailResult{
		MailId:     mail.GetMailId(),
		MailRecord: mailRecord,
	}

	ctx.LogInfo("add_user_mail success",
		"mail_id", mail.GetMailId(),
		"user_id", userId,
	)
	return cd.CreateRpcResultOk(), result
}

// AddUserMailWithTemplate 使用模板发送用户邮件

func AddUserMailWithTemplate(
	ctx cd.AwaitableContext,
	mailTemplateId int32,
	sender *public_protocol_pbdesc.DMailUserInfo,
	receiver *public_protocol_pbdesc.DMailUserInfo,
	zoneId uint32,
	channel int32,
	channelParam int64,
	attachments []*public_protocol_common.DItemOffset,
	reason *public_protocol_pbdesc.DMailFlowReason,
	extensions map[string]string,
	deliveryTime int64,
	expiredTime int64,
) (cd.RpcResult, *AddUserMailResult) {

	userId := receiver.GetProfile().GetUserId()

	// check user_id
	if userId == 0 || mailTemplateId == 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), nil
	}

	// 渲染邮件模板
	mail := &public_protocol_pbdesc.DMailContent{
		DeliveryTime: deliveryTime,
		ExpiredTime:  expiredTime,
	}

	ret := MailRenderTemplate(mailTemplateId, mail, sender,
		attachments, extensions)
	if ret.IsError() {
		ctx.LogError("mail_render_template failed",
			"mail_template_id", mailTemplateId,
			"user_id", userId,
			"error", ret,
		)
		return ret, nil
	}

	if mail.GetMailId() == 0 {
		guid, result := uuid.GenerateGlobalUniqueID(ctx,
			private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_MAIL,
			private_protocol_pbdesc.EnGlobalUUIDMinorType_EN_GLOBAL_UUID_MIT_DEFAULT,
			private_protocol_pbdesc.EnGlobalUUIDPatchType_EN_GLOBAL_UUID_PT_DEFAULT)
		if result.IsError() {
			return result, nil
		}
		mail.MailId = int64(guid)
	}

	// call AddUserMail()
	return AddUserMail(ctx, userId, zoneId, mail, channel, channelParam, reason)
}
