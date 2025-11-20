package atframework_component_router

import (
	"fmt"
	"sync"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type RouterManager[T RouterObject, PrivData RouterPrivateData] struct {
	RouterManagerBase

	cacheFactory RouterCacheFactory[T]

	caches   map[RouterObjectKey]T
	cachesMu sync.RWMutex

	onRemoveCache   RouterRemoveHandler[T, PrivData]
	onCacheRemoved  RouterRemoveHandler[T, PrivData]
	onRemoveObject  RouterRemoveHandler[T, PrivData]
	onObjectRemoved RouterRemoveHandler[T, PrivData]
	onPullCache     RouterPullHandler[T, PrivData]
	onPullObject    RouterPullHandler[T, PrivData]
}

type RouterCacheFactory[T RouterObject] func(key RouterObjectKey) T

type RouterRemoveHandler[T RouterObject, PrivData RouterPrivateData] func(ctx cd.AwaitableContext, manager *RouterManager[T, PrivData], key RouterObjectKey, cache T, priv PrivData) cd.RpcResult

type RouterPullHandler[T RouterObject, PrivData RouterPrivateData] func(ctx cd.AwaitableContext, manager *RouterManager[T, PrivData], cache T, priv PrivData) cd.RpcResult

func CreateRouterManager[T RouterObject, PrivData RouterPrivateData](name string, typeID uint32, factory RouterCacheFactory[T]) *RouterManager[T, PrivData] {
	manager := &RouterManager[T, PrivData]{
		RouterManagerBase: CreateRouterManagerBase(name, typeID),
		cacheFactory:      factory,
		caches:            make(map[RouterObjectKey]T),
	}
	manager.RouterManagerBase.impl = manager
	return manager
}

func (manager *RouterManager[T, PrivData]) OnStop() {
	manager.RouterManagerBase.OnStop()
	for _, v := range manager.caches {
		v.UnsetTimerRef()
	}
}

func (manager *RouterManager[T, PrivData]) GetBaseCache(key RouterObjectKey) RouterObject {
	manager.cachesMu.RLock()
	defer manager.cachesMu.RUnlock()
	if cache, ok := manager.caches[key]; ok {
		return cache
	}
	return nil
}

func (manager *RouterManager[T, PrivData]) GetCache(key RouterObjectKey) T {
	manager.cachesMu.RLock()
	defer manager.cachesMu.RUnlock()
	return manager.caches[key]
}

func (manager *RouterManager[T, PrivData]) GetObject(ctx cd.RpcContext, key RouterObjectKey) T {
	manager.cachesMu.RLock()
	defer manager.cachesMu.RUnlock()
	obj := manager.caches[key]
	if !lu.IsNil(obj) && obj.IsWritable() {
		return obj
	}
	var zero T
	return zero
}

func (manager *RouterManager[T, PrivData]) Size() int {
	return len(manager.caches)
}

func (manager *RouterManager[T, PrivData]) MutableCache(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData, guard *IoTaskGuard) (RouterObject, cd.RpcResult) {
	return manager.MutableCacheWithGuard(ctx, key, privData.(PrivData), guard)
}

func (manager *RouterManager[T, PrivData]) MutableObject(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData, guard *IoTaskGuard) (RouterObject, cd.RpcResult) {
	return manager.MutableObjectWithGuard(ctx, key, privData.(PrivData), guard)
}

func (manager *RouterManager[T, PrivData]) RemoveCache(ctx cd.AwaitableContext, key RouterObjectKey, cache RouterObject, privData RouterPrivateData, guard *IoTaskGuard) cd.RpcResult {
	return manager.RemoveCacheWithGuard(ctx, key, cache, privData.(PrivData), guard)
}

func (manager *RouterManager[T, PrivData]) RemoveObject(ctx cd.AwaitableContext, key RouterObjectKey, cache RouterObject, privData RouterPrivateData, guard *IoTaskGuard) cd.RpcResult {
	return manager.RemoveObjectWithGuard(ctx, key, cache, privData.(PrivData), guard)
}

func (manager *RouterManager[T, PrivData]) MutableCacheWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData PrivData) (T, cd.RpcResult) {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return manager.MutableCacheWithGuard(ctx, key, privData, &guard)
}

