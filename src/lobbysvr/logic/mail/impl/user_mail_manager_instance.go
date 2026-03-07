package lobbysvr_logic_mail_internal

import (
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	mail_util "github.com/atframework/atsf4g-go/component-mail"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/proto"

	config "github.com/atframework/atsf4g-go/component-config"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	logic_condition "github.com/atframework/atsf4g-go/service-lobbysvr/logic/condition"
	logic_global_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	mail_action "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/action"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	rpc_lobbyclientservice "github.com/atframework/atsf4g-go/service-lobbysvr/rpc/lobbyclientservice"
)

func init() {
	var _ logic_mail.UserMailManager = (*UserMailManager)(nil)
	data.RegisterUserModuleManagerCreator[logic_mail.UserMailManager](func(_ cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserMailManager(owner)
	})
}

// UserMailManager 用户邮件管理器实现
type UserMailManager struct {
	data.UserModuleManagerBase

	isDirty             bool
	isGlobalMailsMerged bool
	lazySaveCounter     int

	mailBoxIdIndex       mail_data.MailIndex          // mail_id -> MailData
	mailBoxTypeIndex     map[int32]*mail_data.MailBox // major_type -> MailBox
	mailBoxUnloadedIndex mail_data.MailIndex          // 未加载内容的邮件索引

	pendingRemoveList   mail_data.MailRecordMap // 待移除队列
	receivedGlobalMails mail_data.MailRecordMap // 已接收的全服邮件

	dirtyCache *mail_data.DirtyCache // 脏数据缓存

	// 异步任务相关
	mailAsyncTask                 lu.AtomicInterface[cd.TaskActionImpl]
	mailAsyncTaskProtectTimepoint time.Time
}

// CreateUserMailManager 创建用户邮件管理器
func CreateUserMailManager(owner *data.User) *UserMailManager {
	mgr := &UserMailManager{
		UserModuleManagerBase:         *data.CreateUserModuleManagerBase(owner),
		isDirty:                       false,
		isGlobalMailsMerged:           false,
		lazySaveCounter:               0,
		mailBoxIdIndex:                make(mail_data.MailIndex),
		mailBoxTypeIndex:              make(map[int32]*mail_data.MailBox),
		mailBoxUnloadedIndex:          make(mail_data.MailIndex),
		pendingRemoveList:             make(mail_data.MailRecordMap),
		receivedGlobalMails:           make(mail_data.MailRecordMap),
		dirtyCache:                    mail_data.NewDirtyCache(),
		mailAsyncTaskProtectTimepoint: time.Time{},
	}
	return mgr
}

func (m *UserMailManager) GetOwner() *data.User {
	return m.UserModuleManagerBase.GetOwner()
}

func (m *UserMailManager) CreateInit(_ctx cd.RpcContext, _versionType uint32) {
}

func (m *UserMailManager) LoginInit(_ctx cd.RpcContext) {
}

func (m *UserMailManager) RefreshLimitSecond(ctx cd.RpcContext) {
	if m.needStartAsyncJobs() {
		m.tryToStartAsyncJobs(ctx)
	}
}

