package atframework_component_user_controller

import (
	"fmt"
	"time"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type UserImpl interface {
	cd.TaskActionCSUser

	BindSession(self UserImpl, ctx *cd.RpcContext, session *Session)
	UnbindSession(self UserImpl, ctx *cd.RpcContext, session *Session)

	IsWriteable() bool

	InitFromDB(self UserImpl, ctx *cd.RpcContext, srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult
	DumpToDB(self UserImpl, ctx *cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult

	CreateInit(self UserImpl, ctx *cd.RpcContext, versionType uint32)
	LoginInit(self UserImpl, ctx *cd.RpcContext)

	OnLogin(self UserImpl, ctx *cd.RpcContext)
	OnLogout(self UserImpl, ctx *cd.RpcContext)
	OnSaved(self UserImpl, ctx *cd.RpcContext, version int64)
	OnUpdateSession(self UserImpl, ctx *cd.RpcContext, from *Session, to *Session)

	GetLoginInfo() *private_protocol_pbdesc.DatabaseTableLogin
	GetLoginVersion() uint64
	LoadLoginInfo(self UserImpl, loginTB *private_protocol_pbdesc.DatabaseTableLogin, version uint64)
}

type UserDirtyWrapper[T any] struct {
	value        T
	dirtyVersion int64
}

func (u *UserDirtyWrapper[T]) Get() *T {
	return &u.value
}

func (u *UserDirtyWrapper[T]) Mutable(version int64) *T {
	if version > u.dirtyVersion {
		u.dirtyVersion = version
	}

	return &u.value
}

func (u *UserDirtyWrapper[T]) IsDirty() bool {
	return u.dirtyVersion > 0
}

func (u *UserDirtyWrapper[T]) ClearDirty(version int64) {
	if version >= u.dirtyVersion {
		u.dirtyVersion = 0
	}
}

func (u *UserDirtyWrapper[T]) SetDirty(version int64) {
	if version > u.dirtyVersion {
		u.dirtyVersion = version
	}
}

type UserCache struct {
	zoneId uint32
	userId uint64
	openId string

	session *Session

	actorExecutor *cd.ActorExecutor

	loginInfo    *private_protocol_pbdesc.DatabaseTableLogin
	loginVersion uint64

	account_info_ UserDirtyWrapper[private_protocol_pbdesc.AccountInformation]
	user_data_    UserDirtyWrapper[private_protocol_pbdesc.UserData]
	user_options_ UserDirtyWrapper[private_protocol_pbdesc.UserOptions]
}

func CreateUserCache(zoneId uint32, userId uint64, openId string) UserCache {
	return UserCache{
		zoneId:        zoneId,
		userId:        userId,
		openId:        openId,
		actorExecutor: nil,
	}
}

func (u *UserCache) Init(actorInstance interface{}) {
	if u.actorExecutor == nil && actorInstance != nil {
		u.actorExecutor = cd.CreateActorExecutor(actorInstance)
	}
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

	if session == nil {
		u.UnbindSession(self, ctx, u.session)
		return
	}

	old_session := u.session

	// 覆盖旧绑定,必须先设置成员变量再触发关联绑定，以解决重入问题
	u.session = session
	session.BindUser(ctx, self)

	u.OnUpdateSession(self, ctx, old_session, session)

	if old_session != nil {
		old_session.UnbindUser(ctx, self)
	}
}

func (u *UserCache) UnbindSession(self UserImpl, ctx *cd.RpcContext, session *Session) {
	if u.session == nil {
		return
	}

	if session != nil && u.session != session {
		return
	}

	old_session := u.session
	u.session = nil

	u.OnUpdateSession(self, ctx, old_session, nil)

	if old_session != nil {
		old_session.UnbindUser(ctx, self)
	}

	// TODO: 触发登出保存
	if self.IsWriteable() {
		self.OnLogout(self, ctx)

		// TODO: 触发登出保存
	}
}

func (u *UserCache) IsWriteable() bool {
	return false
}

func (u *UserCache) RefreshLimit(_ctx *cd.RpcContext, _now time.Time) {
}

func (u *UserCache) InitFromDB(_self UserImpl, _ctx *cd.RpcContext, _srcTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (u *UserCache) DumpToDB(_self UserImpl, _ctx *cd.RpcContext, dstTb *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	if dstTb == nil {
		return cd.RpcResult{
			Error:        fmt.Errorf("dstTb should not be nil, zone_id: %d, user_id: %d", u.GetZoneId(), u.GetUserId()),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM),
		}
	}

	dstTb.AccountData = u.account_info_.Get()
	dstTb.UserData = u.user_data_.Get()
	dstTb.Options = u.user_options_.Get()

	return cd.CreateRpcResultOk()
}

func (u *UserCache) CreateInit(_self UserImpl, _ctx *cd.RpcContext, _versionType uint32) {
}

func (u *UserCache) LoginInit(_self UserImpl, _ctx *cd.RpcContext) {
}

func (u *UserCache) OnLogin(_self UserImpl, _ctx *cd.RpcContext) {
}

func (u *UserCache) OnLogout(_self UserImpl, _ctx *cd.RpcContext) {
}

func (u *UserCache) OnSaved(_self UserImpl, _ctx *cd.RpcContext, version int64) {
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
