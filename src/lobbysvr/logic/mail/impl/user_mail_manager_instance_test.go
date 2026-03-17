package lobbysvr_logic_mail_internal

import (
	"context"
	"log/slog"
	"testing"
	"time"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	global_mail_impl "github.com/atframework/atsf4g-go/service-lobbysvr/logic/global_mail/impl"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
	"github.com/atframework/libatapp-go"
	"github.com/stretchr/testify/assert"
)

// ==================== Mock RpcContext ====================

// mockRpcContext 用于测试的模拟 RpcContext
type mockRpcContext struct {
	app libatapp.AppImpl
}

func newMockRpcContext() *mockRpcContext {
	app := libatapp.CreateAppInstance()
	// 注册 GlobalMailManager 模块，AddGlobalMail 等方法需要通过 app 获取该模块
	libatapp.AtappAddModule(app, global_mail_impl.NewGlobalMailManager(app))
	return &mockRpcContext{
		app: app,
	}
}

func (m *mockRpcContext) GetNow() time.Time                                          { return time.Now() }
func (m *mockRpcContext) GetSysNow() time.Time                                       { return time.Now() }
func (m *mockRpcContext) GetApp() libatapp.AppImpl                                   { return m.app }
func (m *mockRpcContext) GetAction() cd.TaskActionImpl                               { return nil }
func (m *mockRpcContext) BindAction(_ cd.TaskActionImpl)                             {}
func (m *mockRpcContext) GetContext() context.Context                                { return context.Background() }
func (m *mockRpcContext) GetCancelFn() context.CancelFunc                            { return nil }
func (m *mockRpcContext) SetContext(_ context.Context)                               {}
func (m *mockRpcContext) SetCancelFn(_ context.CancelFunc)                           {}
func (m *mockRpcContext) SetContextCancelFn(_ context.Context, _ context.CancelFunc) {}

func (m *mockRpcContext) LogWithLevelContextWithCaller(_ uintptr, _ context.Context, _ slog.Level, _ string, _ ...any) {
}
func (m *mockRpcContext) LogWithLevelWithCaller(_ uintptr, _ slog.Level, _ string, _ ...any) {}
func (m *mockRpcContext) LogErrorContext(_ context.Context, _ string, _ ...any)              {}
func (m *mockRpcContext) LogError(_ string, _ ...any)                                        {}
func (m *mockRpcContext) LogWarnContext(_ context.Context, _ string, _ ...any)               {}
func (m *mockRpcContext) LogWarn(_ string, _ ...any)                                         {}
func (m *mockRpcContext) LogInfoContext(_ context.Context, _ string, _ ...any)               {}
func (m *mockRpcContext) LogInfo(_ string, _ ...any)                                         {}
func (m *mockRpcContext) LogDebugContext(_ context.Context, _ string, _ ...any)              {}
func (m *mockRpcContext) LogDebug(_ string, _ ...any)                                        {}

// ==================== 辅助函数 ====================

// newTestUserMailManager 创建测试用的 UserMailManager 实例（不绑定 owner）
func newTestUserMailManager() *UserMailManager {
	return &UserMailManager{
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
}

// newTestMailRecord 创建测试用的邮件记录
func newTestMailRecord(mailId int64, majorType int32) *public_protocol_pbdesc.DMailRecord {
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
	}
}

// newTestMailContent 创建测试用的邮件内容
func newTestMailContent(mailId int64, majorType int32) *public_protocol_pbdesc.DMailContent {
	now := time.Now().Unix()
	return &public_protocol_pbdesc.DMailContent{
		MailId:      mailId,
		MajorType:   majorType,
		MinorType:   1,
		Title:       "Test Mail",
		Content:     "Test Content",
		ExpiredTime: now + 86400,
	}
}

// newExpiredMailRecord 创建已过期的邮件记录
func newExpiredMailRecord(mailId int64, majorType int32) *public_protocol_pbdesc.DMailRecord {
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
	}
}

// newFutureMailRecord 创建未来生效的邮件记录
func newFutureMailRecord(mailId int64, majorType int32, startTime int64) *public_protocol_pbdesc.DMailRecord {
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
	}
}

// ==================== AddMail 测试 ====================

