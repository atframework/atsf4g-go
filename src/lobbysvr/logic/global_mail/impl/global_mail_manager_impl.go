package lobbysvr_logic_global_mail_impl

import (
	"context"
	"math/rand"
	"time"

	"github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/proto"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component/config"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	mail_util "github.com/atframework/atsf4g-go/component/mail"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	uc "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	global_mail_action "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/action"
	global_mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/data"
	logic_mail "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
)

// GlobalMailManager 全局邮件管理器实现
type GlobalMailManager struct {
	libatapp.AppModuleBase

	globalMailSyncTask lu.AtomicInterface[cd.TaskActionImpl]

	// 异步任务相关
	// TODO: mail_async_task_ 异步任务类型需要适配Go的协程模型
	taskNextTimepoint         int64
	lastSuccessFetchTimepoint int64
	forceUsersReload          bool

	// 邮件数据索引
	mailBoxIdIndex          global_mail_data.GlobalMailBox       // 按mail_id索引
	mailBoxTypeIndex        global_mail_data.GlobalMailBoxByType // 按zone_id和major_type索引
	pendingToRemove         global_mail_data.GlobalMailRecordMap // 待移除队列
	mailUnloadedIndex       mail_data.MailIndex                  // 未加载内容的邮件索引
	pendingToRemoveContents map[int64]struct{}                   // 待移除内容的邮件ID集合
}

func init() {
	var _ libatapp.AppModuleImpl = (*GlobalMailManager)(nil)
}

// NewGlobalMailManager 创建全局邮件管理器实例
func NewGlobalMailManager(owner libatapp.AppImpl) *GlobalMailManager {
	now := time.Now().Unix()
	// 启动随机20秒起始拉取时间，以便错峰负载
	randomOffset := rand.Int63n(20)

	ret := &GlobalMailManager{
		AppModuleBase:             libatapp.CreateAppModuleBase(owner),
		taskNextTimepoint:         now + randomOffset,
		lastSuccessFetchTimepoint: 0,
		forceUsersReload:          false,
		mailBoxIdIndex:            make(global_mail_data.GlobalMailBox),
		mailBoxTypeIndex:          make(global_mail_data.GlobalMailBoxByType),
		pendingToRemove:           make(global_mail_data.GlobalMailRecordMap),
		mailUnloadedIndex:         make(mail_data.MailIndex),
		pendingToRemoveContents:   make(map[int64]struct{}),
	}
	return ret
}

func (m *GlobalMailManager) Init(parent context.Context) error {
	return nil
}

func (m *GlobalMailManager) Tick(parent context.Context) bool {

	m.TryToStartAsyncJobs()

	if m.forceUsersReload {
		// 重置所有玩家对象的全服邮件缓存为已过期
		urm := uc.GetUserRouterManager(m.GetApp())
		if urm != nil {
			urm.ForeachObject(func(cache *uc.UserRouterCache) bool {
				userImpl := cache.GetUserImpl()
				if userImpl == nil || !userImpl.IsWriteable() {
					return true
				}
				user, ok := userImpl.(*data.User)
				if !ok || user == nil {
					return true
				}
				mailMgr := data.UserGetModuleManager[logic_mail.UserMailManager](user)
				if mailMgr != nil {
					mailMgr.ResetGlobalMailsCache()
				}
				return true
			})
		}
		m.forceUsersReload = false
	}
	return false
}

func (m *GlobalMailManager) Name() string {
	return "GlobalMailManager"
}

// IsAsyncTaskRunning 检查异步任务是否正在运行
func (m *GlobalMailManager) IsAsyncTaskRunning() bool {
	if lu.IsNil(m.globalMailSyncTask.Load()) {
		return false
	}
	if m.globalMailSyncTask.Load().IsExiting() {
		m.globalMailSyncTask.Store(nil)
		return false
	}
	return true
}

