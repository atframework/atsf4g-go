package lobbysvr_logic_global_mail_impl

import (
	"testing"
	"time"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	global_mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/data"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
	"github.com/atframework/libatapp-go"
	"github.com/stretchr/testify/assert"
)

// ==================== 辅助函数 ====================

// newTestGlobalMailManager 创建测试用的 GlobalMailManager 实例
func newTestGlobalMailManager() *GlobalMailManager {
	app := libatapp.CreateAppInstance()
	return &GlobalMailManager{
		AppModuleBase:             libatapp.CreateAppModuleBase(app),
		taskNextTimepoint:         0,
		lastSuccessFetchTimepoint: 0,
		forceUsersReload:          false,
		mailBoxIdIndex:            make(global_mail_data.GlobalMailBox),
		mailBoxTypeIndex:          make(global_mail_data.GlobalMailBoxByType),
		pendingToRemove:           make(global_mail_data.GlobalMailRecordMap),
		mailUnloadedIndex:         make(mail_data.MailIndex),
		pendingToRemoveContents:   make(map[int64]struct{}),
	}
}

// newTestGlobalMailRecord 创建测试用的全服邮件记录
func newTestGlobalMailRecord(mailId int64, majorType int32) *public_protocol_pbdesc.DMailRecord {
	now := time.Now().Unix()
	return &public_protocol_pbdesc.DMailRecord{
		
		MailId:       mailId,
		MajorType:    majorType,
		MinorType:    1,
		Status:       int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_NONE),
		DeliveryTime: now,
		StartTime:    now - 60,
		ExpiredTime:  now + 86400,
		RemoveTime:   now + 86400*2,
		IsGlobalMail: true,
	}
}

// newExpiredGlobalMailRecord 创建已过期的全服邮件记录
func newExpiredGlobalMailRecord(mailId int64, majorType int32) *public_protocol_pbdesc.DMailRecord {
	now := time.Now().Unix()
	return &public_protocol_pbdesc.DMailRecord{
		MailId:       mailId,
		MajorType:    majorType,
		MinorType:    1,
		Status:       int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_NONE),
		DeliveryTime: now - 172800,
		StartTime:    now - 172800,
		ExpiredTime:  now - 3600, // 已过期
		RemoveTime:   now - 1800, // 已过 remove time
		IsGlobalMail: true,
	}
}

// newFutureGlobalMailRecord 创建未来生效的全服邮件记录
func newFutureGlobalMailRecord(mailId int64, majorType int32, startTime int64) *public_protocol_pbdesc.DMailRecord {
	now := time.Now().Unix()
	return &public_protocol_pbdesc.DMailRecord{
		MailId:       mailId,
		MajorType:    majorType,
		MinorType:    1,
		Status:       int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_NONE),
		DeliveryTime: now,
		StartTime:    startTime,
		ExpiredTime:  startTime + 86400,
		RemoveTime:   startTime + 86400*2,
		IsGlobalMail: true,
	}
}

// ==================== AddGlobalMail 测试 ====================

// TestAddGlobalMailSuccess 测试成功添加全服邮件
// Scenario: 添加有效全服邮件，应成功添加到索引和类型索引中
func TestAddGlobalMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(0), ret, "should return 0 for success")

	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail, "mail should exist in index")
	assert.Equal(t, int64(1001), mail.Record.GetMailId())

	mailBox := mgr.GetMailBoxByType(1, 1)
	assert.NotNil(t, mailBox, "mailbox should exist for zone_id=1, major_type=1")
	assert.Equal(t, 1, len(mailBox.Mails), "mailbox should have 1 mail")
}

// TestAddGlobalMailInvalidParamZeroMailId 测试 mail_id=0 时添加失败
// Scenario: mail_id=0 的邮件不允许添加
func TestAddGlobalMailInvalidParamZeroMailId(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(0, 1)

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), ret,
		"should return EN_ERR_INVALID_PARAM when mail_id is 0")
}

// TestAddGlobalMailInvalidParamZeroMajorType 测试 major_type=0 时添加失败
// Scenario: major_type=0 的邮件不允许添加
func TestAddGlobalMailInvalidParamZeroMajorType(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 0)

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), ret,
		"should return EN_ERR_INVALID_PARAM when major_type is 0")
}

// TestAddGlobalMailDuplicate 测试重复添加全服邮件
// Scenario: 同一 mail_id 重复添加应更新已有记录
func TestAddGlobalMailDuplicate(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record1 := newTestGlobalMailRecord(1001, 1)
	record2 := newTestGlobalMailRecord(1001, 1)
	record2.MinorType = 2

	// Act
	ret1 := mgr.AddGlobalMail(1, record1)
	ret2 := mgr.AddGlobalMail(1, record2)

	// Assert
	assert.Equal(t, int32(0), ret1, "first add should succeed")
	assert.Equal(t, int32(0), ret2, "duplicate add should succeed (update)")

	allMails := mgr.GetAllGlobalMails()
	assert.Equal(t, 1, len(allMails), "should still have 1 mail after duplicate add")
}