func (manager *RouterManager[T, PrivData]) MutableObjectWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData PrivData) (T, cd.RpcResult) {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return manager.MutableObjectWithGuard(ctx, key, privData, &guard)
}

func (manager *RouterManager[T, PrivData]) RemoveCacheWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) cd.RpcResult {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return manager.RemoveCacheWithGuard(ctx, key, cache, privData, &guard)
}

func (manager *RouterManager[T, PrivData]) RemoveObjectWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) cd.RpcResult {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return manager.RemoveObjectWithGuard(ctx, key, cache, privData, &guard)
}

func (manager *RouterManager[T, PrivData]) MutableCacheWithGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData PrivData, guard *IoTaskGuard) (T, cd.RpcResult) {
	leftTTL := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetRetryMaxTtl()
	if leftTTL <= 0 {
		leftTTL = 1
	}
	for ; leftTTL > 0; leftTTL-- {
		cache := manager.ensureCache(key)
		if lu.IsNil(cache) {
			var zero T
			return zero, cd.CreateRpcResultError(fmt.Errorf("create cache failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		// 先等待IO任务完成，完成后可能在其他任务里已经拉取完毕了。
		result := guard.Take(ctx, cache)
		if result.IsError() {
			var zero T
			return zero, result
		}

		if cache.IsCacheAvailable(ctx) {
			cache.UnsetFlag(FlagSchedRemoveCache)
			return cache, cd.CreateRpcResultOk()
		}

		result = cache.InternalPullCache(ctx, guard, privData)
		if result.IsError() {
			code := public_protocol_pbdesc.EnErrorCode(result.GetResponseCode())
			if manager.shouldAbortCacheRetry(code) {
				var zero T
				return zero, result
			}
			if code == public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_EAGAIN {
				manager.waitCacheRetry(ctx)
			}
			continue
		}

		manager.invokePullCache(ctx, cache, privData)
		return cache, cd.CreateRpcResultOk()
	}
	var zero T
	return zero, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_TTL_EXTEND)
}

func (manager *RouterManager[T, PrivData]) MutableObjectWithGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData PrivData, guard *IoTaskGuard) (T, cd.RpcResult) {
	leftTTL := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetRouter().GetRetryMaxTtl()
	if leftTTL <= 0 {
		leftTTL = 1
	}
	for ; leftTTL > 0; leftTTL-- {
		cache := manager.ensureCache(key)
		if lu.IsNil(cache) {
			var zero T
			return zero, cd.CreateRpcResultError(fmt.Errorf("create cache failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		// 先等待IO任务完成，完成后可能在其他任务里已经拉取完毕了。
		result := guard.Take(ctx, cache)
		if result.IsError() {
			var zero T
			return zero, result
		}
		if cache.IsObjectAvailable() {
			// 触发拉取实体并命中cache时要取消移除缓存和降级的计划任务
			cache.UnsetFlag(FlagForceRemoveObject)
			cache.UnsetFlag(FlagSchedRemoveObject)
			cache.UnsetFlag(FlagSchedRemoveCache)
			return cache, cd.CreateRpcResultOk()
		}

		// 如果处于正在关闭的状态，则不允许创建新的实体，只能访问缓存
		if manager.IsClosing() {
			var zero T
			return zero, cd.CreateRpcResultError(fmt.Errorf("router system closing"), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_CLOSING)
		}

		result = cache.InternalPullObject(ctx, guard, privData)
		if result.IsError() {
			code := public_protocol_pbdesc.EnErrorCode(result.GetResponseCode())
			if manager.shouldAbortObjectRetry(code) {
				var zero T
				return zero, result
			}
			if code == public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_EAGAIN {
				manager.waitObjectRetry(ctx)
			}
			continue
		}

		if !cache.CheckFlag(FlagCacheRemoved) {
			manager.invokePullObject(ctx, cache, privData)
			return cache, cd.CreateRpcResultOk()
		}
	}
	var zero T
	return zero, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_TTL_EXTEND)
}

func (manager *RouterManager[T, PrivData]) RemoveCacheWithGuard(ctx cd.AwaitableContext, key RouterObjectKey, cache RouterObject, privData PrivData, guard *IoTaskGuard) cd.RpcResult {
	var managerCache T
	{
		if !lu.IsNil(cache) {
			managerCache = manager.GetCache(cache.GetKey())
		} else {
			managerCache = manager.GetCache(key)
		}

		if lu.IsNil(managerCache) {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND)
		}

		if !lu.IsNil(cache) && !lu.Compare(managerCache, cache) {
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND)
		}

		if managerCache.IsWritable() {
			if result := manager.RemoveObjectWithGuard(ctx, key, managerCache, privData, guard); result.IsError() {
				return result
			}
		}
	}

	result := guard.Take(ctx, managerCache)
	if result.IsError() {
		return result
	}

	removeCacheFlag := NewFlagGuard(managerCache.GetRouterObjectBase(), FlagRemovingCache)
	defer removeCacheFlag.Release()

	triggerEvt := !managerCache.CheckFlag(FlagCacheRemoved)
	if triggerEvt {
		manager.invokeRemoveCache(ctx, key, managerCache, privData)
		managerCache.SetFlag(FlagCacheRemoved)
	}

	manager.cachesMu.Lock()
	if current, ok := manager.caches[managerCache.GetKey()]; ok && lu.Compare(current, managerCache) {
		delete(manager.caches, managerCache.GetKey())
	}
	manager.cachesMu.Unlock()

	if triggerEvt {
		manager.invokeCacheRemoved(ctx, key, managerCache, privData)
	}

	return cd.CreateRpcResultOk()
}

func (manager *RouterManager[T, PrivData]) RemoveObjectWithGuard(ctx cd.AwaitableContext, key RouterObjectKey, cache RouterObject, privData PrivData, guard *IoTaskGuard) cd.RpcResult {
	var managerCache T
	if lu.IsNil(cache) {
		managerCache = manager.GetCache(key)
	} else {
		managerCache = cache.(T)
	}

	if lu.IsNil(managerCache) {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND)
	}

	removeObjectFlag := NewFlagGuard(managerCache.GetRouterObjectBase(), FlagRemovingObject)
	defer removeObjectFlag.Release()

	result := guard.Take(ctx, managerCache)
	if result.IsError() {
		return result
	}

	manager.invokeRemoveObject(ctx, key, managerCache, privData)

	result = managerCache.RemoveObject(ctx, 0, guard, privData)
	if result.IsError() {
		return result
	}

	manager.invokeObjectRemoved(ctx, key, managerCache, privData)
	return cd.CreateRpcResultOk()
}

