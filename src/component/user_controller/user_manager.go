package atframework_component_user_controller

import (
	"context"
	"fmt"
	"sync"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	logical_time "github.com/atframework/atsf4g-go/component/logical_time"
	operation_support_system "github.com/atframework/atsf4g-go/component/operation_support_system"
	private_protocol_log "github.com/atframework/atsf4g-go/component/protocol/private/log/protocol/log"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component/router"
	libatapp "github.com/atframework/libatapp-go"

	cd "github.com/atframework/atsf4g-go/component/dispatcher"
)

type CreateUserCallback func(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl

type UserManager struct {
	libatapp.AppModuleBase

	lastReportOnline int64
	onlineUserLock   sync.Mutex
	onlineUser       map[uint64]UserImpl
}

func init() {
	var _ libatapp.AppModuleImpl = (*UserManager)(nil)
}

func (m *UserManager) Init(parent context.Context) error {
	return nil
}

func (m *UserManager) Tick(parent context.Context) bool {
	now := logical_time.GetSysNow().Unix()
	if now > m.lastReportOnline+60 {
		m.lastReportOnline = now
		onlineCount := int32(m.OnlineCount())
		{
			log := private_protocol_log.OperationSupportSystemLog{}
			log.MutableLog().MutableOnlineFlow().OnlineCount = onlineCount
			operation_support_system.SendOssLog(m.GetApp(), &log)
		}
		{
			log := private_protocol_log.MonitorLog{}
			log.MutableLog().MutableServerOnlineCountFlow().OnlineCount = onlineCount
			operation_support_system.SendMonLog(m.GetApp(), &log)
		}
	}
	return false
}

func CreateUserManager(app libatapp.AppImpl) *UserManager {
	ret := &UserManager{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
		onlineUser:    make(map[uint64]UserImpl),
	}
	return ret
}

func (um *UserManager) Name() string {
	return "UserManager"
}

/////////////////////////////////////////////////////////////////////

func (um *UserManager) Online(user UserImpl) {
	um.onlineUserLock.Lock()
	defer um.onlineUserLock.Unlock()
	um.onlineUser[user.GetUserId()] = user
}

func (um *UserManager) Offline(user UserImpl) {
	um.onlineUserLock.Lock()
	defer um.onlineUserLock.Unlock()
	currentUser := um.onlineUser[user.GetUserId()]
	if !lu.IsNil(currentUser) && currentUser == user {
		delete(um.onlineUser, user.GetUserId())
	}
}

func (um *UserManager) OnlineCount() int {
	return len(um.onlineUser)
}

func (um *UserManager) Find(ctx cd.RpcContext, zoneID uint32, userID uint64) UserImpl {
	routerObject := GetUserRouterManager(um.GetApp()).GetObject(ctx, router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   zoneID,
		ObjectID: userID,
	})
	if lu.IsNil(routerObject) {
		return nil
	}
	return routerObject.obj
}

func (um *UserManager) Remove(ctx cd.AwaitableContext, zoneID uint32, userID uint64, checked UserImpl, forceKickoff bool) cd.RpcResult {
	key := router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   zoneID,
		ObjectID: userID,
	}
	cache := GetUserRouterManager(um.GetApp()).GetCache(key)
	if lu.IsNil(cache) {
		return cd.CreateRpcResultOk()
	}

	if !lu.IsNil(checked) && cache.obj != checked {
		// 不匹配当前缓存 尝试移除Session
		if checked.GetUserSession() != nil {
			checked.UnbindSession(ctx, nil)
			libatapp.AtappGetModule[*SessionManager](ctx.GetApp()).RemoveSession(ctx,
				checked.GetUserSession().GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "closed by server on remove user")
		}
		return cd.CreateRpcResultOk()
	}

	if !forceKickoff && !cache.IsWritable() {
		return cd.CreateRpcResultOk()
	}

	if forceKickoff {
		return GetUserRouterManager(um.GetApp()).RemoveCache(ctx, router.RouterObjectKey{
			TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
			ZoneID:   zoneID,
			ObjectID: userID,
		}, cache, nil)
	} else {
		return GetUserRouterManager(um.GetApp()).RemoveObject(ctx, router.RouterObjectKey{
			TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
			ZoneID:   zoneID,
			ObjectID: userID,
		}, nil, nil)
	}
}