// TryToStartAsyncJobs 尝试启动异步任务
func (m *GlobalMailManager) TryToStartAsyncJobs() {
	d := libatapp.AtappGetModule[*cd.NoMessageDispatcher](m.GetApp())
	ctx := d.CreateRpcContext()
	now := ctx.GetNow().Unix()
	if now < m.taskNextTimepoint {
		return
	}

	if m.IsAsyncTaskRunning() {
		return
	}

	globalMailSyncTask, startData := cd.CreateNoMessageTaskAction(
		d, ctx, nil,
		func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *global_mail_action.TaskActionGlobalMailSyncObjects {
			return global_mail_action.CreateTaskActionGlobalMailSyncObjects(
				cd.CreateNoMessageTaskActionBase(rd, actor, timeout),
				m,
			)
		},
	)

	err := libatapp.AtappGetModule[*cd.TaskManager](ctx.GetApp()).StartTaskAction(ctx, globalMailSyncTask, &startData)
	if err != nil {
		m.GetApp().GetDefaultLogger().LogError("TaskActionAutoSaveObjects StartTaskAction failed", "error", err)
	} else {
		m.globalMailSyncTask.Store(globalMailSyncTask)
	}

	m.GetApp().GetDefaultLogger().LogInfo("GlobalMailManager try to start async jobs")

	// 启动随机一下下一轮时间，以便错峰负载
	interval := global_mail_data.EN_CL_MAIL_GLOBAL_JOBS_TASK_INTERVAL
	randomOffset := rand.Int63n(interval >> 1)
	m.taskNextTimepoint = now + interval - (interval >> 2) + randomOffset
}

// ResetAsyncJobsProtect 重置异步任务保护
func (m *GlobalMailManager) ResetAsyncJobsProtect() {
	m.taskNextTimepoint = 0
}

// UpdateFromDB 从数据库更新全局邮件
func (m *GlobalMailManager) UpdateFromDB(ctx cd.RpcContext, zoneId uint32, majorType int32, blobData *private_protocol_pbdesc.DatabaseGlobalMailBlobData, rewriteDbData bool) bool {
	now := ctx.GetNow().Unix()
	ret := false
	oldMailBox := m.internalGetMailBoxByType(zoneId, majorType)
	invalidIds := make(map[int64]struct{})
	if oldMailBox != nil {
		oldMailBox.RangeUnordered(func(mailId int64, _ *mail_data.MailData) bool {
			invalidIds[mailId] = struct{}{}
			return true
		})
	}

	// 收集需要追加的邮件
	appendMailIndices := make([]int, 0, len(blobData.GetMailRecords()))

	// 先更新，再追加
	for i, record := range blobData.GetMailRecords() {
		if rewriteDbData && mail_util.MailIsHistoryRemovable(now, record) {
			ret = true
		}
		// 更新
		if !m.updateGlobalMailInternal(ctx, zoneId, record) {
			appendMailIndices = append(appendMailIndices, i)
		}
		delete(invalidIds, record.GetMailId())
	}

	// 追加
	for _, i := range appendMailIndices {
		m.addGlobalMailInternal(ctx, zoneId, blobData.GetMailRecords()[i])
	}

	// 不在库里的邮件要全部删除
	for invalidMailId := range invalidIds {
		m.GetApp().GetDefaultLogger().LogInfo("GlobalMailManager", "remove mail record because not found in DB", "major_type", majorType, "mail_id", invalidMailId)
		m.removeGlobalMailInternal(ctx, invalidMailId)
	}

	oldMailBox = m.internalGetMailBoxByType(zoneId, majorType)
	if oldMailBox != nil {
		oldCount := oldMailBox.Len()
		m.compactMails(ctx, oldMailBox)
		if rewriteDbData && oldMailBox.Len() != oldCount {
			ret = true
		}
	}

	// 重写刷回数据库数据
	if ret {
		blobData.MailRecords = nil
		if oldMailBox != nil {
			oldMailBox.RangeUnordered(func(_ int64, mail *mail_data.MailData) bool {
				if mail != nil && mail.Record != nil {
					blobData.MailRecords = append(blobData.MailRecords, mail.Record.Clone())
				}
				return true
			})
		}
	}

	// 处理待移除队列
	checkAdditionalRemoveList := make(map[int64]*public_protocol_pbdesc.DMailRecord)
	for mailId, entry := range m.pendingToRemove {
		if entry.ZoneId == zoneId {
			checkAdditionalRemoveList[mailId] = entry.Record
		}
	}

	for i := 0; i < len(blobData.GetPendingRemoveList()); i++ {
		record := blobData.GetPendingRemoveList()[i]
		if checkRecord, ok := checkAdditionalRemoveList[record.GetMailId()]; ok {
			if mail_util.MailIsHistoryRemovable(now, checkRecord) {
				delete(m.pendingToRemove, record.GetMailId())
				if rewriteDbData {
					m.pendingToRemoveContents[record.GetMailId()] = struct{}{}
				}
			}
			if rewriteDbData {
				proto.Merge(record, checkRecord)
			}
			delete(checkAdditionalRemoveList, record.GetMailId())
		} else if mail_util.MailIsHistoryRemovable(now, record) {
			m.pendingToRemoveContents[record.GetMailId()] = struct{}{}
		} else {
			m.pendingToRemove[record.GetMailId()] = &global_mail_data.GlobalMailRecordEntry{
				ZoneId: zoneId,
				Record: record.Clone(),
			}
		}
	}

	// 过期的移除队列
	if rewriteDbData {
		oldSize := len(blobData.GetPendingRemoveList())
		newPendingRemoveList := make([]*public_protocol_pbdesc.DMailRecord, 0, oldSize)
		for _, record := range blobData.GetPendingRemoveList() {
			if !mail_util.MailIsHistoryRemovable(now, record) {
				newPendingRemoveList = append(newPendingRemoveList, record)
			}
		}
		blobData.PendingRemoveList = newPendingRemoveList
		if oldSize != len(newPendingRemoveList) {
			ret = true
		}
	}

	for mailId, record := range checkAdditionalRemoveList {
		if mail_util.MailIsHistoryRemovable(now, record) {
			delete(m.pendingToRemove, mailId)
			if rewriteDbData {
				m.pendingToRemoveContents[mailId] = struct{}{}
			}
		} else if rewriteDbData {
			blobData.PendingRemoveList = append(blobData.PendingRemoveList, record.Clone())
			ret = true
		}
	}

	return ret
}