func (m *UserMailManager) InitFromDB(ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {

	m.mailBoxIdIndex = make(mail_data.MailIndex)
	m.mailBoxTypeIndex = make(map[int32]*mail_data.MailBox)
	m.pendingRemoveList = make(mail_data.MailRecordMap)
	m.mailBoxUnloadedIndex = make(mail_data.MailIndex)
	m.receivedGlobalMails = make(mail_data.MailRecordMap)

	m.isDirty = false
	m.isGlobalMailsMerged = false
	m.mailAsyncTaskProtectTimepoint = time.Time{}

	for _, record := range dbUser.GetMailData().GetMailBox() {
		mailDataPtr := &mail_data.MailData{
			Record:  record,
			Content: nil,
		}
		m.mailBoxIdIndex[mailDataPtr.Record.GetMailId()] = mailDataPtr

		mailTypeBox, ok := m.mailBoxTypeIndex[mailDataPtr.Record.GetMajorType()]
		if !ok {
			mailTypeBox = &mail_data.MailBox{
				Mails:             make(map[int64]*mail_data.MailData),
				FutureMailCount:   0,
				FutureCacheExpire: 0,
			}
		}
		mailTypeBox.Mails[mailDataPtr.Record.GetMailId()] = mailDataPtr

		m.mailBoxTypeIndex[mailDataPtr.Record.GetMajorType()] = mailTypeBox

		if mailDataPtr.Content == nil {
			m.mailBoxUnloadedIndex[mailDataPtr.Record.GetMailId()] = mailDataPtr
		}
	}

	for _, mail := range dbUser.GetMailData().GetPendingRemoveList() {
		m.pendingRemoveList[mail.GetMailId()] = mail
	}

	for _, globalMail := range dbUser.GetMailData().GetReceivedGlobalMails() {
		m.receivedGlobalMails[globalMail.GetMailId()] = globalMail
	}

	// 整合邮件
	m.compactMails(ctx)

	return cd.RpcResult{Error: nil, ResponseCode: 0}
}

func (m *UserMailManager) DumpToDB(ctx cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	m.scanAndRemoveExpiredMails(ctx)

	userMailData := dbUser.MutableMailData()

	for _, mail := range m.mailBoxIdIndex {
		if mail != nil && mail.Record != nil {
			userMailData.MailBox = append(userMailData.MailBox, mail.Record)
		}
	}

	for _, mail := range m.pendingRemoveList {
		userMailData.PendingRemoveList = append(userMailData.PendingRemoveList, mail)
	}

	for _, mail := range m.receivedGlobalMails {
		userMailData.ReceivedGlobalMails = append(userMailData.ReceivedGlobalMails, mail)
	}

	return cd.RpcResult{Error: nil, ResponseCode: 0}
}

func (m *UserMailManager) IsDirty() bool {
	return m.isDirty
}

func (m *UserMailManager) ClearDirty() {
	m.isDirty = false
}

func (m *UserMailManager) ResetGlobalMailsCache() {
	m.isGlobalMailsMerged = false
	m.mailAsyncTaskProtectTimepoint = time.Time{}
}

func (m *UserMailManager) GetMailRaw(mailId int64) *mail_data.MailData {
	if mail, ok := m.mailBoxIdIndex[mailId]; ok {
		return mail
	}
	return nil
}

func (m *UserMailManager) GetMailBoxByMajorType(majorType int32) *mail_data.MailBox {
	if mailBox, ok := m.mailBoxTypeIndex[majorType]; ok {
		return mailBox
	}
	return nil
}

// AddMail 添加邮件
func (m *UserMailManager) AddMail(ctx cd.RpcContext, mail *public_protocol_pbdesc.DMailRecord, content *public_protocol_pbdesc.DMailContent) int32 {
	if mail.GetMailId() == 0 || mail.GetMajorType() == 0 {
		ctx.LogError("can not add mail, because mail_id=0 or major_type = 0")
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if existMail, ok := m.mailBoxIdIndex[mail.GetMailId()]; ok {
		ctx.LogWarn("add mail twice", "mail_id", mail.GetMailId())
		now := time.Now().Unix()
		oldIsFuture := false
		newIsFuture := mail.GetStartTime() > now

		if existMail != nil && existMail.Record != nil {
			oldIsFuture = existMail.Record.GetStartTime() > now

			// 保留失败状态
			keepFetchErrorCount := existMail.Record.GetFetchErrorCount()
			if mail.GetIsGlobalMail() {
				m.updateGlobalMailRecord(ctx, existMail.Record, mail)
			} else {
				proto.Merge(existMail.Record, mail)
			}
			existMail.Record.FetchErrorCount = keepFetchErrorCount
		}

		// 添加已过期的邮件则直接进删除队列
		if mail.GetIsGlobalMail() {
			if m.isGlobalMailRecordRemovable(ctx, mail) || mail_util.MailIsExpiredOrRemoved(mail) {
				m.removeUserMail(ctx, mail.GetMailId())
			}
		} else if mail_util.MailIsHistoryRemovable(mail) {
			m.removeUserMail(ctx, mail.GetMailId())
		} else if oldIsFuture != newIsFuture {
			// 邮件起始时间变化，标记未来邮件缓存失效
			if mailBox := m.GetMailBoxByMajorType(mail.GetMajorType()); mailBox != nil {
				mailBox.FutureCacheExpire = 0
			}
		}
		return 0
	}

	// 如果在移除队列中则忽略
	if _, ok := m.pendingRemoveList[mail.GetMailId()]; ok {
		return 0
	}

	// 本身就是过期邮件则忽略
	if mail.GetIsGlobalMail() {
		if mail_util.MailIsExpiredOrRemoved(mail) || m.isGlobalMailRecordRemovable(ctx, mail) {
			return 0
		}
	} else if mail_util.MailIsHistoryRemovable(mail) {
		return 0
	}

	// 创建新邮件数据
	mailDataPtr := &mail_data.MailData{
		Record:  mail.Clone(),
		Content: nil,
	}

	if content != nil {
		mailDataPtr.Content = content.Clone()
	}

	if mail.GetIsGlobalMail() {
		m.updateGlobalMailRecord(ctx, mailDataPtr.Record, mail)
	}

	m.mailBoxIdIndex[mailDataPtr.Record.GetMailId()] = mailDataPtr
	if mailDataPtr.Content == nil {
		m.mailBoxUnloadedIndex[mailDataPtr.Record.GetMailId()] = mailDataPtr
		m.mailAsyncTaskProtectTimepoint = time.Time{}
	}

	// 添加到类型索引
	majorType := mailDataPtr.Record.GetMajorType()
	if _, ok := m.mailBoxTypeIndex[majorType]; !ok {
		m.mailBoxTypeIndex[majorType] = mail_data.NewMailBox()
	}
	mailBox := m.mailBoxTypeIndex[majorType]
	mailBox.Mails[mailDataPtr.Record.GetMailId()] = mailDataPtr

	// 未来邮件索引
	now := time.Now().Unix()
	if mailDataPtr.Record.GetStartTime() > now {
		mailBox.FutureMailCount++
		if mailDataPtr.Record.GetStartTime() < mailBox.FutureCacheExpire {
			mailBox.FutureCacheExpire = mailDataPtr.Record.GetStartTime()
		}
	}

	m.isDirty = true
	ctx.LogDebug("add mail success", "major_type", mail.GetMajorType(), "mail_id", mail.GetMailId())

	m.MutableDirtyMail(mailDataPtr.Record, true)

	// 如果本来就是已过期则加入到移除列表
	if mail.GetIsGlobalMail() {
		if m.isGlobalMailRecordRemovable(ctx, mail) || mail_util.MailIsRemoved(mail) {
			m.removeUserMail(ctx, mail.GetMailId())
		}
	} else if mail_util.MailIsHistoryRemovable(mail) {
		m.removeUserMail(ctx, mail.GetMailId())
	}

	// 刷新未来邮件缓存并检查数量限制
	mail_data.RefreshMailBoxFutureCache(mailBox)
	m.checkAndCompactMailBox(ctx, mailBox)

	return 0
}

// AddGlobalMail 添加全服邮件
func (m *UserMailManager) AddGlobalMail(ctx cd.RpcContext, mail *public_protocol_pbdesc.DMailRecord, content *public_protocol_pbdesc.DMailContent) int32 {
	if !mail.GetIsGlobalMail() {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if existRecord, ok := m.receivedGlobalMails[mail.GetMailId()]; ok {
		// 已存在则刷新数据
		m.updateGlobalMailRecord(ctx, existRecord, mail)

		// 没删除的话也刷新邮件record
		if existMail, ok := m.mailBoxIdIndex[mail.GetMailId()]; ok {
			if existMail != nil && existMail.Record != nil {
				m.updateGlobalMailRecord(ctx, existMail.Record, mail)
			}
			// 添加已过期的邮件则直接进删除队列
			if m.isGlobalMailRecordRemovable(ctx, mail) || mail_util.MailIsRemoved(mail) {
				m.removeUserMail(ctx, mail.GetMailId())
			}
			return 0
		}
		return 0
	}

	ret := m.AddMail(ctx, mail, content)
	if ret < 0 {
		return ret
	}

	m.receivedGlobalMails[mail.GetMailId()] = mail.Clone()

	// TODO: OSS Log
	return ret
}

// RemoveMail 移除邮件
func (m *UserMailManager) RemoveMail(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult) int32 {
	mail := m.GetMailRaw(mailId)
	if mail == nil || mail.Record == nil {
		if out != nil {
			out.Result = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
		}
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	m.removeUserMail(ctx, mailId)
	if out != nil {
		out.Record = mail.Record.Clone()
		out.Result = 0
	}
	return 0
}

// ReadMail 读取邮件
func (m *UserMailManager) ReadMail(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) int32 {
	ret := m.readMailInternal(ctx, mailId, out, needRemove)

	if out != nil {
		if out.Record == nil {
			out.Record = &public_protocol_pbdesc.DMailRecord{}
		}
		out.Result = ret
	}
	return ret
}

// readMailInternal 读取邮件内部逻辑
func (m *UserMailManager) readMailInternal(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) int32 {
	mail := m.GetMailRaw(mailId)

	if mail == nil || mail.Content == nil || mail.Record == nil {
		if !needRemove {
			return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
		}
		return 0
	}

	ret := mail_util.MailIsValid(mail.Content, mail.Record.GetExpiredTime())
	if ret < 0 {
		if ret == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND) && needRemove {
			ret = 0
		}
		return ret
	}

	if (mail.Record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ)) == 0 {
		mail.Record.Status = mail.Record.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ)

		if needRemove {
			m.removeUserMail(ctx, mailId)
		} else {
			m.MutableDirtyMail(mail.Record, false)
		}

		// TODO: OSS Log
	} else if needRemove {
		m.removeUserMail(ctx, mailId)
	}

	out.Record = mail.Record.Clone()

	return 0
}

// ReadAll 读取所有邮件
func (m *UserMailManager) ReadAll(ctx cd.RpcContext, majorType int32, minorType int32, out []*public_protocol_pbdesc.DMailOperationResult, needRemove bool) int32 {
	selectedMailBox := m.GetMailBoxByMajorType(majorType)
	if selectedMailBox == nil {
		return 0
	}

	var mailsToProcess []*mail_data.MailData
	for _, mail := range selectedMailBox.Mails {
		if mail == nil || mail.Record == nil || mail.Content == nil {
			continue
		}

		if mail_util.MailIsValid(mail.Content, mail.Record.GetExpiredTime()) != 0 {
			continue
		}

		if (mail.Record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)) != 0 {
			continue
		}

		if !needRemove && (mail.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ)) != 0 {
			continue
		}

		if minorType != 0 && minorType != mail.Record.GetMinorType() {
			continue
		}

		// 一键已读的删除不能覆盖有附件的邮件
		if needRemove && len(mail.Content.GetAttachmentsOffset()) > 0 {
			continue
		}

		mailsToProcess = append(mailsToProcess, mail)
	}

	for _, mail := range mailsToProcess {
		mail.Record.Status = mail.Record.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ)

		if needRemove {
			m.removeUserMail(ctx, mail.Record.GetMailId())
		} else {
			m.MutableDirtyMail(mail.Record, false)
		}

		// TODO: OSS Log

		if out != nil {
			outRes := &public_protocol_pbdesc.DMailOperationResult{
				Record: mail.Record.Clone(),
				Result: 0,
			}
			out = append(out, outRes)
		}
	}

	return 0
}