// TestAddGlobalMailInPendingRemove 测试在移除队列中的邮件不会重新添加
// Scenario: mail_id 已在 pendingToRemove 中应跳过
func TestAddGlobalMailInPendingRemove(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.pendingToRemove[1001] = &global_mail_data.GlobalMailRecordEntry{
		ZoneId: 1,
		Record: record.Clone(),
	}

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(0), ret, "should return 0 for ignored (in pending remove)")
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should not be in mailbox")
}

// TestAddGlobalMailExpired 测试添加已过期的全服邮件
// Scenario: 过期邮件添加后应自动进入删除队列
func TestAddGlobalMailExpired(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newExpiredGlobalMailRecord(1001, 1)

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(0), ret, "expired mail should return 0")
	assert.Nil(t, mgr.GetMailRaw(1001), "expired mail should be removed from index")
	assert.Contains(t, mgr.pendingToRemove, int64(1001), "expired mail should be in pending remove")
}

// TestAddGlobalMailFuture 测试添加未来生效的全服邮件
// Scenario: 未来生效的邮件应成功添加并增加 FutureMailCount
func TestAddGlobalMailFuture(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	futureStart := time.Now().Unix() + 86400
	record := newFutureGlobalMailRecord(1001, 1, futureStart)

	// Act
	ret := mgr.AddGlobalMail(1, record)

	// Assert
	assert.Equal(t, int32(0), ret, "future mail should be added successfully")
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail, "future mail should exist")

	mailBox := mgr.GetMailBoxByType(1, 1)
	assert.NotNil(t, mailBox)
	assert.Equal(t, int32(1), mailBox.FutureMailCount, "should have 1 future mail")
}

// TestAddGlobalMailMultipleZones 测试在多个 zone 中添加邮件
// Scenario: 不同 zone 的邮件应独立管理
func TestAddGlobalMailMultipleZones(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record1 := newTestGlobalMailRecord(1001, 1)
	record2 := newTestGlobalMailRecord(1002, 1)

	// Act
	mgr.AddGlobalMail(1, record1) // zone 1
	mgr.AddGlobalMail(2, record2) // zone 2

	// Assert
	assert.NotNil(t, mgr.GetMailBoxByType(1, 1), "zone 1 mailbox should exist")
	assert.NotNil(t, mgr.GetMailBoxByType(2, 1), "zone 2 mailbox should exist")
	assert.Equal(t, 1, len(mgr.GetMailBoxByType(1, 1).Mails))
	assert.Equal(t, 1, len(mgr.GetMailBoxByType(2, 1).Mails))
}

// TestAddGlobalMailWithContentNotLoaded 测试添加无内容的全服邮件应加入 unloaded 索引
// Scenario: Content 为 nil 的邮件应出现在 mailUnloadedIndex 中
func TestAddGlobalMailWithContentNotLoaded(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)

	// Act
	mgr.AddGlobalMail(1, record)

	// Assert
	_, exists := mgr.mailUnloadedIndex[1001]
	assert.True(t, exists, "mail without content should be in unloaded index")
}

// ==================== UpdateGlobalMail 测试 ====================

// TestUpdateGlobalMailSuccess 测试成功更新全服邮件
// Scenario: 更新已存在的全服邮件应返回 true
func TestUpdateGlobalMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	updateRecord := newTestGlobalMailRecord(1001, 1)
	updateRecord.MinorType = 99

	// Act
	ok := mgr.UpdateGlobalMail(1, updateRecord)

	// Assert
	assert.True(t, ok, "update should succeed for existing mail")
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail)
}

// TestUpdateGlobalMailNotFound 测试更新不存在的全服邮件
// Scenario: 更新不存在的邮件应返回 false
func TestUpdateGlobalMailNotFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	record := newTestGlobalMailRecord(9999, 1)

	// Act
	ok := mgr.UpdateGlobalMail(1, record)

	// Assert
	assert.False(t, ok, "update should return false for non-existent mail")
}

// TestUpdateGlobalMailInvalidParam 测试更新参数无效的全服邮件
// Scenario: mail_id=0 或 major_type=0 时应返回 false
func TestUpdateGlobalMailInvalidParam(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	record1 := newTestGlobalMailRecord(0, 1) // mail_id=0
	record2 := newTestGlobalMailRecord(1, 0) // major_type=0

	// Act & Assert
	assert.False(t, mgr.UpdateGlobalMail(1, record1), "should fail for mail_id=0")
	assert.False(t, mgr.UpdateGlobalMail(1, record2), "should fail for major_type=0")
}