// TestAddMailSuccess 测试成功添加邮件
// Scenario: 正常添加一封有效邮件，应成功添加并标记为脏
func TestAddMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)

	// Act
	ret := mgr.AddMail(ctx, record, content)

	// Assert
	assert.Equal(t, int32(0), ret, "should return 0 for success")
	assert.True(t, mgr.IsDirty(), "manager should be dirty after adding mail")

	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail, "mail should exist in index")
	assert.NotNil(t, mail.Record, "mail record should not be nil")
	assert.NotNil(t, mail.Content, "mail content should not be nil")
	assert.Equal(t, int64(1001), mail.Record.GetMailId(), "mail id should match")

	mailBox := mgr.GetMailBoxByMajorType(1)
	assert.NotNil(t, mailBox, "mailbox for major_type 1 should exist")
	assert.Equal(t, 1, len(mailBox.Mails), "mailbox should contain 1 mail")
}

// TestAddMailInvalidParam 测试添加无效参数的邮件
// Scenario: mail_id=0 或 major_type=0 时添加应失败
func TestAddMailInvalidParamZeroMailId(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(0, 1) // mail_id = 0

	// Act
	ret := mgr.AddMail(ctx, record, nil)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), ret,
		"should return EN_ERR_INVALID_PARAM when mail_id is 0")
	assert.False(t, mgr.IsDirty(), "manager should not be dirty")
}

// TestAddMailInvalidParamZeroMajorType 测试 major_type=0 时添加应失败
// Scenario: major_type=0 的邮件不允许添加
func TestAddMailInvalidParamZeroMajorType(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 0) // major_type = 0

	// Act
	ret := mgr.AddMail(ctx, record, nil)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), ret,
		"should return EN_ERR_INVALID_PARAM when major_type is 0")
}

// TestAddMailDuplicate 测试重复添加邮件
// Scenario: 同一 mail_id 重复添加应更新已有记录而非新增
func TestAddMailDuplicate(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record1 := newTestMailRecord(1001, 1)
	content1 := newTestMailContent(1001, 1)
	record2 := newTestMailRecord(1001, 1)
	record2.MinorType = 2 // 修改 minor_type

	// Act
	ret1 := mgr.AddMail(ctx, record1, content1)
	ret2 := mgr.AddMail(ctx, record2, nil)

	// Assert
	assert.Equal(t, int32(0), ret1, "first add should succeed")
	assert.Equal(t, int32(0), ret2, "second add (duplicate) should return 0")

	// 邮件只应有1份
	mailBox := mgr.GetMailBoxByMajorType(1)
	assert.NotNil(t, mailBox, "mailbox should exist")
	assert.Equal(t, 1, len(mailBox.Mails), "should still have only 1 mail after duplicate add")
}

// TestAddMailWithoutContent 测试添加不带内容的邮件
// Scenario: content=nil 时也应成功添加，邮件应出现在未加载索引中
func TestAddMailWithoutContent(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)

	// Act
	ret := mgr.AddMail(ctx, record, nil)

	// Assert
	assert.Equal(t, int32(0), ret, "should succeed adding mail without content")
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail, "mail should exist")
	assert.Nil(t, mail.Content, "content should be nil")
	assert.Contains(t, mgr.mailBoxUnloadedIndex, int64(1001), "should be in unloaded index")
}

// TestAddMailInPendingRemoveList 测试在移除队列中的邮件添加应被忽略
// Scenario: mail_id 已在 pendingRemoveList 中，再次添加应被跳过
func TestAddMailInPendingRemoveList(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)

	mgr.pendingRemoveList[1001] = record.Clone()

	// Act
	ret := mgr.AddMail(ctx, record, content)

	// Assert
	assert.Equal(t, int32(0), ret, "should return 0 for ignored mail")
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should not be added to mailbox")
}

// TestAddMailExpired 测试添加已过期的邮件应被忽略
// Scenario: 过期邮件不应被添加到邮箱
func TestAddMailExpired(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newExpiredMailRecord(1001, 1)

	// Act
	ret := mgr.AddMail(ctx, record, nil)

	// Assert
	assert.Equal(t, int32(0), ret, "should return 0 for expired mail (silently ignored)")
	assert.Nil(t, mgr.GetMailRaw(1001), "expired mail should not exist in index")
}

// TestAddMailMultipleMajorTypes 测试添加不同 major_type 的邮件
// Scenario: 添加不同类型的邮件应各自分到不同的 mailbox 中
func TestAddMailMultipleMajorTypes(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record1 := newTestMailRecord(1001, 1)
	content1 := newTestMailContent(1001, 1)
	record2 := newTestMailRecord(1002, 2)
	content2 := newTestMailContent(1002, 2)

	// Act
	ret1 := mgr.AddMail(ctx, record1, content1)
	ret2 := mgr.AddMail(ctx, record2, content2)

	// Assert
	assert.Equal(t, int32(0), ret1)
	assert.Equal(t, int32(0), ret2)
	assert.NotNil(t, mgr.GetMailBoxByMajorType(1), "mailbox type 1 should exist")
	assert.NotNil(t, mgr.GetMailBoxByMajorType(2), "mailbox type 2 should exist")
	assert.Equal(t, 1, len(mgr.GetMailBoxByMajorType(1).Mails))
	assert.Equal(t, 1, len(mgr.GetMailBoxByMajorType(2).Mails))
}