func (um *UserManager) Save(ctx cd.AwaitableContext, checkUser UserImpl) cd.RpcResult {
	if lu.IsNil(checkUser) {
		return cd.CreateRpcResultError(fmt.Errorf("user is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}
	cache := GetUserRouterManager(um.GetApp()).GetCache(router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   checkUser.GetZoneId(),
		ObjectID: checkUser.GetUserId(),
	})
	if lu.IsNil(cache) {
		return cd.CreateRpcResultError(fmt.Errorf("user cache not exists"), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND)
	}
	if !cache.IsWritable() {
		return cd.CreateRpcResultError(fmt.Errorf("user cache not writable"), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_WRITABLE)
	}
	if cache.GetUserImpl() != checkUser {
		return cd.CreateRpcResultError(fmt.Errorf("user mismatch"), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND)
	}
	return cache.Save(ctx)
}

func UserManagerFindUserAs[T UserImpl](ctx cd.RpcContext, app libatapp.AppImpl, zoneID uint32, userID uint64) T {
	um := libatapp.AtappGetModule[*UserManager](app)

	userImpl := um.Find(ctx, zoneID, userID)
	if lu.IsNil(userImpl) {
		var zero T
		return zero
	}
	casted, ok := userImpl.(T)
	if !ok {
		var zero T
		return zero
	}

	return casted
}

func UserManagerCreateUserAs[T UserImpl](app libatapp.AppImpl, ctx cd.AwaitableContext,
	zoneID uint32, userID uint64, openID string,
	loginLockTb *private_protocol_pbdesc.DatabaseTableLoginLock,
	loginLockTbCASVersion uint64,
	fillBasicInfo func(user T),
	tryLockUserResource func(user T) cd.RpcResult,
	unlockUserResource func(user T),
) (T, cd.RpcResult) {
	um := libatapp.AtappGetModule[*UserManager](app)
	urm := GetUserRouterManager(app)

	var zero T
	if um == nil || zoneID <= 0 || userID <= 0 || loginLockTb == nil {
		return zero, cd.CreateRpcResultError(fmt.Errorf("invalid param"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	u := um.Find(ctx, zoneID, userID)
	if !lu.IsNil(u) {
		ctx.LogError("already exists, can not create again", "zoneID", zoneID, "userID", userID)
		return zero, cd.CreateRpcResultError(fmt.Errorf("already exists, can not create again"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	cache, result := urm.MutableObject(ctx, router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   zoneID,
		ObjectID: userID,
	}, &UserRouterPrivateData{
		loginLockTb:     loginLockTb,
		loginCASVersion: loginLockTbCASVersion,
		openId:          openID,
	})

	if result.IsError() || cache == nil {
		return zero, result
	}
	u = cache.obj
	convertRet, ok := u.(T)
	if !ok {
		return zero, cd.CreateRpcResultError(fmt.Errorf("user type mismatch, zone_id: %d, user_id: %d, type: %T", zoneID, userID, u), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 路由系统外逻辑
	defer func() {
		if !lu.IsNil(u) && unlockUserResource != nil {
			unlockUserResource(u.(T))
		}
	}()
	if tryLockUserResource != nil {
		result := tryLockUserResource(u.(T))
		if result.IsError() {
			unlockUserResource = nil
			return zero, result
		}
	}

	// 创建初始化
	if !u.HasCreateInit() {
		fillBasicInfo(convertRet)
		// 新用户初始化逻辑
		u.CreateInit(ctx, uint32(public_protocol_common.EnVersionType_EN_VERSION_DEFAULT))
		// 立刻保存一次
		result = cache.SaveObject(ctx, nil)
		if result.IsError() {
			return zero, result
		}
	}

	return convertRet, cd.CreateRpcResultOk()
}