// TestUpdateGlobalMailInPendingRemove 测试更新在移除队列中的邮件
// Scenario: 在 pendingToRemove 中的邮件更新应返回 false
func TestUpdateGlobalMailInPendingRemove(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// 移除邮件使其进入 pendingToRemove
	mgr.RemoveGlobalMail(1001)

	updateRecord := newTestGlobalMailRecord(1001, 1)

	// Act
	ok := mgr.UpdateGlobalMail(1, updateRecord)

	// Assert
	assert.False(t, ok, "should not update mail in pending remove list")
}

// TestUpdateGlobalMailExpiredTriggersRemove 测试更新为过期邮件会触发移除
// Scenario: 将现有邮件更新为过期状态应触发移除
func TestUpdateGlobalMailExpiredTriggersRemove(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// 更新为过期
	expiredRecord := newExpiredGlobalMailRecord(1001, 1)

	// Act
	ok := mgr.UpdateGlobalMail(1, expiredRecord)

	// Assert
	assert.True(t, ok, "update should return true")
	assert.Nil(t, mgr.GetMailRaw(1001), "expired mail should be removed")
	assert.Contains(t, mgr.pendingToRemove, int64(1001), "should be in pending remove list")
}

// TestUpdateGlobalMailStatusChange 测试更新全服邮件状态变化触发用户重载
// Scenario: 邮件状态变化时应设置 forceUsersReload=true
func TestUpdateGlobalMailStatusChange(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	record.Status = 0
	mgr.AddGlobalMail(1, record)
	mgr.forceUsersReload = false

	updateRecord := newTestGlobalMailRecord(1001, 1)
	updateRecord.Status = int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ)

	// Act
	mgr.UpdateGlobalMail(1, updateRecord)

	// Assert
	assert.True(t, mgr.forceUsersReload, "should force users reload when status changes")
}

// ==================== RemoveGlobalMail 测试 ====================

// TestRemoveGlobalMailSuccess 测试成功移除全服邮件
// Scenario: 移除已存在的邮件应从所有索引中清除并加入待移除队列
func TestRemoveGlobalMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// Act
	mgr.RemoveGlobalMail(1001)

	// Assert
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should not exist in index after removal")
	assert.Contains(t, mgr.pendingToRemove, int64(1001), "should be in pending remove list")
	assert.True(t, mgr.forceUsersReload, "should force users reload after removal")
}

// TestRemoveGlobalMailNotFound 测试移除不存在的全服邮件
// Scenario: 移除不存在的邮件应静默忽略
func TestRemoveGlobalMailNotFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act - 不应 panic
	mgr.RemoveGlobalMail(9999)

	// Assert
	assert.Equal(t, 0, len(mgr.pendingToRemove), "pending remove should be empty")
}

// TestRemoveGlobalMailStatusAndPendingQueue 测试移除后的邮件状态和待移除队列
// Scenario: 移除后邮件记录应带 REMOVED 状态，并出现在 pendingToRemove 中
func TestRemoveGlobalMailStatusAndPendingQueue(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// Act
	mgr.RemoveGlobalMail(1001)

	// Assert
	pendingEntry := mgr.pendingToRemove[1001]
	assert.NotNil(t, pendingEntry, "should have pending remove entry")
	assert.NotNil(t, pendingEntry.Record)
	assert.NotEqual(t, 0,
		pendingEntry.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED),
		"pending remove record should have REMOVED status")
	assert.Equal(t, uint32(1), pendingEntry.ZoneId, "zone id should match")
}

// TestRemoveGlobalMailCleansTypeIndex 测试移除后类型索引被清理
// Scenario: 移除邮件后类型索引中对应的 mailbox 应为空或被删除
func TestRemoveGlobalMailCleansTypeIndex(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// 验证前置
	assert.NotNil(t, mgr.GetMailBoxByType(1, 1))

	// Act
	mgr.RemoveGlobalMail(1001)

	// Assert - 删除唯一的邮件后 mailbox 应被清理
	assert.Nil(t, mgr.GetMailBoxByType(1, 1), "mailbox should be cleaned up after removing last mail")
}

