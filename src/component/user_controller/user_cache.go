package atframework_component_user_controller

import (
	"fmt"
	"runtime"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"

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

	CanBeWriteable() bool
	GetUserSession() *Session
	BindSession(ctx cd.RpcContext, session *Session)
	UnbindSession(ctx cd.RpcContext, session *Session)
	AllocSessionSequence() uint64

	// 拉取DB时注册OpenId
	InitOpenId(openId string)

	// 路由登录成功时更新登录数据
	UpdateLoginData(ctx cd.RpcContext)

	IsWriteable() bool

	InitFromDB(ctx cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult
	DumpToDB(ctx cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult

	CreateInit(ctx cd.RpcContext, versionType uint32)
	LoginInit(ctx cd.RpcContext)

	OnLogin(ctx cd.RpcContext)
	OnLogout(ctx cd.RpcContext)
	OnSaved(ctx cd.RpcContext, version uint64)
	OnUpdateSession(ctx cd.RpcContext, from *Session, to *Session)

	GetLoginInfo() *private_protocol_pbdesc.DatabaseTableLogin
	GetLoginVersion() uint64
	LoadLoginInfo(loginTB *private_protocol_pbdesc.DatabaseTableLogin, version uint64)

	GetUserCASVersion() uint64
	SetUserCASVersion(version uint64)
	GetLoginCASVersion() uint64
	SetLoginCASVersion(version uint64)
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
	Impl UserImpl

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

	loginCASVersion uint64
	userCASVersion  uint64
}

func CreateUserCache(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string) (cache UserCache) {
	var writer *libatapp.LogBufferedRotatingWriter
	if config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetUser().GetEnableSessionActorLog() {
		writer, _ = libatapp.NewlogBufferedRotatingWriter(ctx, config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetServer().GetLogPath(),
			openId, config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorLogSize(),
			uint32(config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorLogRotate()),
			config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorAutoFlush().AsDuration(), false, true)
		runtime.SetFinalizer(writer, func(writer *libatapp.LogBufferedRotatingWriter) {
			writer.Close()
		})
	}
	cache = UserCache{
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
	cache.Impl = &cache
	return
}

func CreateUserImpl(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl {
	cache := CreateUserCache(ctx, zoneId, userId, openId)
	return &cache
}

func (u *UserCache) Init(actorInstance interface{}) {
	if lu.IsNil(u.actorExecutor) && !lu.IsNil(actorInstance) {
		u.actorExecutor = cd.CreateActorExecutor(actorInstance)
	}
	u.sessionSequence = 99
}

func (u *UserCache) GetOpenId() string {
	if u == nil {
		return ""
	}

	return u.openId
}

func (u *UserCache) GetUserId() uint64 {
	if u == nil {
		return 0
	}

	return u.userId
}

func (u *UserCache) GetZoneId() uint32 {
	if u == nil {
		return 0
	}

	return u.zoneId
}

func (u *UserCache) InitOpenId(openId string) {
	u.openId = openId
	u.account_info_.Mutable(0).MutableProfile().OpenId = openId
}

func (u *UserCache) CanBeWriteable() bool {
	return false
}

func (u *UserCache) GetSession() cd.TaskActionCSSession {
	if u == nil {
		return nil
	}

	if u.session == nil {
		return nil
	}

	return u.session
}

func (u *UserCache) GetUserSession() *Session {
	if u == nil {
		return nil
	}

	if u.session == nil {
		return nil
	}

	return u.session
}

func (u *UserCache) GetActorExecutor() *cd.ActorExecutor {
	if u == nil {
		return nil
	}

	return u.actorExecutor
}

func (u *UserCache) SendAllSyncData(_ctx cd.RpcContext) error {
	return nil
}

func (u *UserCache) BindSession(ctx cd.RpcContext, session *Session) {
	if u == nil {
		return
	}

	if u.session == session {
		return
	}

	if lu.IsNil(session) {
		u.UnbindSession(ctx, u.session)
		return
	}

	old_session := u.session

	// 覆盖旧绑定,必须先设置成员变量再触发关联绑定，以解决重入问题
	u.session = session
	session.BindUser(ctx, u.Impl)

	logWriter := u.GetCsActorLogWriter()
	if logWriter != nil {
		for _, log := range session.pendingCsLog {
			fmt.Fprint(logWriter, log)
		}
		session.pendingCsLog = nil
		fmt.Fprintf(logWriter, "%s >>>>>>>>>>>>>>>>>>>> Bind Session: %d \n", ctx.GetSysNow().Format("2006-01-02 15:04:05.000"), session.GetSessionId())
	}

	u.OnUpdateSession(ctx, old_session, session)

	if !lu.IsNil(old_session) {
		old_session.UnbindUser(ctx, u.Impl)
	}
}

func (u *UserCache) UnbindSession(ctx cd.RpcContext, session *Session) {
	if u == nil {
		return
	}

	if u.session == nil {
		return
	}

	if !lu.IsNil(session) && u.session != session {
		return
	}

	logWriter := u.GetCsActorLogWriter()
	if logWriter != nil {
		fmt.Fprintf(logWriter, "%s >>>>>>>>>>>>>>>>>>>> Unbind Session: %d \n", ctx.GetSysNow().Format("2006-01-02 15:04:05.000"), u.session.GetSessionId())
	}

	old_session := u.session
	u.session = nil

	u.OnUpdateSession(ctx, old_session, nil)

	if !lu.IsNil(old_session) {
		old_session.UnbindUser(ctx, u.Impl)
	}

	// TODO: 触发登出保存
	if u.Impl.IsWriteable() {
		u.OnLogout(ctx)

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

func (u *UserCache) RefreshLimit(_ctx cd.RpcContext, _now time.Time) {
}

func (u *UserCache) InitFromDB(_ctx cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	u.dataVersion = srcTb.DataVersion
	if srcTb.GetUserData().GetSessionSequence() > u.sessionSequence {
		u.sessionSequence = srcTb.GetUserData().GetSessionSequence()
	}

	if srcTb.GetAccountData() != nil {
		u.MutableAccountInfo().Merge(srcTb.GetAccountData())
	}

	if srcTb.GetUserData() != nil {
		u.MutableUserData().Merge(srcTb.GetUserData())
	}

	if srcTb.GetOptions() != nil {
		u.MutableUserOptions().Merge(srcTb.GetOptions())
	}

	// TODO: 数据版本升级 u.dataVersion -> UserDataCurrentVersion

	return cd.CreateRpcResultOk()
}

func (u *UserCache) DumpToDB(_ctx cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
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

func (u *UserCache) CreateInit(_ctx cd.RpcContext, versionType uint32) {
	if u == nil {
		return
	}

	u.MutableAccountInfo().VersionType = versionType
}

func (u *UserCache) LoginInit(ctx cd.RpcContext) {
	if u == nil {
		return
	}
}

func (u *UserCache) OnLogin(__ctx cd.RpcContext) {
	if u == nil {
		return
	}
}

func (u *UserCache) OnLogout(ctx cd.RpcContext) {
	if u == nil {
		return
	}

	loginInfo := lu.Mutable(&u.loginInfo)

	nowSec := ctx.GetSysNow().Unix()
	loginInfo.BusinessLogoutTime = nowSec
}

func (u *UserCache) OnSaved(_ctx cd.RpcContext, version uint64) {
	if u == nil {
		return
	}

	u.account_info_.ClearDirty(version)
	u.user_data_.ClearDirty(version)
	u.user_options_.ClearDirty(version)
}

func (u *UserCache) OnUpdateSession(_ctx cd.RpcContext, from *Session, to *Session) {
}

func (u *UserCache) GetLoginInfo() *private_protocol_pbdesc.DatabaseTableLogin {
	if u == nil {
		return nil
	}

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
	if u == nil {
		return 0
	}

	return u.loginVersion
}

func (u *UserCache) LoadLoginInfo(info *private_protocol_pbdesc.DatabaseTableLogin, version uint64) {
	if u == nil || info == nil {
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
	if u == nil {
		return 0
	}

	return u.loginInfo.RouterVersion
}

func (u *UserCache) GetAccountInfo() *private_protocol_pbdesc.AccountInformation {
	if u == nil {
		return nil
	}

	return u.account_info_.Get()
}

func (u *UserCache) MutableAccountInfo() *private_protocol_pbdesc.AccountInformation {
	if u == nil {
		return nil
	}

	return u.account_info_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserData() *private_protocol_pbdesc.UserData {
	if u == nil {
		return nil
	}

	return u.user_data_.Get()
}

func (u *UserCache) MutableUserData() *private_protocol_pbdesc.UserData {
	if u == nil {
		return &private_protocol_pbdesc.UserData{}
	}

	return u.user_data_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserOptions() *private_protocol_pbdesc.UserOptions {
	if u == nil {
		return nil
	}

	return u.user_options_.Get()
}

func (u *UserCache) MutableUserOptions() *private_protocol_pbdesc.UserOptions {
	if u == nil {
		return &private_protocol_pbdesc.UserOptions{}
	}

	return u.user_options_.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	if u == nil {
		return nil
	}

	return lu.Mutable(&u.loginInfo).GetLastLogin().GetClientInfo()
}

func (u *UserCache) MutableClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	if u == nil {
		return &public_protocol_pbdesc.DClientDeviceInfo{}
	}

	return lu.Mutable(&lu.Mutable(&lu.Mutable(&u.loginInfo).LastLogin).ClientInfo)
}

func (u *UserCache) GetCsActorLogWriter() libatapp.LogWriter {
	if lu.IsNil(u.cs_actor_log_writer) {
		return nil
	}
	return u.cs_actor_log_writer
}

func (u *UserCache) GetLoginCASVersion() uint64 {
	if u == nil {
		return 0
	}

	return u.loginCASVersion
}

func (u *UserCache) SetLoginCASVersion(version uint64) {
	if u == nil {
		return
	}

	u.loginCASVersion = version
}

func (u *UserCache) GetUserCASVersion() uint64 {
	if u == nil {
		return 0
	}

	return u.userCASVersion
}

func (u *UserCache) SetUserCASVersion(version uint64) {
	if u == nil {
		return
	}

	u.userCASVersion = version
}

func (u *UserCache) UpdateLoginData(ctx cd.RpcContext) {
	if u == nil {
		return
	}

	if u.loginInfo == nil {
		return
	}

	// Patch login table
	nowSec := ctx.GetSysNow().Unix()

	u.loginInfo.LoginCodeExpired = nowSec +
		config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetLoginCodeValidSec().GetSeconds()

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