func (manager *RouterManager[T, PrivData]) RenewCache(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) (T, cd.RpcResult) {
	if !lu.IsNil(cache) && !cache.CheckFlag(FlagCacheRemoved) {
		return cache, cd.CreateRpcResultOk()
	}

	return manager.MutableCacheWithoutGuard(ctx, key, privData)
}

func (manager *RouterManager[T, PrivData]) ensureCache(key RouterObjectKey) T {
	manager.cachesMu.RLock()
	cache, ok := manager.caches[key]
	manager.cachesMu.RUnlock()
	if ok && !lu.IsNil(cache) {
		return cache
	}

	if manager.cacheFactory == nil {
		var zero T
		return zero
	}

	newCache := manager.cacheFactory(key)
	manager.cachesMu.Lock()
	if _, exists := manager.caches[key]; exists {
		newCache = manager.caches[key]
	} else {
		manager.caches[key] = newCache
	}
	manager.cachesMu.Unlock()

	return newCache
}

func (manager *RouterManager[T, PrivData]) waitCacheRetry(ctx cd.AwaitableContext) {
	// TODO 等待Retry 时间后唤醒
	//       time_t wait_interval_ms =
	//       static_cast<time_t>(logic_config::me()->get_cfg_router().cache_retry_interval().seconds() * 1000 +
	//                           logic_config::me()->get_cfg_router().cache_retry_interval().nanos() / 1000000);
	//   if (wait_interval_ms <= 0) {
	//     wait_interval_ms = 512;
	//   }

	//   RPC_AWAIT_IGNORE_RESULT(rpc::wait(ctx, std::chrono::milliseconds{util::random_engine::random_between(
	//                                              wait_interval_ms / 2, wait_interval_ms)}));
}