// IsHistoryRemoveable 检查邮件是否可从历史记录移除
func (m *GlobalMailManager) IsHistoryRemoveable(record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}

	if !record.GetIsGlobalMail() {
		return true
	}

	if record.GetMailId() == 0 || record.GetMajorType() == 0 {
		return true
	}
	// 如果处于待删除队列或者已存在在邮件列表中，历史记录都不能移除
	if _, ok := m.pendingToRemove[record.GetMailId()]; ok {
		return false
	}

	if _, ok := m.mailBoxIdIndex[record.GetMailId()]; ok {
		return false
	}

	// 如果最后拉取成功时间-发件时间大于特定值，且邮件库中找不到则可以删除
	if m.lastSuccessFetchTimepoint > record.GetDeliveryTime()+global_mail_data.EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT {
		return true
	}

	// 可能由于缓存时间差，延期判定
	return false
}

// IsRecordRemoveable 检查邮件是否可从邮箱移除
func (m *GlobalMailManager) IsRecordRemoveable(ctx cd.RpcContext, record *public_protocol_pbdesc.DMailRecord) bool {
	now := ctx.GetNow().Unix()
	if record == nil {
		return true
	}

	if !record.GetIsGlobalMail() {
		return false
	}

	if record.GetMailId() == 0 || record.GetMajorType() == 0 {
		return true
	}

	// 检查是否已存在在邮件列表中
	if entry, ok := m.mailBoxIdIndex[record.GetMailId()]; ok {
		if entry == nil || entry.MailData == nil || entry.MailData.Record == nil {
			m.GetApp().GetDefaultLogger().LogError("GlobalMailManager", "mail.record should not be null")
			return true
		}
		return mail_util.MailIsHistoryRemovable(now, entry.MailData.Record)
	}

	// 如果处于待删除列表则可以移除
	if _, ok := m.pendingToRemove[record.GetMailId()]; ok {
		return true
	}
	maxTime := record.GetExpiredTime()
	if record.GetRemoveTime() > maxTime {
		maxTime = record.GetRemoveTime()
	}
	// TODO: 从配置获取 time_tolerate，暂时使用默认值300秒
	timeTolerate := global_mail_data.EN_CL_MAIL_GLOBAL_TIME_TOLERATE
	if now > maxTime+timeTolerate {
		return true
	}

	// 如果最后拉取成功时间-发件时间大于特定值，且邮件库中找不到则可以删除
	if m.lastSuccessFetchTimepoint > record.GetDeliveryTime()+global_mail_data.EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT {
		return true
	}

	return false
}