// TestAddMailFuture 测试添加未来生效的邮件
// Scenario: 未来生效的邮件应成功添加并增加 FutureMailCount
func TestAddMailFuture(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	futureStart := time.Now().Unix() + 86400
	record := newFutureMailRecord(1001, 1, futureStart)
	content := newTestMailContent(1001, 1)

	// Act
	ret := mgr.AddMail(ctx, record, content)

	// Assert
	assert.Equal(t, int32(0), ret, "future mail should be added successfully")
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail)

	mailBox := mgr.GetMailBoxByMajorType(1)
	assert.NotNil(t, mailBox)
	assert.Equal(t, int32(1), mailBox.FutureMailCount, "should have 1 future mail")
}

// ==================== RemoveMail 测试 ====================

// TestRemoveMailSuccess 测试成功移除邮件
// Scenario: 移除已存在的邮件，验证邮件被正确移除
func TestRemoveMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)
	mgr.ClearDirty()

	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.RemoveMail(ctx, 1001, out)

	// Assert
	assert.Equal(t, int32(0), ret, "remove should succeed")
	assert.NotNil(t, out.Record, "operation result should contain the removed record")
	assert.Equal(t, int64(1001), out.Record.GetMailId())
	assert.True(t, mgr.IsDirty(), "should be dirty after remove")

	// 邮件应已从索引中删除
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should not exist in index after removal")
	assert.Nil(t, mgr.GetMailBoxByMajorType(1), "empty mailbox should be cleaned up")
}

// TestRemoveMailNotFound 测试移除不存在的邮件
// Scenario: 尝试移除不存在的邮件应返回错误
func TestRemoveMailNotFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.RemoveMail(ctx, 9999, out)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND), ret,
		"should return MAIL_NOT_FOUND error")
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND), out.Result)
}

// TestRemoveMailNilOutput 测试移除时 out 为 nil 的场景
// Scenario: 即使 out 为 nil 也应正常移除邮件
func TestRemoveMailNilOutput(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	// Act
	ret := mgr.RemoveMail(ctx, 1001, nil)

	// Assert
	assert.Equal(t, int32(0), ret, "remove with nil output should succeed")
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should be removed")
}

// TestRemoveMailPendingRemoveList 测试移除后的邮件进入待移除队列
// Scenario: 非全服邮件移除后应加入 pendingRemoveList
func TestRemoveMailPendingRemoveList(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	// Act
	mgr.RemoveMail(ctx, 1001, nil)

	// Assert
	assert.Contains(t, mgr.pendingRemoveList, int64(1001),
		"removed mail should be in pending remove list")
}

// TestRemoveMailStatusRemoved 测试移除后邮件状态包含 REMOVED 标记
// Scenario: 移除邮件时 dirty cache 的记录应带有 REMOVED 状态
func TestRemoveMailStatusRemoved(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	// Act
	mgr.RemoveMail(ctx, 1001, nil)

	// Assert
	dirtyRecord, exists := mgr.dirtyCache.DirtyMails[1001]
	assert.True(t, exists, "mail should be in dirty cache")
	assert.NotEqual(t, 0, dirtyRecord.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_REMOVED),
		"dirty record should have REMOVED status")
}

// ==================== ReadMail 测试 ====================

// TestReadMailSuccess 测试成功读取邮件
// Scenario: 读取已有邮件应标记为已读并返回记录
func TestReadMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 1001, out, false)

	// Assert
	assert.Equal(t, int32(0), ret, "read should succeed")
	assert.NotNil(t, out.Record, "output record should not be nil")
	assert.Equal(t, int64(1001), out.Record.GetMailId())

	// 验证邮件被标记为已读
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail)
	assert.NotEqual(t, 0, mail.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
		"mail should have READ status")
}

// TestReadMailNotFound 测试读取不存在的邮件
// Scenario: 读取不存在的邮件应返回 NOT_FOUND
func TestReadMailNotFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 9999, out, false)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND), ret,
		"should return MAIL_NOT_FOUND error")
}

// TestReadMailNoContent 测试读取无内容的邮件
// Scenario: 邮件存在但没有 content 应返回 NOT_FOUND
func TestReadMailNoContent(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	mgr.AddMail(ctx, record, nil) // 不提供 content

	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 1001, out, false)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_MAIL_NOT_FOUND), ret,
		"should return MAIL_NOT_FOUND for mail without content")
}