// ReceiveMailAttachments 领取邮件附件
func (m *UserMailManager) ReceiveMailAttachments(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) cd.RpcResult {
	ret := m.receiveMailAttachmentsInternal(ctx, mailId, out, needRemove)

	if out != nil {
		if out.Record == nil {
			out.Record = &public_protocol_pbdesc.DMailRecord{}
		}
		out.Result = ret
	}

	// 玩家领取邮件后设置为已读状态
	if ret == 0 && !needRemove {
		m.ReadMail(ctx, mailId, nil, false)
	}

	return cd.RpcResult{Error: nil, ResponseCode: ret}
}

func (m *UserMailManager) ReceiveMailAttachmentsAll(ctx cd.RpcContext, out []*public_protocol_pbdesc.DMailOperationResult, needRemove bool) cd.RpcResult {
	var finalResult cd.RpcResult
	for _, mailId := range m.FetchAllUserMailIds() {
		result := &public_protocol_pbdesc.DMailOperationResult{}

		rpcResult := m.ReceiveMailAttachments(ctx, mailId, result, needRemove)
		if rpcResult.IsError() {
			finalResult = rpcResult
			ctx.LogError("ReceiveMailAttachmentsAll failed", "mail_id", mailId, "ret", rpcResult)
		}
		out = append(out, result)
	}
	if finalResult.Error != nil {
		return finalResult
	}
	return cd.CreateRpcResultOk()
}