// TestRemoveGlobalMailMultiple 测试连续移除多封邮件
// Scenario: 移除多封邮件后状态应正确
func TestRemoveGlobalMailMultiple(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	for i := int64(1); i <= 5; i++ {
		record := newTestGlobalMailRecord(i, 1)
		mgr.AddGlobalMail(1, record)
	}
	assert.Equal(t, 5, len(mgr.GetAllGlobalMails()))

	// Act
	mgr.RemoveGlobalMail(1)
	mgr.RemoveGlobalMail(3)
	mgr.RemoveGlobalMail(5)

	// Assert
	assert.Equal(t, 2, len(mgr.GetAllGlobalMails()), "should have 2 remaining mails")
	assert.NotNil(t, mgr.GetMailRaw(2), "mail 2 should exist")
	assert.NotNil(t, mgr.GetMailRaw(4), "mail 4 should exist")
	assert.Nil(t, mgr.GetMailRaw(1), "mail 1 should be removed")
	assert.Nil(t, mgr.GetMailRaw(3), "mail 3 should be removed")
	assert.Nil(t, mgr.GetMailRaw(5), "mail 5 should be removed")
	assert.Equal(t, 3, len(mgr.pendingToRemove), "should have 3 pending removes")
}

// ==================== GetMailRaw / GetMailBoxByType / GetAllGlobalMails 测试 ====================

// TestGetMailRawFound 测试获取存在的邮件
// Scenario: 已添加的邮件应可通过 GetMailRaw 获取
func TestGetMailRawFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// Act
	mail := mgr.GetMailRaw(1001)

	// Assert
	assert.NotNil(t, mail, "should return mail data")
	assert.Equal(t, int64(1001), mail.Record.GetMailId())
}

// TestGetMailRawNotFound 测试获取不存在的邮件
// Scenario: 不存在的邮件应返回 nil
func TestGetMailRawNotFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act
	mail := mgr.GetMailRaw(9999)

	// Assert
	assert.Nil(t, mail)
}

// TestGetMailBoxByTypeFound 测试获取存在的类型邮箱
// Scenario: 已有邮件的类型邮箱应能获取到
func TestGetMailBoxByTypeFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 5)
	mgr.AddGlobalMail(2, record)

	// Act
	mailBox := mgr.GetMailBoxByType(2, 5)

	// Assert
	assert.NotNil(t, mailBox, "should return mailbox")
	assert.Equal(t, 1, len(mailBox.Mails))
}

// TestGetMailBoxByTypeNotFound 测试获取不存在的类型邮箱
// Scenario: 不存在的类型邮箱应返回 nil
func TestGetMailBoxByTypeNotFound(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act & Assert
	assert.Nil(t, mgr.GetMailBoxByType(1, 99), "non-existent zone/type should return nil")
	assert.Nil(t, mgr.GetMailBoxByType(99, 1), "non-existent zone should return nil")
}

// TestGetAllGlobalMails 测试获取所有全服邮件
// Scenario: 应返回当前所有邮件的完整索引
func TestGetAllGlobalMails(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	for i := int64(1); i <= 3; i++ {
		record := newTestGlobalMailRecord(i, 1)
		mgr.AddGlobalMail(1, record)
	}

	// Act
	all := mgr.GetAllGlobalMails()

	// Assert
	assert.Equal(t, 3, len(all), "should return all 3 mails")
}

// ==================== UpdateGlobalMailRecord 测试 ====================

// TestUpdateGlobalMailRecordSuccess 测试更新全服邮件记录字段
// Scenario: 正常更新记录字段
func TestUpdateGlobalMailRecordSuccess(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	dst := &public_protocol_pbdesc.DMailRecord{
		MailId:    1001,
		MajorType: 1,
	}
	src := &public_protocol_pbdesc.DMailRecord{
		MailId:       1001,
		MajorType:    2,
		MinorType:    3,
		Status:       int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
		StartTime:    100,
		ShowTime:     200,
		DeliveryTime: 300,
		ExpiredTime:  400,
		RemoveTime:   500,
	}

	// Act
	mgr.UpdateGlobalMailRecord(dst, src)

	// Assert
	assert.Equal(t, int64(1001), dst.GetMailId())
	assert.Equal(t, int32(2), dst.GetMajorType())
	assert.Equal(t, int32(3), dst.GetMinorType())
	assert.True(t, dst.GetIsGlobalMail(), "should set IsGlobalMail to true")
	assert.Equal(t, int64(100), dst.GetStartTime())
	assert.Equal(t, int64(200), dst.GetShowTime())
	// Status 是 OR 合并
	assert.NotEqual(t, 0, dst.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ))
}

// TestUpdateGlobalMailRecordNilParams 测试 nil 参数不应 panic
// Scenario: dst 或 src 为 nil 时应静默返回
func TestUpdateGlobalMailRecordNilParams(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act & Assert - 不应 panic
	mgr.UpdateGlobalMailRecord(nil, &public_protocol_pbdesc.DMailRecord{})
	mgr.UpdateGlobalMailRecord(&public_protocol_pbdesc.DMailRecord{}, nil)
	mgr.UpdateGlobalMailRecord(nil, nil)
}

