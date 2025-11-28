package atframework_component_user_controller

import (
	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component-router"
	libatapp "github.com/atframework/libatapp-go"
)

type UserRouterManager struct {
	*router.RouterManager[*UserRouterCache, *UserRouterPrivateData]
}

var createUserImplFn func(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) UserImpl = CreateUserImpl

func SetCreateUserImplFn(callback func(ctx cd.RpcContext, zoneId uint32, userId uint64, openId string, actorExecutor *cd.ActorExecutor) UserImpl) {
	if callback != nil {
		createUserImplFn = callback
	}
}

func InitUserRouterManager(app libatapp.AppImpl) {
	playerRouterManager := &UserRouterManager{}
	playerRouterManager.RouterManager = router.CreateRouterManager[*UserRouterCache, *UserRouterPrivateData](
		app,
		"UserRouterManager",
		public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER,
		func(ctx cd.RpcContext, key router.RouterObjectKey) *UserRouterCache {
			cache := &UserRouterCache{
				RouterObjectBase: router.CreateRouterObjectBase(ctx, key),
			}
			cache.obj = createUserImplFn(ctx, key.ZoneID, key.ObjectID, "", cache.GetActorExecutor())
			cache.RouterObjectBase.InitRouterObjectImpl(cache)
			return cache
		},
		playerRouterManager,
	)
	libatapp.AtappGetModule[*router.RouterManagerSet](router.GetReflectTypeRouterManagerSet(), app).RegisterManager(playerRouterManager)
}

func GetUserRouterManager(app libatapp.AppImpl) *UserRouterManager {
	return libatapp.AtappGetModule[*router.RouterManagerSet](router.GetReflectTypeRouterManagerSet(), app).
		GetManager(uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER)).(*UserRouterManager)
}

func (manager *UserRouterManager) OnRemoveObject(ctx cd.RpcContext, key router.RouterObjectKey, obj router.RouterObjectImpl, privData router.RouterPrivateData) {
	// 释放本地数据, 下线相关Session
	cache := obj.(*UserRouterCache).obj
	if !cache.CheckActorExecutor(ctx) {
		ctx.LogError("UserRouterManager OnRemoveObject ActorExecutor mismatch")
	}
	s := cache.GetUserSession()
	mgr := libatapp.AtappGetModule[*SessionManager](GetReflectTypeSessionManager(), ctx.GetApp())
	if !lu.IsNil(s) && mgr != nil {
		cache.UnbindSession(ctx, s)
		mgr.RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "Remove Object")
	}
}

func (manager *UserRouterManager) OnRemoveCache(ctx cd.RpcContext, key router.RouterObjectKey, obj router.RouterObjectImpl, privData router.RouterPrivateData) {
	// 释放本地数据, 下线相关Session
	cache := obj.(*UserRouterCache).obj
	if !cache.CheckActorExecutor(ctx) {
		ctx.LogError("UserRouterManager OnRemoveCache ActorExecutor mismatch")
	}
	s := cache.GetUserSession()
	mgr := libatapp.AtappGetModule[*SessionManager](GetReflectTypeSessionManager(), ctx.GetApp())
	if !lu.IsNil(s) && mgr != nil {
		cache.UnbindSession(ctx, s)
		mgr.RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "Remove Cache")
	}
}

func (manager *UserRouterManager) OnObjectRemoved(ctx cd.RpcContext, key router.RouterObjectKey, obj router.RouterObjectImpl, privData router.RouterPrivateData) {
	// 释放本地数据, 下线相关Session
	cache := obj.(*UserRouterCache).obj
	if !cache.CheckActorExecutor(ctx) {
		ctx.LogError("UserRouterManager OnObjectRemoved ActorExecutor mismatch")
	}
	s := cache.GetUserSession()
	mgr := libatapp.AtappGetModule[*SessionManager](GetReflectTypeSessionManager(), ctx.GetApp())
	if !lu.IsNil(s) && mgr != nil {
		cache.UnbindSession(ctx, s)
		mgr.RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "Remove Object")
	}
}

func (manager *UserRouterManager) OnCacheRemoved(ctx cd.RpcContext, key router.RouterObjectKey, obj router.RouterObjectImpl, privData router.RouterPrivateData) {
	// 释放本地数据, 下线相关Session
	cache := obj.(*UserRouterCache).obj
	if !cache.CheckActorExecutor(ctx) {
		ctx.LogError("UserRouterManager OnCacheRemoved ActorExecutor mismatch")
	}
	s := cache.GetUserSession()
	mgr := libatapp.AtappGetModule[*SessionManager](GetReflectTypeSessionManager(), ctx.GetApp())
	if !lu.IsNil(s) && mgr != nil {
		cache.UnbindSession(ctx, s)
		mgr.RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "Remove Cache")
	}
}

func (manager *UserRouterManager) OnPullObject(ctx cd.RpcContext, obj router.RouterObjectImpl, privData router.RouterPrivateData) {
}

func (manager *UserRouterManager) OnPullCache(ctx cd.RpcContext, cache router.RouterObjectImpl, privData router.RouterPrivateData) {
}
