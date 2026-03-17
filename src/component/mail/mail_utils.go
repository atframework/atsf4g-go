// Copyright 2026 atframework
// @brief 邮件工具函数

package atframework_component_mail

import (
	"strconv"
	"time"

	config "github.com/atframework/atsf4g-go/component/config"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	"google.golang.org/protobuf/proto"
)

// MailIsExpired 检查邮件是否过期
func MailIsExpired(expiredTime int64) bool {
	return expiredTime > 0 && time.Now().Unix() >= expiredTime
}

// MailIsRemoved 检查邮件是否被标记为移除
func MailIsRemoved(record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}
	return (record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0
}

// MailIsExpiredOrRemoved 检查邮件是否过期或被移除
func MailIsExpiredOrRemoved(record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}
	return MailIsRemoved(record) || MailIsExpired(record.GetExpiredTime())
}

// MailIsHistoryRemovable 检查邮件是否可历史移除（过期且超过移除时间）
func MailIsHistoryRemovable(record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}
	now := time.Now().Unix()
	if record.GetRemoveTime() > 0 && now >= record.GetRemoveTime() {
		return true
	}
	return MailIsExpiredOrRemoved(record)
}

// MailPaddingDayTime 邮件时间填充到整天结束
func MailPaddingDayTime(timestamp int64) int64 {
	t := time.Unix(timestamp, 0)
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return startOfDay.Add(24*time.Hour - time.Second).Unix()
}

// MailRenderExtension 邮件渲染扩展类型
type MailRenderExtension map[string]string

// MailRenderTemplate 渲染邮件模板
// @param mailTemplateId 邮件模板ID
// @param mail 输出的邮件内容
// @param sender 发送者信息（可为nil）
// @param itemOffset 道具偏移（可为nil）
// @param extensions 扩展信息（可为nil），存入MailTemplateExtensions
func MailRenderTemplate(
	mailTemplateId int32,
	mail *public_protocol_pbdesc.DMailContent,
	sender *public_protocol_pbdesc.DMailUserInfo,
	itemOffset []*public_protocol_common.DItemOffset,
	extensions map[string]string,
) cd.RpcResult {
	if mail == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 获取邮件模板配置
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	template := configGroup.GetExcelMailTemplateByTypeId(mailTemplateId)
	if template == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	templateConfig := template.GetConfig()
	if templateConfig == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	mail.MailTemplateId = mailTemplateId

	mail.MajorType = templateConfig.GetMajorType()
	mail.MinorType = templateConfig.GetMinorType()

	mail.Title = templateConfig.GetTitle()
	mail.Content = templateConfig.GetContent()

	if extensions != nil {
		if mail.MailTemplateExtensions == nil {
			mail.MailTemplateExtensions = make(map[string]string)
		}
		for k, v := range extensions {
			mail.MailTemplateExtensions[k] = v
		}
	}

	if sender != nil {
		mail.Sender = sender.Clone()
	}

	if len(itemOffset) > 0 {
		for i, offset := range itemOffset {
			mail.AttachmentsOffset = append(mail.GetAttachmentsOffset(), &public_protocol_pbdesc.DMailItemOffset{
				Index: int32(i),
				Item:  offset.Clone(),
			})
		}
	} else {
		for i, attachment := range templateConfig.GetAttachments() {
			mail.AttachmentsOffset = append(mail.GetAttachmentsOffset(), &public_protocol_pbdesc.DMailItemOffset{
				Index: int32(i),
				Item: &public_protocol_common.DItemOffset{
					TypeId: attachment.GetTypeId(),
					Count:  attachment.GetCount(),
				},
			})
		}
	}

	if mail.GetExpiredTime() <= 0 && templateConfig.GetExpiredDurationS() > 0 {
		now := time.Now().Unix()
		mail.ExpiredTime = now + int64(templateConfig.GetExpiredDurationS())
	}

	return cd.CreateRpcResultOk()
}