// UpdateGlobalMailRecord 更新全服邮件记录
func (m *GlobalMailManager) UpdateGlobalMailRecord(ctx cd.RpcContext, dst *public_protocol_pbdesc.DMailRecord, src *public_protocol_pbdesc.DMailRecord) {
	now := ctx.GetNow().Unix()
	if dst == nil || src == nil {
		return
	}

	dst.MailId = src.GetMailId()
	dst.MajorType = src.GetMajorType()
	dst.MinorType = src.GetMinorType()
	dst.Status = dst.GetStatus() | src.GetStatus()
	dst.IsGlobalMail = true
	// 结算首次刷新
	if dst.GetDeliveryTime() <= 0 {
		if src.GetResolveExpiredTime() <= 0 {
			dst.DeliveryTime = src.GetDeliveryTime()
			dst.ExpiredTime = src.GetExpiredTime()
			dst.RemoveTime = src.GetRemoveTime()
		} else {
			dst.DeliveryTime = now
			expiredTime := global_mail_data.MailPaddingDayTime(now + src.GetResolveExpiredTime())
			dst.ExpiredTime = expiredTime
			removeTime := src.GetRemoveTime()
			if expiredTime > removeTime {
				removeTime = expiredTime
			}
			dst.RemoveTime = removeTime
		}
	}
	dst.StartTime = src.GetStartTime()
	dst.ShowTime = src.GetShowTime()
	dst.AfterReadExpiredTime = src.GetAfterReadExpiredTime()

	// 重设过期时间的配置
	dst.ResolveExpiredTime = src.GetResolveExpiredTime()
}

// AddGlobalMail 添加全服邮件
func (m *GlobalMailManager) AddGlobalMail(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) int32 {
	return m.addGlobalMailInternal(ctx, zoneId, mail)
}

