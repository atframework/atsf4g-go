package lobbysvr_logic_mail

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
)

// UserMailManager 用户邮件管理器接口
type UserMailManager interface {
	data.UserModuleManagerImpl

	IsDirty() bool
	ClearDirty()

	ResetGlobalMailsCache()

	GetMailRaw(mailId int64) *mail_data.MailData
	// GetMailBoxByMajorType 根据major_type获取邮件箱
	GetMailBoxByMajorType(majorType int32) *mail_data.MailBox

	// AddMail 添加邮件
	AddMail(ctx cd.RpcContext, mail *public_protocol_pbdesc.DMailRecord, content *public_protocol_pbdesc.DMailContent) int32
	// AddGlobalMail 添加全服邮件
	AddGlobalMail(ctx cd.RpcContext, mail *public_protocol_pbdesc.DMailRecord, content *public_protocol_pbdesc.DMailContent) int32

	// RemoveMail 移除邮件
	RemoveMail(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult) int32
	// ReadMail 读取邮件
	ReadMail(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) int32
	// ReadAll 读取所有邮件
	ReadAll(ctx cd.RpcContext, majorType int32, minorType int32, needRemove bool) ([]*public_protocol_pbdesc.DMailOperationResult, int32)

	// ReceiveMailAttachments 领取邮件附件
	ReceiveMailAttachments(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) cd.RpcResult
	// ReceiveMailAttachmentsAll 领取邮件所有附件
	ReceiveMailAttachmentsAll(ctx cd.RpcContext, needRemove bool) ([]*public_protocol_pbdesc.DMailOperationResult, cd.RpcResult)

	// PackMailUser 打包邮件用户信息
	PackMailUser(userInfo *public_protocol_pbdesc.DMailUserInfo)

	// SendAllSyncData 发送所有同步消息
	SendAllSyncData(ctx cd.RpcContext)

	// MutableDirtyMail 获取可变的脏邮件记录
	MutableDirtyMail(record *public_protocol_pbdesc.DMailRecord, isNew bool) *public_protocol_pbdesc.DMailRecord

	// RemoveExpiredMails 移除过期邮件
	RemoveExpiredMails(ctx cd.RpcContext)

	// FetchAllUserMailIds 获取所有用户邮件ID
	FetchAllUserMailIds() []int64

	// MergeGlobalMails 合并全局邮件（移除过期全服邮件，补全全服邮件内容）
	MergeGlobalMails(ctx cd.RpcContext) (cd.RpcResult, int32)

	// FetchAllUnloadedMails 获取所有未加载内容的邮件ID
	FetchAllUnloadedMails(ctx cd.RpcContext) []int64

	// SetMailContentLoaded 设置邮件内容已加载
	SetMailContentLoaded(ctx cd.RpcContext, mailId int64)

	// RemoveUserMail 移除用户邮件
	RemoveUserMail(ctx cd.RpcContext, mailId int64)

	// GetPendingRemoveList 获取待移除邮件列表
	GetPendingRemoveList() []int64

	GetMailRedPoint() bool

	RemovePendingRemoveItem(mailId int64)

	GetLazySaveCounter() int

	IncrementLazySaveCounter() int

	ResetLazySaveCounter()

	TryToStartAsyncJobs(ctx cd.RpcContext)

	WaitForAsyncTask(ctx cd.AwaitableContext) cd.RpcResult
}