// receiveMailAttachmentsInternal 领取邮件附件内部逻辑
func (m *UserMailManager) receiveMailAttachmentsInternal(ctx cd.RpcContext, mailId int64, out *public_protocol_pbdesc.DMailOperationResult, needRemove bool) int32 {
	mail := m.GetMailRaw(mailId)

	if mail == nil || mail.Content == nil || mail.Record == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)
	}

	ret := mail_util.MailIsValid(mail.Content, mail.Record.GetExpiredTime())
	if ret < 0 {
		return ret
	}

	if (mail.Record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_TOKEN_ATTACHMENT)) == 0 {
		result := m.sendMailAttachments(ctx, mail, needRemove)
		if result.IsError() {
			ctx.LogError("receive_mail_attachments failed", "mail_id", mailId, "ret", result)
			return result.GetResponseCode()
		}
		ctx.LogWarn("receive_mail_attachments not fully implemented", "mail_id", mailId)

		if needRemove {
			m.removeUserMail(ctx, mailId)
		} else {
			m.MutableDirtyMail(mail.Record, false)
		}
	} else {
		if needRemove {
			m.removeUserMail(ctx, mailId)
		}
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_ALREADY_RECEIVED)
	}

	if out != nil {
		out.Record = mail.Record.Clone()
	}
	return ret
}

// PackMailUser 打包邮件用户信息
func (m *UserMailManager) PackMailUser(userInfo *public_protocol_pbdesc.DMailUserInfo) {
	if userInfo == nil {
		return
	}
	owner := m.GetOwner()
	if owner == nil {
		return
	}

	// TODO: 需要从owner获取账户信息
	// userInfo.Profile = proto.Clone(owner.get_account_info().profile())
	// userInfo.AccountId = owner.get_account_info().account_id()
	// userInfo.AccountType = owner.get_account_info().account_type()
	// userInfo.LoginChannel = owner.get_account_info().account_login_channel()
}

// SendAllSyncData 发送所有同步消息
func (m *UserMailManager) SendAllSyncData(ctx cd.RpcContext) {
	if len(m.dirtyCache.DirtyMails) == 0 {
		return
	}

	if m.IsAsyncTaskRunning() {
		return
	}

	// m.GetOwner().SendAllSyncData(ctx)
	syncData := &service_protocol.SCMailChangeSync{}

	hasData := false
	for mailId, record := range m.dirtyCache.DirtyMails {
		if _, ok := m.dirtyCache.NewMails[mailId]; ok {
			continue
		}
		hasData = true
		syncData.Mails = append(syncData.Mails, record)
		break
	}

	// 新邮件处理
	needAsyncJob := m.needStartAsyncJobs()
	for mailId := range m.dirtyCache.NewMails {
		idIter, ok := m.mailBoxIdIndex[mailId]
		if !ok {
			delete(m.dirtyCache.NewMails, mailId)
			continue
		}

		if idIter.Record == nil {
			delete(m.dirtyCache.NewMails, mailId)
			continue
		}

		// 未拉取邮件内容，需要重新拉取
		if idIter.Content == nil {
			if _, ok := m.mailBoxUnloadedIndex[mailId]; ok {
				needAsyncJob = true
			} else {
				ctx.LogError("dirty mail data", "mail_id", mailId)
				delete(m.dirtyCache.NewMails, mailId)
			}
			continue
		}

		peddingAddMail := public_protocol_pbdesc.DMailContent{}
		peddingAddMail = *idIter.Content.Clone()
		syncData.NewMails = append(syncData.NewMails, &peddingAddMail)

		delete(m.dirtyCache.NewMails, mailId)
		hasData = true
	}

	// 没有数据则放弃发送
	if hasData {
		m.clearDirtyCache(ctx)
		// 发送邮件变更同步消息给客户端
		owner := m.GetOwner()
		if owner != nil {
			session := owner.GetSession()
			if session != nil {
				err := rpc_lobbyclientservice.SendMailChangeSync(session, syncData, 0)
				if err != nil {
					ctx.LogError("send mail change sync failed", "error", err)
				}
			}
		}
	}

	// 存在未拉取的全服邮件
	if needAsyncJob {
		m.tryToStartAsyncJobs(ctx)
	}
}