// TestUpdateGlobalMailRecordStatusMerge 测试状态字段 OR 合并
// Scenario: dst 和 src 各有不同的状态，合并后应都包含
func TestUpdateGlobalMailRecordStatusMerge(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	dst := &public_protocol_pbdesc.DMailRecord{
		Status: int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
	}
	src := &public_protocol_pbdesc.DMailRecord{
		MailId:    1,
		MajorType: 1,
		Status:    int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_TOKEN_ATTACHMENT),
	}

	// Act
	mgr.UpdateGlobalMailRecord(dst, src)

	// Assert
	assert.NotEqual(t, 0, dst.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
		"should keep READ status")
	assert.NotEqual(t, 0, dst.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_TOKEN_ATTACHMENT),
		"should add TOKEN_ATTACHMENT status")
}

// ==================== IsHistoryRemoveable 测试 ====================

// TestIsHistoryRemoveableNil 测试 nil 记录应可移除
// Scenario: nil 记录视为可从历史移除
func TestIsHistoryRemoveableNil(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act & Assert
	assert.True(t, mgr.IsHistoryRemoveable(nil), "nil record should be history removeable")
}

// TestIsHistoryRemoveableNonGlobal 测试非全服邮件应可移除
// Scenario: 非全服邮件的全服邮件历史记录可以移除
func TestIsHistoryRemoveableNonGlobal(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := &public_protocol_pbdesc.DMailRecord{
		MailId:       1001,
		MajorType:    1,
		IsGlobalMail: false,
	}

	// Act & Assert
	assert.True(t, mgr.IsHistoryRemoveable(record), "non-global mail should be history removeable")
}

// TestIsHistoryRemoveableInvalidMailId 测试无效 mail_id 应可移除
// Scenario: mail_id=0 或 major_type=0 的记录均可从历史移除
func TestIsHistoryRemoveableInvalidMailId(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := &public_protocol_pbdesc.DMailRecord{
		MailId:       0,
		MajorType:    1,
		IsGlobalMail: true,
	}

	// Act & Assert
	assert.True(t, mgr.IsHistoryRemoveable(record))
}

// TestIsHistoryRemoveableInPendingRemove 测试在待移除队列中的邮件不应可移除
// Scenario: 处于 pendingToRemove 中的邮件历史不可移除
func TestIsHistoryRemoveableInPendingRemove(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.pendingToRemove[1001] = &global_mail_data.GlobalMailRecordEntry{
		ZoneId: 1,
		Record: record.Clone(),
	}

	// Act & Assert
	assert.False(t, mgr.IsHistoryRemoveable(record),
		"mail in pending remove should not be history removeable")
}

// TestIsHistoryRemoveableStillInMailbox 测试仍在邮箱中的邮件不应可移除
// Scenario: 在 mailBoxIdIndex 中的邮件历史不可移除
func TestIsHistoryRemoveableStillInMailbox(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	// Act & Assert
	assert.False(t, mgr.IsHistoryRemoveable(record),
		"mail still in mailbox should not be history removeable")
}

// TestIsHistoryRemoveableAfterLeakTimeout 测试超过泄漏检查超时后可移除
// Scenario: lastSuccessFetchTimepoint 足够新时，找不到的旧邮件可以移除
func TestIsHistoryRemoveableAfterLeakTimeout(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	now := time.Now().Unix()
	record := newTestGlobalMailRecord(1001, 1)
	record.DeliveryTime = now - global_mail_data.EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT - 100
	mgr.lastSuccessFetchTimepoint = now

	// Act & Assert
	assert.True(t, mgr.IsHistoryRemoveable(record),
		"mail past leak check timeout should be history removeable")
}

// ==================== IsRecordRemoveable 测试 ====================

// TestIsRecordRemoveableNil 测试 nil 记录应可移除
// Scenario: nil 记录视为可移除
func TestIsRecordRemoveableNil(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act & Assert
	assert.True(t, mgr.IsRecordRemoveable(nil), "nil record should be removeable")
}

// TestIsRecordRemoveableNonGlobal 测试非全服邮件不应可移除
// Scenario: 非全服邮件的是个人邮件，全服邮件管理器不应移除
func TestIsRecordRemoveableNonGlobal(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := &public_protocol_pbdesc.DMailRecord{
		MailId:       1001,
		MajorType:    1,
		IsGlobalMail: false,
	}

	// Act & Assert
	assert.False(t, mgr.IsRecordRemoveable(record), "non-global mail should not be removeable by global manager")
}