// TestReadMailWithRemove 测试读取邮件并移除
// Scenario: needRemove=true 时读取邮件后应将其移除
func TestReadMailWithRemove(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	out := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 1001, out, true)

	// Assert
	assert.Equal(t, int32(0), ret, "read with remove should succeed")
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should be removed after read with needRemove")
}

// TestReadMailAlreadyRead 测试重复读取已读邮件
// Scenario: 已读的邮件再次读取不应改变状态
func TestReadMailAlreadyRead(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	out1 := &public_protocol_pbdesc.DMailOperationResult{}
	mgr.ReadMail(ctx, 1001, out1, false) // 第一次读取

	out2 := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 1001, out2, false)

	// Assert
	assert.Equal(t, int32(0), ret, "reading already-read mail should succeed")
	assert.NotNil(t, mgr.GetMailRaw(1001), "mail should still exist")
}

// TestReadMailAlreadyReadWithRemove 测试已读邮件再读并移除
// Scenario: 已读的邮件再次读取时 needRemove=true 应移除
func TestReadMailAlreadyReadWithRemove(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	out1 := &public_protocol_pbdesc.DMailOperationResult{}
	mgr.ReadMail(ctx, 1001, out1, false) // 第一次读取

	out2 := &public_protocol_pbdesc.DMailOperationResult{}

	// Act
	ret := mgr.ReadMail(ctx, 1001, out2, true)

	// Assert
	assert.Equal(t, int32(0), ret, "should succeed")
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should be removed")
}

// ==================== ReadAll 测试 ====================

// TestReadAllSuccess 测试批量读取邮件
// Scenario: 批量读取指定 majorType 的所有邮件
func TestReadAllSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	for i := int64(1); i <= 3; i++ {
		record := newTestMailRecord(i, 1)
		content := newTestMailContent(i, 1)
		mgr.AddMail(ctx, record, content)
	}

	// Act
	ret := mgr.ReadAll(ctx, 1, 0, nil, false)

	// Assert
	assert.Equal(t, int32(0), ret, "read all should succeed")

	// 验证所有邮件被标记为已读
	for i := int64(1); i <= 3; i++ {
		mail := mgr.GetMailRaw(i)
		assert.NotNil(t, mail, "mail %d should still exist", i)
		assert.NotEqual(t, 0, mail.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
			"mail %d should be marked as read", i)
	}
}

// TestReadAllEmptyMailbox 测试批量读取空邮箱
// Scenario: 读取不存在的 majorType 应正常返回
func TestReadAllEmptyMailbox(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	// Act
	ret := mgr.ReadAll(ctx, 99, 0, nil, false)

	// Assert
	assert.Equal(t, int32(0), ret, "read all on empty mailbox should succeed")
}

// TestReadAllWithRemove 测试批量读取并移除（不带附件的邮件应被移除）
// Scenario: needRemove=true 时应移除不带附件的邮件
func TestReadAllWithRemove(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	for i := int64(1); i <= 3; i++ {
		record := newTestMailRecord(i, 1)
		content := newTestMailContent(i, 1) // 无附件
		mgr.AddMail(ctx, record, content)
	}

	// Act
	ret := mgr.ReadAll(ctx, 1, 0, nil, true)

	// Assert
	assert.Equal(t, int32(0), ret)
	// 无附件的邮件应全部被移除
	for i := int64(1); i <= 3; i++ {
		assert.Nil(t, mgr.GetMailRaw(i), "mail %d should be removed", i)
	}
}

// TestReadAllWithMinorTypeFilter 测试批量读取时按 minorType 过滤
// Scenario: 指定 minorType 时只应读取匹配的邮件
func TestReadAllWithMinorTypeFilter(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	record1 := newTestMailRecord(1, 1)
	record1.MinorType = 1
	content1 := newTestMailContent(1, 1)
	mgr.AddMail(ctx, record1, content1)

	record2 := newTestMailRecord(2, 1)
	record2.MinorType = 2
	content2 := newTestMailContent(2, 1)
	mgr.AddMail(ctx, record2, content2)

	// Act - 只读 minorType=1 的邮件
	ret := mgr.ReadAll(ctx, 1, 1, nil, false)

	// Assert
	assert.Equal(t, int32(0), ret)

	mail1 := mgr.GetMailRaw(1)
	assert.NotNil(t, mail1)
	assert.NotEqual(t, 0, mail1.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
		"mail with minorType=1 should be read")

	mail2 := mgr.GetMailRaw(2)
	assert.NotNil(t, mail2)
	assert.Equal(t, int32(0), mail2.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ),
		"mail with minorType=2 should not be read")
}