func (m *UserMailManager) clearDirtyCache(ctx cd.RpcContext) {
	if m.IsAsyncTaskRunning() {
		return
	}
	m.dirtyCache.DirtyMails = make(map[int64]*public_protocol_pbdesc.DMailRecord)
}

// MutableDirtyMail 获取可变的脏邮件记录
func (m *UserMailManager) MutableDirtyMail(record *public_protocol_pbdesc.DMailRecord, isNew bool) *public_protocol_pbdesc.DMailRecord {
	if record == nil {
		return nil
	}
	ret := record.Clone()
	m.dirtyCache.DirtyMails[record.GetMailId()] = ret

	if isNew {
		m.dirtyCache.NewMails[record.GetMailId()] = struct{}{}
	}

	return ret
}

// RemoveExpiredMails 移除过期邮件
func (m *UserMailManager) RemoveExpiredMails(ctx cd.RpcContext) {
	m.scanAndRemoveExpiredMails(ctx)
}

// FetchAllUserMailIds 获取所有用户邮件ID
func (m *UserMailManager) FetchAllUserMailIds() []int64 {
	var out []int64
	for _, mail := range m.mailBoxIdIndex {
		if mail != nil && mail.Record != nil && !mail.Record.GetIsGlobalMail() {
			out = append(out, mail.Record.GetMailId())
		}
	}
	return out
}

// ========== 私有方法 ==========

// refreshMailBoxFutureCache 刷新邮件箱的未来邮件缓存
func (m *UserMailManager) refreshMailBoxFutureCache(mailBox *mail_data.MailBox) {
	mail_data.RefreshMailBoxFutureCache(mailBox)
}

// compactMails 压缩所有类型的邮件
func (m *UserMailManager) compactMails(ctx cd.RpcContext) {
	// 复制索引，内部有删除操作不能直接迭代成员变量
	copyIndex := make(map[int32]*mail_data.MailBox)
	for k, v := range m.mailBoxTypeIndex {
		copyIndex[k] = v
	}
	for _, mailBox := range copyIndex {
		if mailBox != nil {
			m.compactMailsForBox(ctx, mailBox)
		}
	}
}

// checkAndCompactMailBox 检查并压缩邮件箱
func (m *UserMailManager) checkAndCompactMailBox(ctx cd.RpcContext, mailBox *mail_data.MailBox) {
	if mailBox == nil {
		return
	}

	futureReserveCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUserMailFutureReserveMaxCountPerMajorType()
	maxMailCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUserMailMaxCountPerMajorType()

	if futureReserveCount > mailBox.FutureMailCount {
		futureReserveCount = mailBox.FutureMailCount
	}
	if futureReserveCount < 0 {
		futureReserveCount = 0
	}

	if maxMailCount > 0 && int32(len(mailBox.Mails)) > maxMailCount+futureReserveCount {
		m.compactMailsForBox(ctx, mailBox)
	}
}

// compactMailsForBox 压缩指定邮件箱的邮件
func (m *UserMailManager) compactMailsForBox(ctx cd.RpcContext, mailBox *mail_data.MailBox) {
	m.refreshMailBoxFutureCache(mailBox)

	maxMailCountPerMajorType := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUserMailMaxCountPerMajorType()
	if len(mailBox.Mails) == 0 || maxMailCountPerMajorType <= 0 {
		return
	}

	futureReserveCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUserMailFutureReserveMaxCountPerMajorType()
	if futureReserveCount > mailBox.FutureMailCount {
		futureReserveCount = mailBox.FutureMailCount
	}
	if futureReserveCount < 0 {
		futureReserveCount = 0
	}

	maxMailCount := int(maxMailCountPerMajorType) + int(futureReserveCount)
	if len(mailBox.Mails) <= maxMailCount {
		return
	}

	deliveryTimeMaxOffset := int64(config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetMailCompactDeliveryTimeMaxOffset().GetSeconds()) // logic_config::me()->get_logic().mail().compact_delivery_time_max_offset().seconds()

	var pendingRemoveList []int64
	var selectMails []*mail_data.MailData

	for mailId, mail := range mailBox.Mails {
		if mail == nil || mail.Record == nil {
			pendingRemoveList = append(pendingRemoveList, mailId)
			continue
		}

		// 全服邮件在失效时就可以移除邮件内容但保留receivedGlobalMails
		if mail.Record.GetIsGlobalMail() {
			if m.isGlobalMailRecordRemovable(ctx, mail.Record) || mail_util.MailIsRemoved(mail.Record) {
				pendingRemoveList = append(pendingRemoveList, mailId)
				continue
			}
		} else if mail_util.MailIsHistoryRemovable(mail.Record) {
			pendingRemoveList = append(pendingRemoveList, mailId)
			continue
		}

		selectMails = append(selectMails, mail)
	}

	if len(selectMails) > maxMailCount {
		rmcnt := len(selectMails) - maxMailCount
		mail_data.SortMailDataByCompactTime(selectMails, deliveryTimeMaxOffset)
		for i := 0; i < rmcnt; i++ {
			pendingRemoveList = append(pendingRemoveList, selectMails[i].Record.GetMailId())
		}
	}

	for _, mailId := range pendingRemoveList {
		m.removeUserMail(ctx, mailId)
	}
}

