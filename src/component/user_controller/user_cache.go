package atframework_component_user_controller

import (
	"fmt"
	"runtime"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	libatapp "github.com/atframework/libatapp-go"
)

const (
	UserDataCurrentVersion uint64 = 1
)

type UserImpl interface {
	cd.TaskActionCSUser

	BindSession(self UserImpl, ctx *cd.RpcContext, session *Session)
	UnbindSession(self UserImpl, ctx *cd.RpcContext, session *Session)
	AllocSessionSequence() uint64

	IsWriteable() bool

	InitFromDB(self UserImpl, ctx *cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult
	DumpToDB(self UserImpl, ctx *cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult

	CreateInit(self UserImpl, ctx *cd.RpcContext, versionType uint32)
	LoginInit(self UserImpl, ctx *cd.RpcContext)

	OnLogin(self UserImpl, ctx *cd.RpcContext)
	OnLogout(self UserImpl, ctx *cd.RpcContext)
	OnSaved(self UserImpl, ctx *cd.RpcContext, version uint64)
	OnUpdateSession(self UserImpl, ctx *cd.RpcContext, from *Session, to *Session)

	GetLoginInfo() *private_protocol_pbdesc.DatabaseTableLogin
	GetLoginVersion() uint64
	LoadLoginInfo(self UserImpl, loginTB *private_protocol_pbdesc.DatabaseTableLogin, version uint64)

	SyncClientDirtyCache()
	CleanupClientDirtyCache()
}

type UserDirtyWrapper[T any] struct {
	value        T
	dirtyVersion uint64
}

func (u *UserDirtyWrapper[T]) Get() *T {
	return &u.value
}

func (u *UserDirtyWrapper[T]) Mutable(version uint64) *T {
	if version > u.dirtyVersion {
		u.dirtyVersion = version
	}

	return &u.value
}

func (u *UserDirtyWrapper[T]) IsDirty() bool {
	return u.dirtyVersion > 0
}

func (u *UserDirtyWrapper[T]) ClearDirty(version uint64) {
	if version >= u.dirtyVersion {
		u.dirtyVersion = 0
	}
}

func (u *UserDirtyWrapper[T]) SetDirty(version uint64) {
	if version > u.dirtyVersion {
		u.dirtyVersion = version
	}
}

type UserCache struct {
	zoneId uint32
	userId uint64
	openId string

	session         *Session
	sessionSequence uint64

	actorExecutor *cd.ActorExecutor

	loginInfo    *private_protocol_pbdesc.DatabaseTableLogin
	loginVersion uint64

	dataVersion uint64

	account_info_ UserDirtyWrapper[private_protocol_pbdesc.AccountInformation]
	user_data_    UserDirtyWrapper[private_protocol_pbdesc.UserData]
	user_options_ UserDirtyWrapper[private_protocol_pbdesc.UserOptions]

	cs_actor_log_writer libatapp.LogWriter
}

func CreateUserCache(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) UserCache {
	writer, _ := libatapp.NewlogBufferedRotatingWriter("../log",
		openId, 1*1024*1024, 3, time.Second*3, false, true)
	runtime.SetFinalizer(writer, func(writer *libatapp.LogBufferedRotatingWriter) {
		writer.Close()
	})
	return UserCache{
		zoneId:        zoneId,
		userId:        userId,
		openId:        openId,
		actorExecutor: nil,
		loginInfo:     nil,
		loginVersion:  0,
		dataVersion:   0,
		account_info_: UserDirtyWrapper[private_protocol_pbdesc.AccountInformation]{
			value: private_protocol_pbdesc.AccountInformation{
				Profile: &public_protocol_pbdesc.DUserProfile{
					OpenId: openId,
					UserId: userId,
				},
			},
			dirtyVersion: 0,
		},
		cs_actor_log_writer: writer,
	}
}

func (u *UserCache) Init(actorInstance interface{}) {
	if lu.IsNil(u.actorExecutor) && !lu.IsNil(actorInstance) {
		u.actorExecutor = cd.CreateActorExecutor(actorInstance)
	}
	u.sessionSequence = 99
}

func (u *UserCache) GetOpenId() string {
	return u.openId
}

func (u *UserCache) GetUserId() uint64 {
	return u.userId
}

func (u *UserCache) GetZoneId() uint32 {
	return u.zoneId
}

func (u *UserCache) GetSession() cd.TaskActionCSSession {
	return u.session
}

func (u *UserCache) GetUserSession() *Session {
	return u.session
}

func (u *UserCache) GetActorExecutor() *cd.ActorExecutor {
	return u.actorExecutor
}

func (u *UserCache) SendAllSyncData() error {
	return nil
}

func (u *UserCache) BindSession(self UserImpl, ctx *cd.RpcContext, session *Session) {
	if u.session == session {
		return
	}

	if lu.IsNil(session) {
		u.UnbindSession(self, ctx, u.session)
		return
	}

	old_session := u.session

	// 覆盖旧绑定,必须先设置成员变量再触发关联绑定，以解决重入问题
	u.session = session
	session.BindUser(ctx, self)

	u.OnUpdateSession(self, ctx, old_session, session)

	if !lu.IsNil(old_session) {
		old_session.UnbindUser(ctx, self)
	}
}

func (u *UserCache) UnbindSession(self UserImpl, ctx *cd.RpcContext, session *Session) {
	if u.session == nil {
		return
	}

	if !lu.IsNil(session) && u.session != session {
		return
	}

	old_session := u.session
	u.session = nil

	u.OnUpdateSession(self, ctx, old_session, nil)

	if !lu.IsNil(old_session) {
		old_session.UnbindUser(ctx, self)
	}

	// TODO: 触发登出保存
	if self.IsWriteable() {
		self.OnLogout(self, ctx)

		// TODO: 触发登出保存
	}
}

func (u *UserCache) AllocSessionSequence() uint64 {
	u.sessionSequence++
	return u.sessionSequence
}

func (u *UserCache) IsWriteable() bool {
	return false
}

func (u *UserCache) RefreshLimit(_ctx *cd.RpcContext, _now time.Time) {
}

func (u *UserCache) InitFromDB(_self UserImpl, _ctx *cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	u.dataVersion = srcTb.DataVersion
	if srcTb.GetUserData().GetSessionSequence() > u.sessionSequence {
		u.sessionSequence = srcTb.GetUserData().GetSessionSequence()
	}

	return cd.CreateRpcResultOk()
}

func (u *UserCache) DumpToDB(_self UserImpl, _ctx *cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	if dstTb == nil {
		return cd.RpcResult{
			Error:        fmt.Errorf("dstTb should not be nil, zone_id: %d, user_id: %d", u.GetZoneId(), u.GetUserId()),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM),
		}
	}

	dstTb.OpenId = u.GetOpenId()
	dstTb.UserId = u.GetUserId()
	dstTb.ZoneId = u.GetZoneId()
	// always use current version
	dstTb.DataVersion = UserDataCurrentVersion

	lu.Mutable(&dstTb.UserData).SessionSequence = u.sessionSequence

	dstTb.AccountData = u.account_info_.Get()
	dstTb.UserData = u.user_data_.Get()
	dstTb.Options = u.user_options_.Get()

	return cd.CreateRpcResultOk()
}

func (u *UserCache) CreateInit(_self UserImpl, _ctx *cd.RpcContext, versionType uint32) {
	u.MutableAccountInfo().VersionType = versionType
}

func (u *UserCache) LoginInit(_self UserImpl, ctx *cd.RpcContext) {
	u.updateLoginData(ctx)
}

func (u *UserCache) OnLogin(_self UserImpl, _ctx *cd.RpcContext) {
}

func (u *UserCache) OnLogout(_self UserImpl, ctx *cd.RpcContext) {
	loginInfo := lu.Mutable(&u.loginInfo)

	nowSec := ctx.GetNow().Unix()
	loginInfo.BusinessLogoutTime = nowSec
}

func (u *UserCache) OnSaved(_self UserImpl, _ctx *cd.RpcContext, version uint64) {
	u.account_info_.ClearDirty(version)
	u.user_data_.ClearDirty(version)
	u.user_options_.ClearDirty(version)
}

func (u *UserCache) OnUpdateSession(_self UserImpl, ctx *cd.RpcContext, from *Session, to *Session) {
}

func (u *UserCache) GetLoginInfo() *private_protocol_pbdesc.DatabaseTableLogin {
	if u.loginInfo == nil {
		u.loginInfo = &private_protocol_pbdesc.DatabaseTableLogin{}
		u.loginInfo.UserId = u.userId
		u.loginInfo.ZoneId = u.zoneId
		u.loginInfo.OpenId = u.openId

		u.loginInfo.Account = &private_protocol_pbdesc.AccountInformation{}
		u.loginInfo.Account.Profile.OpenId = u.openId
		u.loginInfo.Account.Profile.UserId = u.userId

		u.loginVersion = 0
	}

	return u.loginInfo
}

func (u *UserCache) GetLoginVersion() uint64 {
	return u.loginVersion
}

func (u *UserCache) LoadLoginInfo(_self UserImpl, info *private_protocol_pbdesc.DatabaseTableLogin, version uint64) {
	if info == nil {
		return
	}

	u.loginInfo = info
	u.loginVersion = version
}

func (u *UserCache) SyncClientDirtyCache() {
}

func (u *UserCache) CleanupClientDirtyCache() {
}

func (u *UserCache) GetCurrentDbDataVersion() uint64 {
	return u.loginInfo.RouterVersion
}

func (u *UserCache) GetAccountInfo() *private_protocol_pbdesc.AccountInformation {
	return u.account_info_.Get()
}

func (u *UserCache) MutableAccountInfo() *private_protocol_pbdesc.AccountInformation {
	return u.account_info_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserData() *private_protocol_pbdesc.UserData {
	return u.user_data_.Get()
}

func (u *UserCache) MutableUserData() *private_protocol_pbdesc.UserData {
	return u.user_data_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserOptions() *private_protocol_pbdesc.UserOptions {
	return u.user_options_.Get()
}

func (u *UserCache) MutableUserOptions() *private_protocol_pbdesc.UserOptions {
	return u.user_options_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	return lu.Mutable(&u.loginInfo).GetLastLogin().GetClientInfo()
}

func (u *UserCache) MutableClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	return lu.Mutable(&lu.Mutable(&lu.Mutable(&u.loginInfo).LastLogin).ClientInfo)
}

func (u *UserCache) GetCsActorLogWriter() libatapp.LogWriter {
	return u.cs_actor_log_writer
}

func (u *UserCache) updateLoginData(ctx *cd.RpcContext) {
	if u.loginInfo == nil {
		return
	}

	// Patch login table
	nowSec := ctx.GetNow().Unix()
	// TODO: 有效期来自配置
	u.loginInfo.LoginCodeExpired = nowSec + int64(20*60)

	if u.loginInfo.GetBusinessRegisterTime() <= 0 {
		u.loginInfo.BusinessRegisterTime = nowSec
	}
	u.loginInfo.BusinessLoginTime = nowSec

	// 默认昵称
	if u.loginInfo.GetAccount().GetProfile().GetNickName() == "" {
		lu.Mutable(&lu.Mutable(&u.loginInfo.Account).Profile).NickName = fmt.Sprintf("User-%v-%v", u.GetZoneId(), u.GetUserId())
	}

	u.loginInfo.StatLoginSuccessTimes++
	u.loginInfo.StatLoginTotalTimes++

	// Copy to user table
	*u.MutableAccountInfo() = *u.loginInfo.GetAccount()
}