// ==================== InitFromDB / DumpToDB 测试 ====================

// TestInitFromDBSuccess 测试从数据库初始化邮件
// Scenario: 正常从 DatabaseTableUser 加载邮件数据
func TestInitFromDBSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	now := time.Now().Unix()
	dbUser := &private_protocol_pbdesc.DatabaseTableUser{}
	mailData := dbUser.MutableMailData()
	mailData.MailBox = []*public_protocol_pbdesc.DMailRecord{
		{
			MailId:       1001,
			MajorType:    1,
			MinorType:    1,
			DeliveryTime: now,
			StartTime:    now - 60,
			ExpiredTime:  now + 86400,
			RemoveTime:   now + 86400*2,
		},
		{
			MailId:       1002,
			MajorType:    2,
			MinorType:    1,
			DeliveryTime: now,
			StartTime:    now - 60,
			ExpiredTime:  now + 86400,
			RemoveTime:   now + 86400*2,
		},
	}
	mailData.PendingRemoveList = []*public_protocol_pbdesc.DMailRecord{
		{
			MailId: 2001,
		},
	}
	mailData.ReceivedGlobalMails = []*public_protocol_pbdesc.DMailRecord{
		{
			MailId:       3001,
			IsGlobalMail: true,
		},
	}

	// Act
	result := mgr.InitFromDB(ctx, dbUser)

	// Assert
	assert.Nil(t, result.Error, "InitFromDB should succeed")
	assert.Equal(t, int32(0), result.ResponseCode)

	assert.NotNil(t, mgr.GetMailRaw(1001), "mail 1001 should be loaded")
	assert.NotNil(t, mgr.GetMailRaw(1002), "mail 1002 should be loaded")
	assert.NotNil(t, mgr.GetMailBoxByMajorType(1), "mailbox for major_type 1 should exist")
	assert.NotNil(t, mgr.GetMailBoxByMajorType(2), "mailbox for major_type 2 should exist")
	assert.Contains(t, mgr.pendingRemoveList, int64(2001), "pending remove list should contain 2001")
	assert.Contains(t, mgr.receivedGlobalMails, int64(3001), "received global mails should contain 3001")
	assert.False(t, mgr.isDirty, "should not be dirty after init")
}

// TestInitFromDBEmpty 测试从空数据库初始化
// Scenario: 没有邮件数据时应正常初始化
func TestInitFromDBEmpty(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	dbUser := &private_protocol_pbdesc.DatabaseTableUser{}

	// Act
	result := mgr.InitFromDB(ctx, dbUser)

	// Assert
	assert.Nil(t, result.Error, "InitFromDB with empty data should succeed")
	assert.Equal(t, 0, len(mgr.mailBoxIdIndex), "should have no mails")
}

// TestDumpToDBRoundTrip 测试 InitFromDB 和 DumpToDB 的往返一致性
// Scenario: 先添加邮件再 DumpToDB，然后 InitFromDB 加载回来，数据应一致
func TestDumpToDBRoundTrip(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	record1 := newTestMailRecord(1001, 1)
	content1 := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record1, content1)

	record2 := newTestMailRecord(1002, 2)
	content2 := newTestMailContent(1002, 2)
	mgr.AddMail(ctx, record2, content2)

	// Act - DumpToDB
	dbUser := &private_protocol_pbdesc.DatabaseTableUser{}
	dumpResult := mgr.DumpToDB(ctx, dbUser)
	assert.Nil(t, dumpResult.Error)

	// 用另一个 mgr 从 DB 数据加载
	mgr2 := newTestUserMailManager()
	loadResult := mgr2.InitFromDB(ctx, dbUser)

	// Assert
	assert.Nil(t, loadResult.Error, "load from dumped data should succeed")
	assert.NotNil(t, mgr2.GetMailRaw(1001), "mail 1001 should be restored")
	assert.NotNil(t, mgr2.GetMailRaw(1002), "mail 1002 should be restored")
}

// ==================== IsDirty / ClearDirty 测试 ====================

// TestDirtyState 测试 dirty 状态管理
// Scenario: 初始非脏，添加后脏，清除后非脏
func TestDirtyState(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	// Assert - 初始状态
	assert.False(t, mgr.IsDirty(), "should not be dirty initially")

	// Act
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	// Assert - 添加后
	assert.True(t, mgr.IsDirty(), "should be dirty after add")

	// Act
	mgr.ClearDirty()

	// Assert - 清除后
	assert.False(t, mgr.IsDirty(), "should not be dirty after clear")
}