// scanAndRemoveExpiredMails 扫描并移除过期邮件
func (m *UserMailManager) scanAndRemoveExpiredMails(ctx cd.RpcContext) {
	var removeIds []int64
	for _, mail := range m.mailBoxIdIndex {
		if mail != nil && mail.Record != nil {
			if mail.Record.GetIsGlobalMail() {
				if m.isGlobalMailRecordRemovable(ctx, mail.Record) || mail_util.MailIsRemoved(mail.Record) {
					removeIds = append(removeIds, mail.Record.GetMailId())
				}
			} else if mail_util.MailIsHistoryRemovable(mail.Record) {
				removeIds = append(removeIds, mail.Record.GetMailId())
			}
		}
	}

	for _, mailId := range removeIds {
		m.removeUserMail(ctx, mailId)
	}

	// 移除过期的全服邮件记录
	removeIds = nil
	for mailId, record := range m.receivedGlobalMails {
		if m.isGlobalMailHistoryRemovable(ctx, record) {
			removeIds = append(removeIds, mailId)
		}
	}

	for _, mailId := range removeIds {
		delete(m.receivedGlobalMails, mailId)
		m.removeUserMail(ctx, mailId)
	}

	m.compactMails(ctx)
}

// removeUserMail 移除用户邮件
func (m *UserMailManager) removeUserMail(ctx cd.RpcContext, mailId int64) {
	// 从待拉取的新邮件集合中移除
	delete(m.dirtyCache.NewMails, mailId)

	idIter, ok := m.mailBoxIdIndex[mailId]
	if !ok {
		return
	}

	if idIter == nil || idIter.Record == nil {
		ctx.LogError("mail.record should not be null")
		delete(m.mailBoxIdIndex, mailId)
		delete(m.mailBoxUnloadedIndex, mailId)
		return
	}

	m.isDirty = true

	// 从类型索引中移除
	if typeIter, ok := m.mailBoxTypeIndex[idIter.Record.GetMajorType()]; ok {
		if typeIter != nil {
			// 重置未来邮件缓存
			if typeIter.FutureCacheExpire <= idIter.Record.GetStartTime() {
				typeIter.FutureCacheExpire = 0
			}
			delete(typeIter.Mails, mailId)
			if len(typeIter.Mails) == 0 {
				delete(m.mailBoxTypeIndex, idIter.Record.GetMajorType())
			}
		} else {
			delete(m.mailBoxTypeIndex, idIter.Record.GetMajorType())
		}
	}

	idIter.Record.Status = idIter.Record.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)
	m.MutableDirtyMail(idIter.Record, false)

	ctx.LogDebug("remove mail success", "major_type", idIter.Record.GetMajorType(), "mail_id", idIter.Record.GetMailId())

	if !idIter.Record.GetIsGlobalMail() {
		m.pendingRemoveList[mailId] = idIter.Record.Clone()
		m.mailAsyncTaskProtectTimepoint = time.Time{}
	}

	delete(m.mailBoxUnloadedIndex, mailId)
	delete(m.mailBoxIdIndex, mailId)

	// TODO: OSS LOG
}

// needStartAsyncJobs 检查是否需要启动异步任务
func (m *UserMailManager) needStartAsyncJobs() bool {
	if time.Now().Before(m.mailAsyncTaskProtectTimepoint) {
		return false
	}
	return len(m.pendingRemoveList) > 0 || !m.isGlobalMailsMerged || len(m.mailBoxUnloadedIndex) > 0
}

// tryToStartAsyncJobs 尝试启动异步任务
func (m *UserMailManager) tryToStartAsyncJobs(ctx cd.RpcContext) {
	m.TryToStartAsyncJobs(ctx)
}