// addGlobalMailInternal 添加全服邮件
func (m *GlobalMailManager) addGlobalMailInternal(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) int32 {
	now := ctx.GetNow().Unix()
	if mail.GetMailId() == 0 || mail.GetMajorType() == 0 {
		m.GetApp().GetDefaultLogger().LogError("GlobalMailManager", "cannot add mail, mail_id=0 or major_type=0")
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 如果在移除队列中则忽略
	if _, ok := m.pendingToRemove[mail.GetMailId()]; ok {
		return 0
	}

	if entry, ok := m.mailBoxIdIndex[mail.GetMailId()]; ok {
		oldIsFuture := entry.MailData.Record.GetStartTime() > now
		newIsFuture := mail.GetStartTime() > now

		// 已存在直接刷新
		if entry.MailData != nil && entry.MailData.Record != nil {
			if entry.MailData.Record.GetStatus() != mail.GetStatus() {
				m.forceUsersReload = true
			}

			// 保留失败状态
			keepFetchErrorCount := entry.MailData.Record.GetFetchErrorCount()
			entry.MailData.Record = mail.Clone()
			// proto.Merge(entry.MailData.Record, mail)
			entry.MailData.Record.FetchErrorCount = keepFetchErrorCount
		}

		// 如果添加过期全服邮件则直接进删除队列
		if mail_util.MailIsHistoryRemovable(now, mail) {
			m.removeGlobalMailInternal(ctx, mail.GetMailId())
		} else if oldIsFuture != newIsFuture {
			if zoneBox, ok := m.mailBoxTypeIndex[zoneId]; ok {
				if box, ok := zoneBox[mail.GetMajorType()]; ok {
					box.FutureCacheExpire = 0
				}
			}
		}

		return 0
	}

	// 创建新邮件
	mailData := &mail_data.MailData{
		Record: mail.Clone(),
	}

	m.mailBoxIdIndex[mail.GetMailId()] = &global_mail_data.GlobalMailBoxEntry{
		ZoneId:   zoneId,
		MailData: mailData,
	}

	if mailData.Content == nil {
		m.mailUnloadedIndex[mail.GetMailId()] = mailData
	}

	// 确保类型索引存在
	if _, ok := m.mailBoxTypeIndex[zoneId]; !ok {
		m.mailBoxTypeIndex[zoneId] = make(map[int32]*mail_data.MailBox)
	}
	if _, ok := m.mailBoxTypeIndex[zoneId][mail.GetMajorType()]; !ok {
		m.mailBoxTypeIndex[zoneId][mail.GetMajorType()] = mail_data.NewMailBoxWithSort(mail_data.MailDataLessByMailIdDesc)
	}
	mailBoxTypeIndex := m.mailBoxTypeIndex[zoneId][mail.GetMajorType()]
	mailBoxTypeIndex.InsertSorted(mail.GetMailId(), mailData)

	m.GetApp().GetDefaultLogger().LogInfo("GlobalMailManager", "add mail success", "major_type", mail.GetMajorType(), "mail_id", mail.GetMailId())

	if mailData.Record.GetStartTime() > now {
		mailBoxTypeIndex.FutureMailCount++
		if mailData.Record.GetStartTime() < mailBoxTypeIndex.FutureCacheExpire {
			mailBoxTypeIndex.FutureCacheExpire = mailData.Record.GetStartTime()
		}
	}

	// 如果本来就是已过期则加入到移除列表
	if mail_util.MailIsHistoryRemovable(now, mail) {
		m.removeGlobalMailInternal(ctx, mail.GetMailId())
	}

	global_mail_data.RefreshGlobalMailBoxFutureCache(mailBoxTypeIndex)

	futureReserveCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetGlobalMailFutureReserveMaxCountPerMajorType()
	if futureReserveCount > mailBoxTypeIndex.FutureMailCount {
		futureReserveCount = mailBoxTypeIndex.FutureMailCount
	}
	if futureReserveCount < 0 {
		futureReserveCount = 0
	}

	maxMailCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetGlobalMailMaxCountPerMajorType()
	if maxMailCount > 0 && mailBoxTypeIndex.Len() > int(maxMailCount)+int(futureReserveCount) {
		m.compactMails(ctx, mailBoxTypeIndex)
	}

	return 0
}

// UpdateGlobalMail 更新全服邮件
func (m *GlobalMailManager) UpdateGlobalMail(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) bool {
	return m.updateGlobalMailInternal(ctx, zoneId, mail)
}

// updateGlobalMailInternal 更新全服邮件
func (m *GlobalMailManager) updateGlobalMailInternal(ctx cd.RpcContext, zoneId uint32, mail *public_protocol_pbdesc.DMailRecord) bool {
	now := ctx.GetNow().Unix()
	if mail.GetMailId() == 0 || mail.GetMajorType() == 0 {
		m.GetApp().GetDefaultLogger().LogError("GlobalMailManager", "cannot update mail", "mail_id", mail.GetMailId(), "major_type", mail.GetMajorType())
		return false
	}

	// 如果在移除队列中则忽略
	if _, ok := m.pendingToRemove[mail.GetMailId()]; ok {
		return false
	}

	entry, ok := m.mailBoxIdIndex[mail.GetMailId()]
	if !ok {
		return false
	}

	oldIsFuture := entry.MailData.Record.GetStartTime() > now
	newIsFuture := mail.GetStartTime() > now

	// 直接刷新
	if entry.MailData != nil && entry.MailData.Record != nil {
		if entry.MailData.Record.GetStatus() != mail.GetStatus() {
			m.forceUsersReload = true
		}

		// 保留失败状态
		keepFetchErrorCount := entry.MailData.Record.GetFetchErrorCount()
		proto.Merge(entry.MailData.Record, mail)
		entry.MailData.Record.FetchErrorCount = keepFetchErrorCount
	}

	// 如果过期全服邮件则直接进删除队列
	if mail_util.MailIsHistoryRemovable(now, mail) {
		m.removeGlobalMailInternal(ctx, mail.GetMailId())
	} else if oldIsFuture != newIsFuture {
		if zoneBox, ok := m.mailBoxTypeIndex[zoneId]; ok {
			if box, ok := zoneBox[mail.GetMajorType()]; ok {
				box.FutureCacheExpire = 0
			}
		}
	}

	return true
}

// GetMailRaw 获取邮件原始数据
func (m *GlobalMailManager) GetMailRaw(mailId int64) *mail_data.MailData {
	if entry, ok := m.mailBoxIdIndex[mailId]; ok {
		return entry.MailData
	}
	return nil
}

// GetMailBoxByType 按类型获取邮件箱
func (m *GlobalMailManager) GetMailBoxByType(zoneId uint32, majorType int32) *mail_data.MailBox {
	return m.internalGetMailBoxByType(zoneId, majorType)
}

// internalGetMailBoxByType 按类型获取邮件箱
func (m *GlobalMailManager) internalGetMailBoxByType(zoneId uint32, majorType int32) *mail_data.MailBox {
	if zoneBox, ok := m.mailBoxTypeIndex[zoneId]; ok {
		if box, ok := zoneBox[majorType]; ok {
			return box
		}
	}
	return nil
}

// GetAllGlobalMails 获取所有全局邮件
func (m *GlobalMailManager) GetAllGlobalMails() global_mail_data.GlobalMailBox {
	return m.mailBoxIdIndex
}

// RemoveGlobalMail 移除全服邮件（公开方法，带锁）
func (m *GlobalMailManager) RemoveGlobalMail(ctx cd.RpcContext, mailId int64) {
	m.removeGlobalMailInternal(ctx, mailId)
}

// removeGlobalMailInternal 移除全服邮件（内部方法，需要持有锁）
func (m *GlobalMailManager) removeGlobalMailInternal(ctx cd.RpcContext, mailId int64) {
	now := ctx.GetNow().Unix()
	entry, ok := m.mailBoxIdIndex[mailId]
	if !ok {
		return
	}

	zoneId := entry.ZoneId
	if entry.MailData == nil || entry.MailData.Record == nil {
		m.GetApp().GetDefaultLogger().LogError("GlobalMailManager", "mail.record should not be null")
		delete(m.mailBoxIdIndex, mailId)
		delete(m.mailUnloadedIndex, mailId)
		return
	}

	m.forceUsersReload = true

	// 从类型索引中移除
	if zoneBox, ok := m.mailBoxTypeIndex[zoneId]; ok {
		majorType := entry.MailData.Record.GetMajorType()
		if box, ok := zoneBox[majorType]; ok {
			// 重置未来邮件缓存
			if box.FutureCacheExpire <= entry.MailData.Record.GetStartTime() {
				box.FutureCacheExpire = 0
			}

			box.Remove(mailId)
			if box.Len() == 0 {
				delete(zoneBox, majorType)
			}
		}

		if len(zoneBox) == 0 {
			delete(m.mailBoxTypeIndex, zoneId)
		}
	}

	entry.MailData.Record.Status = entry.MailData.Record.GetStatus() | int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED)

	time_tolerate := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetMailTimeTolerate().GetSeconds()
	// 重设失效时间为当前时间+容忍误差时间
	removeTime := now + time_tolerate
	if entry.MailData.Record.GetExpiredTime() > removeTime {
		entry.MailData.Record.ExpiredTime = removeTime
	}
	if entry.MailData.Record.GetRemoveTime() > removeTime {
		entry.MailData.Record.RemoveTime = removeTime
	}

	m.GetApp().GetDefaultLogger().LogInfo("GlobalMailManager", "remove mail and reset expired time success",
		"major_type", entry.MailData.Record.GetMajorType(),
		"mail_id", mailId,
		"expired_time", entry.MailData.Record.GetExpiredTime())

	// 加入待移除队列
	m.pendingToRemove[mailId] = &global_mail_data.GlobalMailRecordEntry{
		ZoneId: zoneId,
		Record: entry.MailData.Record.Clone(),
	}

	mailToRemove := m.pendingToRemove[mailId]
	if mailToRemove.Record.GetRemoveTime() > 0 {
		mailToRemove.Record.RemoveTime += time_tolerate
	}

	if mailToRemove.Record.GetExpiredTime() > 0 {
		mailToRemove.Record.ExpiredTime += time_tolerate
	}

	// TODO:发送 OSS 日志
	delete(m.mailUnloadedIndex, mailId)
	delete(m.mailBoxIdIndex, mailId)
}

