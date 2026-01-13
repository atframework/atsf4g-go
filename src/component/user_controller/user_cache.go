package atframework_component_user_controller

import (
	"fmt"
	"log/slog"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"

	operation_support_system "github.com/atframework/atsf4g-go/component-operation-support-system"
	private_protocol_log "github.com/atframework/atsf4g-go/component-protocol-private/log/protocol/log"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	router "github.com/atframework/atsf4g-go/component-router"
	libatapp "github.com/atframework/libatapp-go"
)

const (
	UserDataCurrentVersion uint64 = 1
)

type UserImpl interface {
	cd.TaskActionCSUser

	CanBeWriteable() bool
	IsWriteable() bool

	GetUserSession() *Session
	BindSession(ctx cd.RpcContext, session *Session)
	UnbindSession(ctx cd.RpcContext, session *Session)
	AllocSessionSequence() uint64

	GetActorExecutor() *cd.ActorExecutor
	CheckActorExecutor(ctx cd.RpcContext) bool

	// 拉取DB时注册OpenId
	InitOpenId(openId string)
	// 登录成功时更新登录数据
	UpdateLoginData(ctx cd.RpcContext)

	InitFromDB(ctx cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult
	DumpToDB(ctx cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult

	CreateInit(ctx cd.RpcContext, versionType uint32)
	LoginInit(ctx cd.RpcContext)

	OnLogin(ctx cd.RpcContext)
	OnLogout(ctx cd.RpcContext)
	OnSaved(ctx cd.RpcContext, routerVersion uint64)
	OnUpdateSession(ctx cd.RpcContext, from *Session, to *Session)

	GetLoginLockInfo() *private_protocol_pbdesc.DatabaseTableLoginLock
	LoadLoginLockInfo(loginTB *private_protocol_pbdesc.DatabaseTableLoginLock)
	HasCreateInit() bool

	GetUserCASVersion() uint64
	SetUserCASVersion(CASVersion uint64)
	GetLoginLockCASVersion() uint64
	SetLoginLockCASVersion(CASVersion uint64)

	SendUserOssLog(ctx cd.RpcContext, ossLog *private_protocol_log.OperationSupportSystemLog)
}

func init() {
	var _ UserImpl = (*UserCache)(nil)
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

	loginLockInfo *private_protocol_pbdesc.DatabaseTableLoginLock

	// Basic Data
	dataVersion uint64
	loginData   UserDirtyWrapper[private_protocol_pbdesc.LoginData]
	accountInfo UserDirtyWrapper[private_protocol_pbdesc.AccountInformation]
	userData    UserDirtyWrapper[private_protocol_pbdesc.UserData]
	// Basic Data

	csActorLogWriter libatapp.LogWriter

	hasCreateInit bool

	loginLockCASVersion uint64
	userCASVersion      uint64
}

func CreateUserCache(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) (cache UserCache) {
	// 由路由系统创建可能没有OpenId
	var writer *libatapp.LogBufferedRotatingWriter
	if config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetEnableActorLog() {
		writer, _ = libatapp.NewLogBufferedRotatingWriter(ctx, fmt.Sprintf("%s/%%F/%d-%d.%%N.log", config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetServer().GetLogPath(), zoneId, userId),
			fmt.Sprintf("%s/%%F/%d-%d.log", config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetServer().GetLogPath(), zoneId, userId), config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetActorLogSize(),
			uint32(config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetActorLogRotate()),
			config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetActorAutoFlush().AsDuration(), 0)
	}
	cache = UserCache{
		zoneId:           zoneId,
		userId:           userId,
		openId:           openId,
		csActorLogWriter: writer,
	}
	cache.actorExecutor = actorExecutor
	cache.actorExecutor.Instance = &cache
	cache.Impl = &cache
	cache.Init()
	return
}

func CreateUserImpl(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) UserImpl {
	cache := CreateUserCache(ctx, zoneId, userId, openId, actorExecutor)
	return &cache
}

func (u *UserCache) Init() {
	if u.actorExecutor == nil {
		u.actorExecutor = cd.CreateActorExecutor(u)
	}
	u.sessionSequence = 99
}

func (obj *UserCache) LogAttr() []slog.Attr {
	if obj == nil {
		return nil
	}
	return []slog.Attr{
		slog.Uint64("zone_id", uint64(obj.zoneId)),
		slog.Uint64("user_id", obj.userId),
	}
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

func (u *UserCache) CheckActorExecutor(ctx cd.RpcContext) bool {
	if u == nil {
		return false
	}

	return u.actorExecutor.CheckActorExecutor(ctx)
}

func (u *UserCache) OnSendResponse(ctx cd.RpcContext) error {
	// 刷新路由系统
	cache := GetUserRouterManager(ctx.GetApp()).GetCache(router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   u.GetZoneId(),
		ObjectID: u.GetUserId(),
	})
	if cache != nil && cache.obj == u.Impl {
		if ctx.GetAction().GetActorExecutor() != nil && cache.CheckActorExecutor(ctx) {
			cache.RefreshVisitTime(ctx)
		}
	}
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

	outputLog := fmt.Sprintf("%s >>>>>>>>>>>>>>>>>>>> Bind Session: %d", ctx.GetSysNow().Format("2006-01-02 15:04:05.000"), session.GetSessionId())
	logWriter := u.GetCsActorLogWriter()
	if logWriter != nil {
		session.FlushPendingActorLog(logWriter)
		fmt.Fprint(logWriter, outputLog)
	}
	ctx.LogDebug(outputLog)

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

	outputLog := fmt.Sprintf("%s >>>>>>>>>>>>>>>>>>>> Unbind Session: %d", ctx.GetSysNow().Format("2006-01-02 15:04:05.000"), u.session.GetSessionId())
	logWriter := u.GetCsActorLogWriter()
	if logWriter != nil {
		fmt.Fprint(logWriter, outputLog)
	}
	ctx.LogDebug(outputLog)

	old_session := u.session
	u.session = nil

	u.OnUpdateSession(ctx, old_session, nil)

	if !lu.IsNil(old_session) {
		old_session.UnbindUser(ctx, u.Impl)
	}

	if u.Impl.IsWriteable() {
		u.Impl.OnLogout(ctx)
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

	if srcTb.GetLoginData() != nil {
		u.MutableUserLogin().Merge(srcTb.GetLoginData())
	}

	u.hasCreateInit = srcTb.GetCreateInit()

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

	dstTb.AccountData = u.accountInfo.Get()
	dstTb.UserData = u.userData.Get()
	dstTb.LoginData = u.loginData.Get()
	dstTb.MutableUserData().SessionSequence = u.sessionSequence
	dstTb.CreateInit = u.hasCreateInit

	return cd.CreateRpcResultOk()
}

func (u *UserCache) CreateInit(_ctx cd.RpcContext, versionType uint32) {
	if u == nil {
		return
	}
	u.hasCreateInit = true
	u.MutableAccountInfo().VersionType = versionType
}

func (u *UserCache) HasCreateInit() bool {
	if u == nil {
		return false
	}

	return u.hasCreateInit
}

func (u *UserCache) LoginInit(ctx cd.RpcContext) {
	if u == nil {
		return
	}
	u.UpdateLoginData(ctx)
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

	// 设置登出时间
	u.loginData.Mutable(u.GetCurrentDbDataVersion()).BusinessLogoutTime = ctx.GetSysNow().Unix()
	u.csActorLogWriter.Flush()
}

func (u *UserCache) OnSaved(_ctx cd.RpcContext, routerVersion uint64) {
	if u == nil {
		return
	}

	u.loginData.ClearDirty(routerVersion)
	u.accountInfo.ClearDirty(routerVersion)
	u.userData.ClearDirty(routerVersion)
}

func (u *UserCache) OnUpdateSession(_ctx cd.RpcContext, from *Session, to *Session) {
}

func (u *UserCache) GetLoginLockInfo() *private_protocol_pbdesc.DatabaseTableLoginLock {
	if u == nil {
		return nil
	}
	return u.loginLockInfo
}

func (u *UserCache) LoadLoginLockInfo(info *private_protocol_pbdesc.DatabaseTableLoginLock) {
	if u == nil || info == nil {
		return
	}
	u.loginLockInfo = info
}

func (u *UserCache) SyncClientDirtyCache() {
}

func (u *UserCache) CleanupClientDirtyCache() {
}

func (u *UserCache) GetCurrentDbDataVersion() uint64 {
	if u == nil {
		return 0
	}

	return u.loginLockInfo.GetRouterVersion()
}

func (u *UserCache) GetAccountInfo() *private_protocol_pbdesc.AccountInformation {
	if u == nil {
		return nil
	}

	return u.accountInfo.Get()
}

func (u *UserCache) MutableAccountInfo() *private_protocol_pbdesc.AccountInformation {
	if u == nil {
		return nil
	}

	return u.accountInfo.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserData() *private_protocol_pbdesc.UserData {
	if u == nil {
		return nil
	}

	return u.userData.Get()
}

func (u *UserCache) MutableUserData() *private_protocol_pbdesc.UserData {
	if u == nil {
		return &private_protocol_pbdesc.UserData{}
	}

	return u.userData.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetUserLogin() *private_protocol_pbdesc.LoginData {
	if u == nil {
		return nil
	}

	return u.loginData.Get()
}

func (u *UserCache) MutableUserLogin() *private_protocol_pbdesc.LoginData {
	if u == nil {
		return &private_protocol_pbdesc.LoginData{}
	}

	return u.loginData.Mutable(u.GetCurrentDbDataVersion())
}

func (u *UserCache) GetClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	if u == nil {
		return nil
	}

	return u.loginData.Get().GetLastLoginRecord().GetClientInfo()
}

func (u *UserCache) MutableClientInfo() *public_protocol_pbdesc.DClientDeviceInfo {
	if u == nil {
		return &public_protocol_pbdesc.DClientDeviceInfo{}
	}

	return u.MutableUserLogin().MutableLastLoginRecord().MutableClientInfo()
}

func (u *UserCache) GetCsActorLogWriter() libatapp.LogWriter {
	if u == nil {
		return nil
	}

	return u.csActorLogWriter
}

func (u *UserCache) GetLoginLockCASVersion() uint64 {
	if u == nil {
		return 0
	}

	return u.loginLockCASVersion
}

func (u *UserCache) SetLoginLockCASVersion(version uint64) {
	if u == nil {
		return
	}

	u.loginLockCASVersion = version
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

	nowSec := ctx.GetSysNow().Unix()
	// 更新登录数据
	if u.IsNewUser() {
		u.loginData.Mutable(u.GetCurrentDbDataVersion()).BusinessRegisterTime = nowSec
	}
	u.loginData.Mutable(u.GetCurrentDbDataVersion()).BusinessLoginTime = nowSec

	u.loginData.Mutable(u.GetCurrentDbDataVersion()).StatLoginSuccessTimes++
	u.loginData.Mutable(u.GetCurrentDbDataVersion()).StatLoginTotalTimes++
}

func (u *UserCache) IsNewUser() bool {
	if u == nil {
		return false
	}
	return u.loginData.Get().GetBusinessRegisterTime() <= 0
}

func (u *UserCache) SendUserOssLog(ctx cd.RpcContext, ossLog *private_protocol_log.OperationSupportSystemLog) {
	if u == nil {
		return
	}
	ossLog.MutableBasic().UserId = u.GetUserId()
	ossLog.MutableBasic().ZoneId = u.GetZoneId()
	operation_support_system.SendOssLog(ctx.GetApp(), ossLog)
}