// TryToStartAsyncJobs 尝试启动异步任务
func (m *UserMailManager) TryToStartAsyncJobs(ctx cd.RpcContext) {
	owner := m.GetOwner()
	if owner == nil {
		return
	}

	if !owner.IsWriteable() {
		return
	}

	if !m.needStartAsyncJobs() {
		return
	}

	if m.IsAsyncTaskRunning() {
		return
	}

	timeoutDuration := 30 * time.Second // TODO: 从配置获取
	m.mailAsyncTaskProtectTimepoint = time.Now().Add(timeoutDuration + time.Second)

	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](ctx.GetApp())
	if d == nil {
		ctx.LogError("TryToStartAsyncJobs failed: NoMessageDispatcher not found")
		return
	}

	mailAsyncTask, startData := cd.CreateNoMessageTaskAction(
		d, d.CreateRpcContext(), m.GetOwner().GetActorExecutor(),
		func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *mail_action.TaskActionMailAsyncJobs {
			return mail_action.CreateTaskActionMailAsyncJobs(rd, actor, owner, m, timeout)
		},
	)

	err := libatapp.AtappGetModule[*cd.TaskManager](ctx.GetApp()).StartTaskAction(ctx, mailAsyncTask, &startData)
	if err != nil {
		ctx.LogError("TryToStartAsyncJobs StartTaskAction failed", "error", err, "user_id", owner.GetUserId())
	} else {
		m.mailAsyncTask.Store(mailAsyncTask)
		ctx.LogDebug("TryToStartAsyncJobs started", "user_id", owner.GetUserId())
	}
}

// IsAsyncTaskRunning 检查异步任务是否正在运行
func (m *UserMailManager) IsAsyncTaskRunning() bool {
	task := m.mailAsyncTask.Load()
	if lu.IsNil(task) {
		return false
	}
	if task.IsExiting() {
		m.mailAsyncTask.Store(nil)
		return false
	}
	return true
}

// ========== 异步任务相关的公开接口 ==========

// MergeGlobalMails 合并全局邮件（移除过期全服邮件，补全全服邮件内容）
func (m *UserMailManager) MergeGlobalMails(ctx cd.RpcContext) (cd.RpcResult, int32) {
	ret := int32(0)
	if m.isGlobalMailsMerged {
		return cd.CreateRpcResultOk(), ret
	}

	// 收集无效的全服邮件
	invalidMails := make(map[int64]*mail_data.MailData)
	for mailId, mail := range m.mailBoxIdIndex {
		if mail != nil && mail.Record != nil && mail.Record.GetIsGlobalMail() {
			invalidMails[mailId] = mail
		}
	}

	// 先移除无效邮件，防止无效的全服邮件顶替掉现有个人邮件
	for _, mail := range logic_global_mail.GetUserRouterManager(ctx.GetApp()).GetAllGlobalMails() {
		if mail.MailData.Record != nil {
			delete(invalidMails, mail.MailData.Record.GetMailId())
		}
	}

	// 移除过期的全服邮件
	for mailId, mail := range invalidMails {
		if m.isGlobalMailRecordRemovable(ctx, mail.Record) {
			m.removeUserMail(ctx, mailId)
			ret++
		}
	}

	conditionMgr := data.UserGetModuleManager[logic_condition.UserConditionManager](m.GetOwner())
	if conditionMgr == nil {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_UNKNOWN), 0
	}

	for _, mail := range logic_global_mail.GetUserRouterManager(ctx.GetApp()).GetAllGlobalMails() {
		if mail.MailData.Content != nil {
			// 检查各类限制后添加
			if len(mail.MailData.Content.MailLimit) > 0 {
				readOnlyRuls := []*public_protocol_common.Readonly_DConditionRule{}
				for _, rule := range mail.MailData.Content.MailLimit {
					readOnlyRuls = append(readOnlyRuls, rule.ToReadonly())
				}
				ok := conditionMgr.CheckRules(ctx, readOnlyRuls, logic_condition.CreateEmptyRuleCheckerRuntime())
				if !ok.IsOK() {
					return ok, ret
				}
			}
			m.AddGlobalMail(ctx, mail.MailData.Record, mail.MailData.Content)
			ret++
		}
	}

	m.isGlobalMailsMerged = true
	return cd.CreateRpcResultOk(), ret
}

// FetchAllUnloadedMails 获取所有未加载内容的邮件ID
func (m *UserMailManager) FetchAllUnloadedMails(ctx cd.RpcContext) []int64 {
	var invalidMailIds []int64
	var out []int64

	for mailId, checked := range m.mailBoxUnloadedIndex {
		if checked != nil && checked.Record != nil && checked.Content == nil {
			// 如果是全服邮件的话检查一下是否已经失效
			if checked.Record.GetIsGlobalMail() &&
				(m.isGlobalMailRecordRemovable(ctx, checked.Record)) {
				invalidMailIds = append(invalidMailIds, mailId)
			} else {
				out = append(out, mailId)
			}
		} else {
			invalidMailIds = append(invalidMailIds, mailId)
		}
	}

	for _, invalidMailId := range invalidMailIds {
		delete(m.mailBoxUnloadedIndex, invalidMailId)
	}

	return out
}

// SetMailContentLoaded 设置邮件内容已加载
func (m *UserMailManager) SetMailContentLoaded(ctx cd.RpcContext, mailId int64) {
	delete(m.mailBoxUnloadedIndex, mailId)

	mail := m.GetMailRaw(mailId)

	if _, ok := m.dirtyCache.NewMails[mailId]; ok {
		if mail != nil && mail.Record != nil {
			m.MutableDirtyMail(mail.Record, false)
		}
	}
}

