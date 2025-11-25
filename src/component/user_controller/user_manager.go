package atframework_component_user_controller

import (
	"context"
	"fmt"
	"reflect"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component-router"
	libatapp "github.com/atframework/libatapp-go"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type CreateUserCallback func(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl

var userManagerReflectType reflect.Type

type UserManager struct {
	libatapp.AppModuleBase
}

func init() {
	userManagerReflectType = reflect.TypeOf((*UserManager)(nil)).Elem()
	var _ libatapp.AppModuleImpl = (*UserManager)(nil)
}

func GetReflectTypeUserManager() reflect.Type {
	return userManagerReflectType
}

func (m *UserManager) GetReflectType() reflect.Type {
	return userManagerReflectType
}

func (m *UserManager) Init(parent context.Context) error {
	return nil
}

func CreateUserManager(app libatapp.AppImpl) *UserManager {
	ret := &UserManager{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
	}
	return ret
}

func (um *UserManager) Name() string {
	return "UserManager"
}

/////////////////////////////////////////////////////////////////////

func (um *UserManager) Find(ctx cd.RpcContext, zoneID uint32, userID uint64) UserImpl {
	routerObject := GetUserRouterManager(um.GetApp()).GetObject(ctx, router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   zoneID,
		ObjectID: userID,
	})
	if lu.IsNil(routerObject) {
		return nil
	}
	return routerObject.UserImpl
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

	if !lu.IsNil(checked) && !lu.Compare(cache.UserImpl, checked) {
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

func UserManagerFindUserAs[T UserImpl](ctx cd.RpcContext, app libatapp.AppImpl, zoneID uint32, userID uint64) T {
	um := libatapp.AtappGetModule[*UserManager](GetReflectTypeUserManager(), app)

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
	loginTb *private_protocol_pbdesc.DatabaseTableLogin,
	loginTbVersion uint64, loginCASVersion uint64,
	tryLockUserResource func(user T) cd.RpcResult,
	unlockUserResource func(user T),
) (T, cd.RpcResult) {
	um := libatapp.AtappGetModule[*UserManager](GetReflectTypeUserManager(), app)
	urm := GetUserRouterManager(app)

	var zero T
	if um == nil || zoneID <= 0 || userID <= 0 || loginTb == nil {
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
		loginTb:         loginTb,
		loginCASVersion: loginCASVersion,
		loginTbVersion:  loginTbVersion,
	})

	if result.IsError() || cache == nil {
		return zero, result
	}
	u = cache.UserImpl
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
	if u.GetLoginVersion() <= 0 {
		// 新用户初始化逻辑
		u.CreateInit(ctx, uint32(public_protocol_common.EnVersionType_EN_VERSION_DEFAULT))

		// 设置版本号
		u.GetLoginInfo().RouterVersion = 0
		// 更新Login Table版本号
		u.LoadLoginInfo(u.GetLoginInfo(), u.GetLoginInfo().RouterVersion)
		u.SetLoginCASVersion(loginCASVersion)

		result = cache.SaveObject(ctx, nil)
		if result.IsError() {
			return zero, result
		}
	} else {
		u.SetLoginCASVersion(loginCASVersion)
	}

	return convertRet, cd.CreateRpcResultOk()
}