// TestIsRecordRemoveableInPendingRemove 测试在待删除列表中的邮件可移除
// Scenario: pendingToRemove 中的邮件应可移除
func TestIsRecordRemoveableInPendingRemove(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.pendingToRemove[1001] = &global_mail_data.GlobalMailRecordEntry{
		ZoneId: 1,
		Record: record.Clone(),
	}

	// Act & Assert
	assert.True(t, mgr.IsRecordRemoveable(record),
		"mail in pending remove should be removeable")
}

// TestIsRecordRemoveableExpired 测试过期邮件可移除
// Scenario: 超过 expiredTime + timeTolerate 后应可移除
func TestIsRecordRemoveableExpired(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	now := time.Now().Unix()
	record := &public_protocol_pbdesc.DMailRecord{
		MailId:       1001,
		MajorType:    1,
		IsGlobalMail: true,
		ExpiredTime:  now - global_mail_data.EN_CL_MAIL_GLOBAL_TIME_TOLERATE - 100,
		RemoveTime:   now - global_mail_data.EN_CL_MAIL_GLOBAL_TIME_TOLERATE - 100,
	}

	// Act & Assert
	assert.True(t, mgr.IsRecordRemoveable(record),
		"mail past expired+tolerate time should be removeable")
}

// TestIsRecordRemoveableNotExpired 测试未过期邮件不可移除
// Scenario: 仍在有效期内且不在其他队列中应不可移除
func TestIsRecordRemoveableNotExpired(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1) // 正常有效期

	// Act & Assert
	assert.False(t, mgr.IsRecordRemoveable(record),
		"valid mail should not be removeable")
}

// ==================== UpdateFromDB 测试 ====================

// TestUpdateFromDBAddNew 测试从数据库添加新邮件
// Scenario: 数据库中的新邮件应被添加到管理器中
func TestUpdateFromDBAddNew(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{
			newTestGlobalMailRecord(1001, 1),
			newTestGlobalMailRecord(1002, 1),
		},
	}

	// Act
	ret := mgr.UpdateFromDB(1, 1, blobData, false)

	// Assert
	assert.False(t, ret, "should return false when no rewrite needed")
	assert.NotNil(t, mgr.GetMailRaw(1001), "mail 1001 should be added")
	assert.NotNil(t, mgr.GetMailRaw(1002), "mail 1002 should be added")
}

// TestUpdateFromDBUpdateExisting 测试从数据库更新已有邮件
// Scenario: 数据库中已有的邮件应被更新而非重新添加
func TestUpdateFromDBUpdateExisting(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	updateRecord := newTestGlobalMailRecord(1001, 1)
	updateRecord.MinorType = 99
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{updateRecord},
	}

	// Act
	mgr.UpdateFromDB(1, 1, blobData, false)

	// Assert
	assert.Equal(t, 1, len(mgr.GetAllGlobalMails()), "should still have 1 mail")
}

// TestUpdateFromDBRemoveInvalid 测试从数据库同步时移除不在数据库中的邮件
// Scenario: 本地有而数据库没有的邮件应被删除
func TestUpdateFromDBRemoveInvalid(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record1 := newTestGlobalMailRecord(1001, 1)
	record2 := newTestGlobalMailRecord(1002, 1)
	mgr.AddGlobalMail(1, record1)
	mgr.AddGlobalMail(1, record2)

	// 数据库只有 1001
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{
			newTestGlobalMailRecord(1001, 1),
		},
	}

	// Act
	mgr.UpdateFromDB(1, 1, blobData, false)

	// Assert
	assert.NotNil(t, mgr.GetMailRaw(1001), "mail 1001 should still exist")
	assert.Nil(t, mgr.GetMailRaw(1002), "mail 1002 should be removed (not in DB)")
}

// TestUpdateFromDBRewriteMode 测试从数据库更新时重写模式
// Scenario: rewriteDbData=true 时过期邮件应触发重写
func TestUpdateFromDBRewriteMode(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	expiredRecord := newExpiredGlobalMailRecord(1001, 1)
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{expiredRecord},
	}

	// Act
	ret := mgr.UpdateFromDB(1, 1, blobData, true)

	// Assert
	assert.True(t, ret, "should return true when rewrite is needed for expired mails")
}

// TestUpdateFromDBPendingRemoveList 测试从数据库更新待移除队列
// Scenario: blobData 中的 PendingRemoveList 应被处理
func TestUpdateFromDBPendingRemoveList(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	pendingRecord := newTestGlobalMailRecord(2001, 1)
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{
			newTestGlobalMailRecord(1001, 1),
		},
		PendingRemoveList: []*public_protocol_pbdesc.DMailRecord{pendingRecord},
	}

	// Act
	mgr.UpdateFromDB(1, 1, blobData, false)

	// Assert
	assert.NotNil(t, mgr.GetMailRaw(1001), "mail 1001 should be added")
	assert.Contains(t, mgr.pendingToRemove, int64(2001),
		"pending remove from DB should be tracked")
}