// ==================== MutableDirtyMail 测试 ====================

// TestMutableDirtyMailNew 测试新邮件的脏记录
// Scenario: isNew=true 时记录应同时出现在 DirtyMails 和 NewMails 中
func TestMutableDirtyMailNew(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	record := newTestMailRecord(1001, 1)

	// Act
	ret := mgr.MutableDirtyMail(record, true)

	// Assert
	assert.NotNil(t, ret)
	assert.Contains(t, mgr.dirtyCache.DirtyMails, int64(1001), "should be in dirty mails")
	assert.Contains(t, mgr.dirtyCache.NewMails, int64(1001), "should be in new mails")
}

// TestMutableDirtyMailExisting 测试已有邮件的脏记录更新
// Scenario: isNew=false 时记录只出现在 DirtyMails 中
func TestMutableDirtyMailExisting(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	record := newTestMailRecord(1001, 1)

	// Act
	ret := mgr.MutableDirtyMail(record, false)

	// Assert
	assert.NotNil(t, ret)
	assert.Contains(t, mgr.dirtyCache.DirtyMails, int64(1001))
	_, exists := mgr.dirtyCache.NewMails[1001]
	assert.False(t, exists, "should not be in new mails when isNew=false")
}

// TestMutableDirtyMailNil 测试 nil 记录返回 nil
// Scenario: 传入 nil 应返回 nil
func TestMutableDirtyMailNil(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()

	// Act
	ret := mgr.MutableDirtyMail(nil, true)

	// Assert
	assert.Nil(t, ret, "should return nil for nil record")
}

// ==================== FetchAllUserMailIds 测试 ====================

// TestFetchAllUserMailIds 测试获取所有用户邮件ID
// Scenario: 应只返回非全服邮件的ID
func TestFetchAllUserMailIds(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	// 添加普通邮件
	record1 := newTestMailRecord(1001, 1)
	content1 := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record1, content1)

	record2 := newTestMailRecord(1002, 1)
	content2 := newTestMailContent(1002, 1)
	mgr.AddMail(ctx, record2, content2)

	// 手动添加一封全服邮件到索引中
	globalRecord := newTestMailRecord(2001, 1)
	globalRecord.IsGlobalMail = true
	mgr.mailBoxIdIndex[2001] = &mail_data.MailData{
		Record: globalRecord,
	}

	// Act
	ids := mgr.FetchAllUserMailIds()

	// Assert
	assert.Equal(t, 2, len(ids), "should return 2 user mail ids (excluding global)")
	assert.NotContains(t, ids, int64(2001), "should not contain global mail id")
}

// TestFetchAllUserMailIdsEmpty 测试空邮箱获取ID
// Scenario: 空邮箱应返回空列表
func TestFetchAllUserMailIdsEmpty(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()

	// Act
	ids := mgr.FetchAllUserMailIds()

	// Assert
	assert.Empty(t, ids, "should return empty list for empty mailbox")
}

// ==================== GetMailRaw 测试 ====================

// TestGetMailRawFound 测试获取已有邮件
// Scenario: 邮件存在时应返回对应数据
func TestGetMailRawFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)

	// Act
	mail := mgr.GetMailRaw(1001)

	// Assert
	assert.NotNil(t, mail, "should return mail data")
	assert.Equal(t, int64(1001), mail.Record.GetMailId())
}

// TestGetMailRawNotFound 测试获取不存在的邮件
// Scenario: 邮件不存在时应返回nil
func TestGetMailRawNotFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()

	// Act
	mail := mgr.GetMailRaw(9999)

	// Assert
	assert.Nil(t, mail, "should return nil for non-existent mail")
}

// ==================== GetMailBoxByMajorType 测试 ====================

// TestGetMailBoxByMajorTypeFound 测试获取已有类型的邮箱
// Scenario: 邮箱类型存在时应返回对应 MailBox
func TestGetMailBoxByMajorTypeFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 5)
	content := newTestMailContent(1001, 5)
	mgr.AddMail(ctx, record, content)

	// Act
	mailBox := mgr.GetMailBoxByMajorType(5)

	// Assert
	assert.NotNil(t, mailBox, "should return mailbox for existing major type")
	assert.Equal(t, 1, len(mailBox.Mails))
}

// TestGetMailBoxByMajorTypeNotFound 测试获取不存在类型的邮箱
// Scenario: 邮箱类型不存在时应返回nil
func TestGetMailBoxByMajorTypeNotFound(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()

	// Act
	mailBox := mgr.GetMailBoxByMajorType(99)

	// Assert
	assert.Nil(t, mailBox, "should return nil for non-existent major type")
}