// FormatMailTitle 格式化邮件标题
func FormatMailTitle(template string, params map[string]string) string {
	result := template
	for k, v := range params {
		placeholder := "{" + k + "}"
		result = replaceAll(result, placeholder, v)
	}
	return result
}

// FormatMailContent 格式化邮件内容
func FormatMailContent(template string, params map[string]string) string {
	return FormatMailTitle(template, params)
}

// replaceAll 替换所有匹配的字符串
func replaceAll(s, old, new string) string {
	result := s
	for {
		idx := indexOf(result, old)
		if idx < 0 {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

// indexOf 查找子字符串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Int32ToStr 整数转字符串
func Int32ToStr(n int32) string {
	return strconv.Itoa(int(n))
}

// Int64ToStr 整数转字符串
func Int64ToStr(n int64) string {
	return strconv.FormatInt(n, 10)
}

// MailIsValidRecord 检查邮件记录是否有效
// 返回0表示有效，否则返回错误码
func MailIsValidRecord(record *public_protocol_pbdesc.DMailRecord) int32 {
	if record == nil || record.GetMailId() == 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	if record.GetMajorType() == 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 获取时间容忍度配置
	timeTolerate := int64(0)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup != nil {
		constIndex := configGroup.GetCustomIndex().GetConstIndex()
		if constIndex != nil && constIndex.GetMailTimeTolerate() != nil {
			timeTolerate = constIndex.GetMailTimeTolerate().GetSeconds()
		}
	}

	now := time.Now().Unix()
	if now > 0 && now > record.GetExpiredTime()+timeTolerate {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_EXPIRED)
	}

	if (record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	return 0
}

// MailIsValidContent 检查邮件内容是否有效
// 返回0表示有效，否则返回错误码
func MailIsValidContent(content *public_protocol_pbdesc.DMailContent, expiredTime int64) int32 {
	if content == nil || content.GetMailId() == 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	if content.GetMajorType() == 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 获取时间容忍度配置
	timeTolerate := int64(0)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup != nil {
		constIndex := configGroup.GetCustomIndex().GetConstIndex()
		if constIndex != nil && constIndex.GetMailTimeTolerate() != nil {
			timeTolerate = constIndex.GetMailTimeTolerate().GetSeconds()
		}
	}

	now := time.Now().Unix()
	if now > 0 && now > expiredTime+timeTolerate {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_EXPIRED)
	}

	if now+timeTolerate < content.GetStartTime() {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_START)
	}

	if (content.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	return 0
}

// MailIsExpiredOrRemovedContent 检查邮件内容是否过期或被移除
func MailIsExpiredOrRemovedContent(content *public_protocol_pbdesc.DMailContent) bool {
	if content == nil || content.GetMailId() == 0 {
		return true
	}

	if content.GetMajorType() == 0 {
		return true
	}

	timeTolerate := int64(0)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup != nil {
		constIndex := configGroup.GetCustomIndex().GetConstIndex()
		if constIndex != nil && constIndex.GetMailTimeTolerate() != nil {
			timeTolerate = constIndex.GetMailTimeTolerate().GetSeconds()
		}
	}

	now := time.Now().Unix()
	if now > 0 && now > content.GetExpiredTime()+timeTolerate {
		return true
	}

	if (content.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return true
	}

	return false
}

// MailIsRemovedContent 检查邮件内容是否被移除
func MailIsRemovedContent(content *public_protocol_pbdesc.DMailContent) bool {
	if content == nil || content.GetMailId() == 0 {
		return true
	}

	if content.GetMajorType() == 0 {
		return true
	}

	timeTolerate := int64(0)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup != nil {
		constIndex := configGroup.GetCustomIndex().GetConstIndex()
		if constIndex != nil && constIndex.GetMailTimeTolerate() != nil {
			timeTolerate = constIndex.GetMailTimeTolerate().GetSeconds()
		}
	}

	now := time.Now().Unix()
	if now > 0 && now > content.GetRemoveTime()+timeTolerate {
		return true
	}

	if (content.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return true
	}

	return false
}

// MailIsHistoryRemovableContent 检查邮件内容是否可历史移除
func MailIsHistoryRemovableContent(content *public_protocol_pbdesc.DMailContent) bool {
	if content == nil || content.GetMailId() == 0 {
		return true
	}

	if content.GetMajorType() == 0 {
		return true
	}

	timeTolerate := int64(0)
	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	if configGroup != nil {
		constIndex := configGroup.GetCustomIndex().GetConstIndex()
		if constIndex != nil && constIndex.GetMailTimeTolerate() != nil {
			timeTolerate = constIndex.GetMailTimeTolerate().GetSeconds()
		}
	}

	now := time.Now().Unix()
	maxTime := content.GetExpiredTime()
	if content.GetRemoveTime() > maxTime {
		maxTime = content.GetRemoveTime()
	}
	if now > 0 && now > maxTime+timeTolerate {
		return true
	}

	return false
}

// MailMergeContentAndRecord 合并邮件内容和记录
func MailMergeContentAndRecord(out *public_protocol_pbdesc.DMailContent,
	content *public_protocol_pbdesc.DMailContent,
	record *public_protocol_pbdesc.DMailRecord) {
	if out == nil || content == nil || record == nil {
		return
	}

	if out != content {
		proto.Merge(out, content)
	}

	out.Status = record.GetStatus()

	out.DeliveryTime = record.GetDeliveryTime()
	out.StartTime = record.GetStartTime()
	out.ShowTime = record.GetShowTime()
	out.ExpiredTime = record.GetExpiredTime()
	out.ResolveExpiredTime = record.GetResolveExpiredTime()

	removeTime := record.GetExpiredTime()
	if record.GetRemoveTime() > removeTime {
		removeTime = record.GetRemoveTime()
	}
	out.RemoveTime = removeTime
}

// MailFillAdminSender 填充系统管理员发送者信息
func MailFillAdminSender(sender *public_protocol_pbdesc.DMailUserInfo) {
	if sender == nil {
		return
	}

	sender.AccountType = uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER)
	sender.AccountId = 0
	if sender.Profile == nil {
		sender.Profile = &public_protocol_common.DUserIDKey{}
	}
	sender.Profile.UserId = 0
	sender.Profile.ZoneId = 0
}

// MailPaddingDayTimeWithOffset 邮件时间填充到整天偏移时间
// @param in 输入时间戳
// @param off 每日偏移秒数（如每天5点刷新则为5*3600）
func MailPaddingDayTimeWithOffset(in int64, off int64) int64 {
	t := time.Unix(in, 0)
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	ret := startOfDay.Unix() + off

	if ret < in {
		ret += 24 * 3600
	}

	return ret
}

// MailRenderData 邮件渲染数据
type MailRenderData struct {
	MailContent *public_protocol_pbdesc.DMailContent
	Sender      *public_protocol_pbdesc.DMailUserInfo
	Receiver    *public_protocol_pbdesc.DMailUserInfo
	Extensions  map[string]string
}

// 返回0表示有效，否则返回错误码
func MailIsValid(content *public_protocol_pbdesc.DMailContent, expiredTime int64) int32 {
	if content == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}
	// 检查状态
	if (content.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}
	// 检查过期
	if MailIsExpired(expiredTime) {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_EXPIRED)
	}
	// 检查start_time
	now := time.Now().Unix()
	if content.GetStartTime() > 0 && now < content.GetStartTime() {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_START)
	}
	return 0
}

// IsMailShown 检查邮件是否应该显示
func IsMailShown(content *public_protocol_pbdesc.DMailContent, record *public_protocol_pbdesc.DMailRecord) bool {
	if content == nil || record == nil {
		return false
	}
	if (content.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
		return false
	}
	return !MailIsExpired(record.GetExpiredTime())
}