// compactMails 压缩邮件箱（移除过期邮件）
func (m *GlobalMailManager) compactMails(ctx cd.RpcContext, mailBox *mail_data.MailBox) {
	now := ctx.GetNow().Unix()
	global_mail_data.RefreshGlobalMailBoxFutureCache(mailBox)

	maxMailCountConfig := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetGlobalMailMaxCountPerMajorType()
	if mailBox.Len() == 0 || maxMailCountConfig <= 0 {
		return
	}

	futureReserveCount := config.GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetGlobalMailFutureReserveMaxCountPerMajorType()
	if futureReserveCount > mailBox.FutureMailCount {
		futureReserveCount = mailBox.FutureMailCount
	}
	if futureReserveCount < 0 {
		futureReserveCount = 0
	}

	maxMailCount := int(maxMailCountConfig) + int(futureReserveCount)
	if mailBox.Len() <= maxMailCount {
		return
	}

	deliveryTimeMaxOffset := global_mail_data.DEFAULT_COMPACT_DELIVERY_TIME_MAX_OFFSET

	pendingToRemove := make([]int64, 0, mailBox.Len()-maxMailCount)
	rmcnt := mailBox.Len() - maxMailCount

	m.GetApp().GetDefaultLogger().LogInfo("GlobalMailManager", "start to compact mails",
		"rmcnt", rmcnt,
		"sum_mail_count", mailBox.Len(),
		"future_mail_count", mailBox.FutureMailCount,
		"limit", maxMailCountConfig,
		"future_limit", futureReserveCount)

	for i := 0; i < rmcnt && mailBox.Len() > maxMailCount; i++ {
		var selectMailId int64 = 0
		var selectMailCompactTime int64 = 0

		mailBox.RangeUnordered(func(mailId int64, mail *mail_data.MailData) bool {
			if mail == nil || mail.Record == nil {
				pendingToRemove = append(pendingToRemove, mailId)
				return true
			}

			maxExpireRemove := mail.Record.GetExpiredTime()
			if mail.Record.GetRemoveTime() > maxExpireRemove {
				maxExpireRemove = mail.Record.GetRemoveTime()
			}
			if maxExpireRemove < now {
				pendingToRemove = append(pendingToRemove, mailId)
				return true
			}

			compactTime := mail.Record.GetStartTime()
			if mail.Record.GetDeliveryTime() > compactTime {
				compactTime = mail.Record.GetDeliveryTime()
			} else if compactTime > mail.Record.GetDeliveryTime()+deliveryTimeMaxOffset {
				compactTime = mail.Record.GetDeliveryTime() + deliveryTimeMaxOffset
			}

			// 相同时间选mail_id小的
			if selectMailId == 0 || compactTime < selectMailCompactTime ||
				(compactTime == selectMailCompactTime && selectMailId > mailId) {
				selectMailCompactTime = compactTime
				selectMailId = mailId
			}
			return true
		})

		if selectMailId != 0 && mailBox.Len() > maxMailCount+len(pendingToRemove) {
			pendingToRemove = append(pendingToRemove, selectMailId)
		}

		for _, mailId := range pendingToRemove {
			m.removeGlobalMailInternal(ctx, mailId)
		}
		pendingToRemove = pendingToRemove[:0]
	}
}