// ==================== ResetGlobalMailsCache 测试 ====================

// TestResetGlobalMailsCache 测试重置全服邮件缓存
// Scenario: 重置后 isGlobalMailsMerged 应为 false
func TestResetGlobalMailsCache(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	mgr.isGlobalMailsMerged = true
	mgr.mailAsyncTaskProtectTimepoint = time.Now().Add(time.Hour)

	// Act
	mgr.ResetGlobalMailsCache()

	// Assert
	assert.False(t, mgr.isGlobalMailsMerged, "should reset global mails merged flag")
	assert.True(t, mgr.mailAsyncTaskProtectTimepoint.IsZero(), "should reset async task protect timepoint")
}

// ==================== LazySaveCounter 测试 ====================

// TestLazySaveCounter 测试懒保存计数器
// Scenario: 初始为0，递增后正确，重置后回0
func TestLazySaveCounter(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()

	// Assert initial
	assert.Equal(t, 0, mgr.GetLazySaveCounter())

	// Act
	val1 := mgr.IncrementLazySaveCounter()
	val2 := mgr.IncrementLazySaveCounter()

	// Assert
	assert.Equal(t, 1, val1)
	assert.Equal(t, 2, val2)
	assert.Equal(t, 2, mgr.GetLazySaveCounter())

	// Act
	mgr.ResetLazySaveCounter()

	// Assert
	assert.Equal(t, 0, mgr.GetLazySaveCounter())
}

// ==================== SetMailContentLoaded 测试 ====================

// TestSetMailContentLoaded 测试设置邮件内容已加载
// Scenario: 从 unloaded 索引移除并更新 dirty cache
func TestSetMailContentLoaded(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	mgr.AddMail(ctx, record, nil) // 无 content，会进入 unloaded

	// 验证前置条件
	assert.Contains(t, mgr.mailBoxUnloadedIndex, int64(1001))

	// Act
	mgr.SetMailContentLoaded(ctx, 1001)

	// Assert
	assert.NotContains(t, mgr.mailBoxUnloadedIndex, int64(1001),
		"should be removed from unloaded index")
}

// ==================== RemovePendingRemoveItem 测试 ====================

// TestRemovePendingRemoveItem 测试从待移除队列中移除指定项
// Scenario: 移除后不再存在于 pendingRemoveList
func TestRemovePendingRemoveItem(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)
	mgr.AddMail(ctx, record, content)
	mgr.RemoveMail(ctx, 1001, nil) // 触发加入 pendingRemoveList

	assert.Contains(t, mgr.pendingRemoveList, int64(1001))

	// Act
	mgr.RemovePendingRemoveItem(1001)

	// Assert
	assert.NotContains(t, mgr.pendingRemoveList, int64(1001),
		"should be removed from pending remove list")
}

// ==================== 综合场景测试 ====================

// TestAddAndRemoveMultipleMails 测试添加和移除多封邮件
// Scenario: 添加多封邮件再逐个移除，验证状态一致性
func TestAddAndRemoveMultipleMails(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	for i := int64(1); i <= 5; i++ {
		record := newTestMailRecord(i, 1)
		content := newTestMailContent(i, 1)
		mgr.AddMail(ctx, record, content)
	}

	assert.Equal(t, 5, len(mgr.mailBoxIdIndex), "should have 5 mails")

	// Act - 移除前3封
	for i := int64(1); i <= 3; i++ {
		mgr.RemoveMail(ctx, i, nil)
	}

	// Assert
	assert.Equal(t, 2, len(mgr.mailBoxIdIndex), "should have 2 mails remaining")
	assert.Nil(t, mgr.GetMailRaw(1), "mail 1 should be removed")
	assert.Nil(t, mgr.GetMailRaw(2), "mail 2 should be removed")
	assert.Nil(t, mgr.GetMailRaw(3), "mail 3 should be removed")
	assert.NotNil(t, mgr.GetMailRaw(4), "mail 4 should still exist")
	assert.NotNil(t, mgr.GetMailRaw(5), "mail 5 should still exist")
	assert.Equal(t, 3, len(mgr.pendingRemoveList), "pending remove list should have 3 items")
}