func (manager *RouterManager[T, PrivData]) waitObjectRetry(ctx cd.AwaitableContext) {
	// TODO
}

func (manager *RouterManager[T, PrivData]) shouldAbortCacheRetry(code public_protocol_pbdesc.EnErrorCode) bool {
	switch code {
	case public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND:
		return true
	default:
		return false
	}
}

func (manager *RouterManager[T, PrivData]) shouldAbortObjectRetry(code public_protocol_pbdesc.EnErrorCode) bool {
	switch code {
	case public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_FOUND,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_NOT_WRITABLE,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND,
		public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_BUSSINESS_VERSION_DENY:
		return true
	default:
		return false
	}
}

/////////////////////////////////// 回调接口 ///////////////////////////////////

func (manager *RouterManager[T, PrivData]) SetOnRemoveCache(fn RouterRemoveHandler[T, PrivData]) {
	manager.onRemoveCache = fn
}

func (manager *RouterManager[T, PrivData]) SetOnCacheRemoved(fn RouterRemoveHandler[T, PrivData]) {
	manager.onCacheRemoved = fn
}

func (manager *RouterManager[T, PrivData]) SetOnRemoveObject(fn RouterRemoveHandler[T, PrivData]) {
	manager.onRemoveObject = fn
}

func (manager *RouterManager[T, PrivData]) SetOnObjectRemoved(fn RouterRemoveHandler[T, PrivData]) {
	manager.onObjectRemoved = fn
}

func (manager *RouterManager[T, PrivData]) SetOnPullCache(fn RouterPullHandler[T, PrivData]) {
	manager.onPullCache = fn
}

func (manager *RouterManager[T, PrivData]) SetOnPullObject(fn RouterPullHandler[T, PrivData]) {
	manager.onPullObject = fn
}

func (manager *RouterManager[T, PrivData]) invokeRemoveCache(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) {
	if manager.onRemoveCache == nil {
		return
	}
	_ = manager.onRemoveCache(ctx, manager, key, cache, privData)
}

func (manager *RouterManager[T, PrivData]) invokeCacheRemoved(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) {
	if manager.onCacheRemoved == nil {
		return
	}
	_ = manager.onCacheRemoved(ctx, manager, key, cache, privData)
}

func (manager *RouterManager[T, PrivData]) invokeRemoveObject(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) {
	if manager.onRemoveObject == nil {
		return
	}
	_ = manager.onRemoveObject(ctx, manager, key, cache, privData)
}

func (manager *RouterManager[T, PrivData]) invokeObjectRemoved(ctx cd.AwaitableContext, key RouterObjectKey, cache T, privData PrivData) {
	if manager.onObjectRemoved == nil {
		return
	}
	_ = manager.onObjectRemoved(ctx, manager, key, cache, privData)
}

func (manager *RouterManager[T, PrivData]) invokePullCache(ctx cd.AwaitableContext, cache T, privData PrivData) {
	if manager.onPullCache == nil {
		return
	}
	_ = manager.onPullCache(ctx, manager, cache, privData)
}

func (manager *RouterManager[T, PrivData]) invokePullObject(ctx cd.AwaitableContext, cache T, privData PrivData) {
	if manager.onPullObject == nil {
		return
	}
	_ = manager.onPullObject(ctx, manager, cache, privData)
}