// ==================== SetMailContentLoaded 测试 ====================

// TestSetMailContentLoaded 测试设置邮件内容已加载
// Scenario: 加载内容后应从 unloaded 索引移除并触发用户重载
func TestSetMailContentLoaded(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)

	assert.Contains(t, mgr.mailUnloadedIndex, int64(1001))

	// Act
	mgr.SetMailContentLoaded(1001)

	// Assert
	assert.NotContains(t, mgr.mailUnloadedIndex, int64(1001),
		"should be removed from unloaded index")
	assert.True(t, mgr.forceUsersReload, "should force users reload after content loaded")
}

// TestSetMailContentLoadedNotInUnloaded 测试设置不在 unloaded 索引中的邮件
// Scenario: 如果不在 unloaded 中，应静默忽略
func TestSetMailContentLoadedNotInUnloaded(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act - 不应 panic
	mgr.SetMailContentLoaded(9999)

	// Assert
	assert.False(t, mgr.forceUsersReload, "should not trigger reload for non-existent mail")
}

// ==================== SetLastSuccessFetchTimepoint 测试 ====================

// TestSetLastSuccessFetchTimepoint 测试设置最后成功拉取时间点
// Scenario: 设置后应反映在后续判断中
func TestSetLastSuccessFetchTimepoint(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	now := time.Now().Unix()

	// Act
	mgr.SetLastSuccessFetchTimepoint(now)

	// Assert
	assert.Equal(t, now, mgr.lastSuccessFetchTimepoint)
}

// ==================== PendingToRemoveContents 测试 ====================

// TestPendingToRemoveContents 测试待移除内容管理
// Scenario: 添加、获取、删除和清除操作
func TestPendingToRemoveContents(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act - 添加
	mgr.AddPendingToRemoveContent(1001)
	mgr.AddPendingToRemoveContent(1002)
	mgr.AddPendingToRemoveContent(1003)

	// Assert
	list := mgr.GetPendingToRemoveContentsList()
	assert.Equal(t, 3, len(list), "should have 3 pending remove contents")

	// Act - 移除一个
	mgr.RemovePendingToRemoveContent(1002)
	list = mgr.GetPendingToRemoveContentsList()
	assert.Equal(t, 2, len(list), "should have 2 pending remove contents after removal")

	// Act - 清除所有
	mgr.ClearPendingToRemoveContents()
	list = mgr.GetPendingToRemoveContentsList()
	assert.Equal(t, 0, len(list), "should be empty after clear")
}

// ==================== FetchAllUnloadedMails 测试 ====================

// TestFetchAllUnloadedMails 测试获取所有未加载内容的邮件
// Scenario: 应只返回有效的未加载邮件ID
func TestFetchAllUnloadedMails(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record1 := newTestGlobalMailRecord(1001, 1)
	record2 := newTestGlobalMailRecord(1002, 1)
	mgr.AddGlobalMail(1, record1)
	mgr.AddGlobalMail(1, record2)

	// Act
	unloaded := mgr.FetchAllUnloadedMails()

	// Assert
	assert.Equal(t, 2, len(unloaded), "should return 2 unloaded mail IDs")
}

// TestFetchAllUnloadedMailsAfterLoad 测试加载内容后不再返回
// Scenario: 加载内容后邮件不应再出现在未加载列表中
func TestFetchAllUnloadedMailsAfterLoad(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	record := newTestGlobalMailRecord(1001, 1)
	mgr.AddGlobalMail(1, record)
	mgr.SetMailContentLoaded(1001)

	// Act
	unloaded := mgr.FetchAllUnloadedMails()

	// Assert
	assert.Equal(t, 0, len(unloaded), "should return empty after content loaded")
}

// ==================== ResetAsyncJobsProtect 测试 ====================

// TestResetAsyncJobsProtect 测试重置异步任务保护
// Scenario: 重置后 taskNextTimepoint 应为 0
func TestResetAsyncJobsProtect(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()
	mgr.taskNextTimepoint = time.Now().Unix() + 1000

	// Act
	mgr.ResetAsyncJobsProtect()

	// Assert
	assert.Equal(t, int64(0), mgr.taskNextTimepoint, "should reset to 0")
}

// ==================== Name 测试 ====================

// TestGlobalMailManagerName 测试模块名称
// Scenario: Name() 应返回固定名称
func TestGlobalMailManagerName(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Act & Assert
	assert.Equal(t, "GlobalMailManager", mgr.Name())
}

// ==================== 综合场景测试 ====================