// TestAddReadRemoveWorkflow 测试完整的邮件添加→读取→移除流程
// Scenario: 验证邮件生命周期的完整流程
func TestAddReadRemoveWorkflow(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()

	record := newTestMailRecord(1001, 1)
	content := newTestMailContent(1001, 1)

	// Step 1: 添加邮件
	ret := mgr.AddMail(ctx, record, content)
	assert.Equal(t, int32(0), ret)

	// Step 2: 读取邮件
	out := &public_protocol_pbdesc.DMailOperationResult{}
	ret = mgr.ReadMail(ctx, 1001, out, false)
	assert.Equal(t, int32(0), ret)
	assert.NotEqual(t, 0, out.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ))

	// Step 3: 验证邮件已读状态
	mail := mgr.GetMailRaw(1001)
	assert.NotNil(t, mail)
	assert.NotEqual(t, 0, mail.Record.GetStatus()&int32(public_protocol_common.EnMailStatusType_EN_MAIL_STATUS_READ))

	// Step 4: 移除邮件
	removeOut := &public_protocol_pbdesc.DMailOperationResult{}
	ret = mgr.RemoveMail(ctx, 1001, removeOut)
	assert.Equal(t, int32(0), ret)
	assert.Nil(t, mgr.GetMailRaw(1001), "mail should be removed")
}

// TestAddGlobalMailSuccess 测试添加全服邮件
// Scenario: 添加全服邮件应同时记录到 receivedGlobalMails 中
func TestAddGlobalMailSuccess(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	record.IsGlobalMail = true
	content := newTestMailContent(1001, 1)

	// Act
	ret := mgr.AddGlobalMail(ctx, record, content)

	// Assert
	assert.Equal(t, int32(0), ret, "should succeed")
	assert.Contains(t, mgr.receivedGlobalMails, int64(1001),
		"global mail should be recorded in receivedGlobalMails")
}

// TestAddGlobalMailNotGlobalFlag 测试添加非全服标记的邮件为全服邮件
// Scenario: IsGlobalMail=false 时应返回错误
func TestAddGlobalMailNotGlobalFlag(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record := newTestMailRecord(1001, 1)
	record.IsGlobalMail = false // 非全服邮件

	// Act
	ret := mgr.AddGlobalMail(ctx, record, nil)

	// Assert
	assert.Equal(t, int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM), ret,
		"should return error for non-global mail")
}

// TestAddGlobalMailDuplicate 测试重复添加全服邮件
// Scenario: 同一全服邮件重复添加应刷新已有记录
func TestAddGlobalMailDuplicate(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	ctx := newMockRpcContext()
	record1 := newTestMailRecord(1001, 1)
	record1.IsGlobalMail = true
	content1 := newTestMailContent(1001, 1)
	mgr.AddGlobalMail(ctx, record1, content1)

	record2 := newTestMailRecord(1001, 1)
	record2.IsGlobalMail = true
	record2.MinorType = 99

	// Act
	ret := mgr.AddGlobalMail(ctx, record2, nil)

	// Assert
	assert.Equal(t, int32(0), ret, "duplicate global mail should return 0")
	assert.Contains(t, mgr.receivedGlobalMails, int64(1001))
}

// ==================== needStartAsyncJobs 测试 ====================

// TestNeedStartAsyncJobsWithPendingRemove 测试有待移除邮件时需要启动异步任务
// Scenario: pendingRemoveList 不为空时应需要启动
func TestNeedStartAsyncJobsWithPendingRemove(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	mgr.pendingRemoveList[1001] = newTestMailRecord(1001, 1)
	mgr.isGlobalMailsMerged = true

	// Act
	result := mgr.needStartAsyncJobs()

	// Assert
	assert.True(t, result, "should need async jobs with pending removes")
}

// TestNeedStartAsyncJobsGlobalNotMerged 测试全服邮件未合并时需要启动异步任务
// Scenario: isGlobalMailsMerged=false 时应需要启动
func TestNeedStartAsyncJobsGlobalNotMerged(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	mgr.isGlobalMailsMerged = false

	// Act
	result := mgr.needStartAsyncJobs()

	// Assert
	assert.True(t, result, "should need async jobs when global mails not merged")
}

// TestNeedStartAsyncJobsProtected 测试保护期内不需要启动异步任务
// Scenario: 在保护期内应返回 false
func TestNeedStartAsyncJobsProtected(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	mgr.mailAsyncTaskProtectTimepoint = time.Now().Add(time.Hour)
	mgr.isGlobalMailsMerged = false

	// Act
	result := mgr.needStartAsyncJobs()

	// Assert
	assert.False(t, result, "should not need async jobs during protect period")
}

// TestNeedStartAsyncJobsNoWork 测试没有工作时不需要启动异步任务
// Scenario: 没有待处理的工作时应返回 false
func TestNeedStartAsyncJobsNoWork(t *testing.T) {
	// Arrange
	mgr := newTestUserMailManager()
	mgr.isGlobalMailsMerged = true

	// Act
	result := mgr.needStartAsyncJobs()

	// Assert
	assert.False(t, result, "should not need async jobs when no work to do")
}