// FetchAllUnloadedMails 获取所有未加载内容的邮件ID
func (m *GlobalMailManager) FetchAllUnloadedMails(ctx cd.RpcContext) []int64 {
	now := ctx.GetNow().Unix()
	invalidMailIds := make([]int64, 0)
	out := make([]int64, 0, len(m.mailUnloadedIndex))

	for mailId, checked := range m.mailUnloadedIndex {
		if checked != nil && checked.Record != nil && checked.Content == nil {
			if m.IsRecordRemoveableInternal(now, checked.Record) {
				invalidMailIds = append(invalidMailIds, mailId)
				m.removeGlobalMailInternal(ctx, checked.Record.GetMailId())
			} else {
				out = append(out, mailId)
			}
		} else {
			invalidMailIds = append(invalidMailIds, mailId)
		}
	}

	for _, invalidMailId := range invalidMailIds {
		delete(m.mailUnloadedIndex, invalidMailId)
	}

	return out
}

// IsRecordRemoveableInternal 检查邮件是否可从邮箱移除（内部方法，不加锁）
func (m *GlobalMailManager) IsRecordRemoveableInternal(now int64, record *public_protocol_pbdesc.DMailRecord) bool {
	if record == nil {
		return true
	}

	if !record.GetIsGlobalMail() {
		return false
	}

	if record.GetMailId() == 0 || record.GetMajorType() == 0 {
		return true
	}

	// 检查是否已存在在邮件列表中
	if entry, ok := m.mailBoxIdIndex[record.GetMailId()]; ok {
		if entry == nil || entry.MailData == nil || entry.MailData.Record == nil {
			return true
		}
		return mail_util.MailIsHistoryRemovable(now, entry.MailData.Record)
	}

	// 如果处于待删除列表则可以移除
	if _, ok := m.pendingToRemove[record.GetMailId()]; ok {
		return true
	}

	maxTime := record.GetExpiredTime()
	if record.GetRemoveTime() > maxTime {
		maxTime = record.GetRemoveTime()
	}
	timeTolerate := int64(300)
	if now > maxTime+timeTolerate {
		return true
	}

	if m.lastSuccessFetchTimepoint > record.GetDeliveryTime()+global_mail_data.EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT {
		return true
	}

	return false
}