// RemoveUserMail 移除用户邮件（公开方法）
func (m *UserMailManager) RemoveUserMail(ctx cd.RpcContext, mailId int64) {
	m.removeUserMail(ctx, mailId)
}

// GetPendingRemoveList 获取待移除邮件列表
func (m *UserMailManager) GetPendingRemoveList() []int64 {
	result := make([]int64, 0, len(m.pendingRemoveList))
	for mailId := range m.pendingRemoveList {
		result = append(result, mailId)
	}
	return result
}

func (m *UserMailManager) RemovePendingRemoveItem(mailId int64) {
	delete(m.pendingRemoveList, mailId)
}

func (m *UserMailManager) GetLazySaveCounter() int {
	return m.lazySaveCounter
}

func (m *UserMailManager) IncrementLazySaveCounter() int {
	m.lazySaveCounter++
	return m.lazySaveCounter
}

func (m *UserMailManager) ResetLazySaveCounter() {
	m.lazySaveCounter = 0
}

func (m *UserMailManager) updateGlobalMailRecord(ctx cd.RpcContext, dst *public_protocol_pbdesc.DMailRecord, src *public_protocol_pbdesc.DMailRecord) {
	if dst == nil || src == nil {
		return
	}
	logic_global_mail.GetUserRouterManager(ctx.GetApp()).UpdateGlobalMailRecord(dst, src)
}

// isGlobalMailRecordRemovable 检查全服邮件是否可移除
func (m *UserMailManager) isGlobalMailRecordRemovable(ctx cd.RpcContext, record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}
	return logic_global_mail.GetUserRouterManager(ctx.GetApp()).IsRecordRemoveable(record)
}

// isGlobalMailHistoryRemovable 检查全服邮件是否可历史移除
func (m *UserMailManager) isGlobalMailHistoryRemovable(ctx cd.RpcContext, record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}
	return logic_global_mail.GetUserRouterManager(ctx.GetApp()).IsHistoryRemoveable(record)
}

func (m *UserMailManager) sendMailAttachments(ctx cd.RpcContext, mail *mail_data.MailData, needRemove bool) cd.RpcResult {
	if mail.Record == nil || mail.Content == nil {
		ctx.LogError("SendMailAttachments failed, mail record or content is nil")
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND)

	}

	if len(mail.Content.AttachmentsOffset) == 0 {
		ctx.LogDebug("SendMailAttachments failed, mail has no attachments", "mail_id", mail.Record.GetMailId())
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NO_ATTACHMENTS)

	}

	if (mail.Record.GetStatus() & int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_TOKEN_ATTACHMENT)) != 0 {
		// 附件已经领取完毕
		return cd.CreateRpcResultOk()
	}

	rewardOffsets := make([]*public_protocol_common.DItemOffset, 0)

	for _, attach := range mail.Content.GetAttachmentsOffset() {
		rewardOffsets = append(rewardOffsets, attach.Item)
	}

	rewardItemInsts, ret := m.GetOwner().GenerateMultipleItemInstancesFromOffset(ctx, rewardOffsets, false)
	if ret.IsError() {
		ctx.LogError("generate quest reward items failed",
			"mail_id", mail.Record.GetMailId(),
			"error", ret.GetStandardError(),
			"response_code", ret.GetResponseCode(),
		)
		return ret
	}

	addGuards, ret := m.GetOwner().CheckAddItem(ctx, rewardItemInsts)
	if ret.IsError() {
		ctx.LogError("check add quest reward failed",
			"mail_id", mail.Record.GetMailId(),
			"error", ret.GetStandardError(),
			"response_code", ret.GetResponseCode(),
		)
		return ret
	}

	// 更新邮件记录已领取附件

	mail.Record.Status = mail.Record.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_TOKEN_ATTACHMENT)

	itemFlowReason := &data.ItemFlowReason{
		MajorReason: int32(public_protocol_common.EnItemFlowReasonMajorType_EN_ITEM_FLOW_REASON_MAJOR_MAIL),
		MinorReason: int32(public_protocol_common.EnItemFlowReasonMinorType_EN_ITEM_FLOW_REASON_MINOR_MAIL_ATTACHMENTS),
		Parameter:   int64(mail.Record.GetMailId()),
	}

	ret = m.GetOwner().AddItem(ctx, addGuards, itemFlowReason)
	if !ret.IsOK() {
		ctx.LogError("add mail attachment items failed",
			"mail_id", mail.Record.GetMailId(),
			"error", ret.GetStandardError(),
			"response_code", ret.GetResponseCode(),
		)
		return ret
	}

	return ret
}

func (m *UserMailManager) WaitForAsyncTask(ctx cd.RpcContext) cd.RpcResult {
	if !m.IsAsyncTaskRunning() {
		return cd.CreateRpcResultOk()
	}
	task := m.mailAsyncTask.Load()
	result := cd.AwaitTask(m.GetOwner().GetSession().GetDispatcher().CreateAwaitableContext(), task)
	return result
}