// TestAddUpdateRemoveWorkflow 测试完整的添加→更新→移除工作流
// Scenario: 验证全服邮件从添加到删除的完整生命周期
func TestAddUpdateRemoveWorkflow(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// Step 1: 添加邮件
	record := newTestGlobalMailRecord(1001, 1)
	ret := mgr.AddGlobalMail(1, record)
	assert.Equal(t, int32(0), ret)
	assert.NotNil(t, mgr.GetMailRaw(1001))

	// Step 2: 更新邮件
	updateRecord := newTestGlobalMailRecord(1001, 1)
	updateRecord.MinorType = 99
	ok := mgr.UpdateGlobalMail(1, updateRecord)
	assert.True(t, ok, "update should succeed")

	// Step 3: 移除邮件
	mgr.RemoveGlobalMail(1001)
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should be removed")
	assert.Contains(t, mgr.pendingToRemove, int64(1001), "should be in pending remove")

	// Step 4: 验证不能再添加已移除的邮件
	addRecord := newTestGlobalMailRecord(1001, 1)
	ret = mgr.AddGlobalMail(1, addRecord)
	assert.Equal(t, int32(0), ret, "should return 0 for ignored mail")
	assert.Nil(t, mgr.GetMailRaw(1001), "removed mail should not be re-added")
}

// TestMultipleZonesAndTypesManagement 测试多区域多类型管理
// Scenario: 在不同 zone 和 majorType 下管理邮件
func TestMultipleZonesAndTypesManagement(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// 添加不同 zone 和类型的邮件
	mgr.AddGlobalMail(1, newTestGlobalMailRecord(1001, 1))
	mgr.AddGlobalMail(1, newTestGlobalMailRecord(1002, 2))
	mgr.AddGlobalMail(2, newTestGlobalMailRecord(1003, 1))
	mgr.AddGlobalMail(2, newTestGlobalMailRecord(1004, 3))

	// Assert
	assert.Equal(t, 4, len(mgr.GetAllGlobalMails()))
	assert.NotNil(t, mgr.GetMailBoxByType(1, 1))
	assert.NotNil(t, mgr.GetMailBoxByType(1, 2))
	assert.NotNil(t, mgr.GetMailBoxByType(2, 1))
	assert.NotNil(t, mgr.GetMailBoxByType(2, 3))
	assert.Nil(t, mgr.GetMailBoxByType(1, 3), "zone 1 should not have major_type 3")
	assert.Nil(t, mgr.GetMailBoxByType(2, 2), "zone 2 should not have major_type 2")

	// 移除 zone1 的 major_type 1 邮件
	mgr.RemoveGlobalMail(1001)
	assert.Nil(t, mgr.GetMailBoxByType(1, 1), "zone 1 major_type 1 should be empty")
	assert.NotNil(t, mgr.GetMailBoxByType(1, 2), "zone 1 major_type 2 should still exist")
}

// TestUpdateFromDBThenRemoveSyncIntegration 测试 UpdateFromDB 与 Remove 的整合场景
// Scenario: 先通过 UpdateFromDB 加入邮件，然后移除部分，再次 UpdateFromDB 应正确同步
func TestUpdateFromDBThenRemoveSyncIntegration(t *testing.T) {
	// Arrange
	mgr := newTestGlobalMailManager()

	// 第一次从 DB 加载
	blobData := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{
			newTestGlobalMailRecord(1001, 1),
			newTestGlobalMailRecord(1002, 1),
			newTestGlobalMailRecord(1003, 1),
		},
	}
	mgr.UpdateFromDB(1, 1, blobData, false)
	assert.Equal(t, 3, len(mgr.GetAllGlobalMails()))

	// 手动移除一封
	mgr.RemoveGlobalMail(1002)
	assert.Equal(t, 2, len(mgr.GetAllGlobalMails()))

	// 第二次从 DB 加载（DB 中仍有 1001、1003，新增 1004）
	blobData2 := &private_protocol_pbdesc.DatabaseGlobalMailBlobData{
		MailRecords: []*public_protocol_pbdesc.DMailRecord{
			newTestGlobalMailRecord(1001, 1),
			newTestGlobalMailRecord(1003, 1),
			newTestGlobalMailRecord(1004, 1),
		},
	}
	mgr.UpdateFromDB(1, 1, blobData2, false)

	// Assert
	assert.NotNil(t, mgr.GetMailRaw(1001), "mail 1001 should exist")
	assert.NotNil(t, mgr.GetMailRaw(1003), "mail 1003 should exist")
	assert.NotNil(t, mgr.GetMailRaw(1004), "mail 1004 should be added")
	// 1002 已被移除，不应被恢复
	assert.Nil(t, mgr.GetMailRaw(1002), "removed mail 1002 should not be restored")
}