// SetMailContentLoaded 设置邮件内容已加载
func (m *GlobalMailManager) SetMailContentLoaded(mailId int64) {
	if _, ok := m.mailUnloadedIndex[mailId]; ok {
		delete(m.mailUnloadedIndex, mailId)
		m.forceUsersReload = true
	}
}

// SetLastSuccessFetchTimepoint 设置最后成功拉取时间点
func (m *GlobalMailManager) SetLastSuccessFetchTimepoint(t int64) {
	m.lastSuccessFetchTimepoint = t
}

// ClearPendingToRemoveContents 清除待移除内容的邮件ID集合
func (m *GlobalMailManager) ClearPendingToRemoveContents() {
	m.pendingToRemoveContents = make(map[int64]struct{})
}

// GetPendingToRemoveContentsList 获取待移除内容的邮件ID列表（返回副本）
func (m *GlobalMailManager) GetPendingToRemoveContentsList() []int64 {
	result := make([]int64, 0, len(m.pendingToRemoveContents))
	for mailId := range m.pendingToRemoveContents {
		result = append(result, mailId)
	}
	return result
}

// RemovePendingToRemoveContent 从待移除内容列表中移除指定邮件ID
func (m *GlobalMailManager) RemovePendingToRemoveContent(mailId int64) {
	delete(m.pendingToRemoveContents, mailId)
}

// AddPendingToRemoveContent 添加邮件ID到待移除内容列表
func (m *GlobalMailManager) AddPendingToRemoveContent(mailId int64) {
	m.pendingToRemoveContents[mailId] = struct{}{}
}
