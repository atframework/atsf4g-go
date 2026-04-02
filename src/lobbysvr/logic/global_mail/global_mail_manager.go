package lobbysvr_logic_global_mail

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	global_mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/data"
	impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/impl"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
	"github.com/atframework/libatapp-go"
)

// GlobalMailManager 全局邮件管理器接口
type GlobalMailManager interface {
	libatapp.AppModuleImpl

	// IsAsyncTaskRunning 检查异步任务是否正在运行
	IsAsyncTaskRunning() bool

	// TryToStartAsyncJobs 尝试启动异步任务
	TryToStartAsyncJobs()

	// ResetAsyncJobsProtect 重置异步任务保护
	ResetAsyncJobsProtect()

	// UpdateFromDB 从数据库更新全局邮件
	UpdateFromDB(ctx cd.RpcContext, zoneId uint32, majorType int32, blobData *private_protocol_pbdesc.DatabaseGlobalMailBlobData, rewriteDbData bool) bool

	// IsHistoryRemoveable 检查邮件是否可从历史记录移除
	IsHistoryRemoveable(record *public_protocol_pbdesc.DMailRecord) bool

	// IsRecordRemoveable 检查邮件是否可从邮箱移除
	IsRecordRemoveable(ctx cd.RpcContext, record *public_protocol_pbdesc.DMailRecord) bool

	// UpdateGlobalMailRecord 更新全服邮件记录
	UpdateGlobalMailRecord(ctx cd.RpcContext, dst *public_protocol_pbdesc.DMailRecord, src *public_protocol_pbdesc.DMailRecord)

	// AddGlobalMail 添加全服邮件
	AddGlobalMail(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) int32

	// UpdateGlobalMail 更新全服邮件
	UpdateGlobalMail(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) bool

	// GetMailRaw 获取邮件原始数据
	GetMailRaw(mailId int64) *mail_data.MailData

	// GetMailBoxByType 按类型获取邮件箱
	GetMailBoxByType(zoneId uint32, majorType int32) *mail_data.MailBox

	// GetAllGlobalMails 获取所有全局邮件
	GetAllGlobalMails() global_mail_data.GlobalMailBox

	// RemoveGlobalMail 移除全服邮件
	RemoveGlobalMail(ctx cd.RpcContext, mailId int64)

	// FetchAllUnloadedMails 获取所有未加载内容的邮件ID
	FetchAllUnloadedMails(ctx cd.RpcContext) []int64

	// SetMailContentLoaded 设置邮件内容已加载
	SetMailContentLoaded(mailId int64)

	// SetLastSuccessFetchTimepoint 设置最后成功拉取时间点
	SetLastSuccessFetchTimepoint(t int64)

	// GetPendingToRemoveContents 获取待移除内容的邮件ID集合
	GetPendingToRemoveContents() map[int64]struct{}

	// ClearPendingToRemoveContents 清除待移除内容的邮件ID集合
	ClearPendingToRemoveContents()

	// GetPendingToRemoveContentsList 获取待移除内容的邮件ID列表（返回副本）
	GetPendingToRemoveContentsList() []int64

	// RemovePendingToRemoveContent 从待移除内容列表中移除指定邮件ID
	RemovePendingToRemoveContent(mailId int64)
}

// CreateGlobalMailManager 创建全局邮件管理器实例
func CreateGlobalMailManager(owner libatapp.AppImpl) *impl.GlobalMailManager {
	return impl.NewGlobalMailManager(owner)
}

func GetUserRouterManager(app libatapp.AppImpl) *impl.GlobalMailManager {
	return libatapp.AtappGetModule[*impl.GlobalMailManager](app)
}
