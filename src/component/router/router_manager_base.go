package atframework_component_router

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type RouterManagerBase struct {
	impl      RouterManagerBaseImpl
	name      string
	typeID    uint32
	isClosing bool
}

type RouterManagerBaseImpl interface {
	Name() string
	GetTypeID() uint32
	IsClosing() bool
	OnStop()

	//////////////////////////// 待实现接口 ////////////////////////////
	MutableCache(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData, guard *IoTaskGuard) (RouterObject, cd.RpcResult)
	MutableObject(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData, guard *IoTaskGuard) (RouterObject, cd.RpcResult)
	RemoveCache(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObject, privData RouterPrivateData, guard *IoTaskGuard) cd.RpcResult
	RemoveObject(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObject, privData RouterPrivateData, guard *IoTaskGuard) cd.RpcResult

	////////////////////////////
	IsAutoMutableObject() bool
	IsAutoMutableCache() bool
	GetDefaultRouterServerID(RouterObjectKey) uint64
	PullOnlineServer(ctx cd.AwaitableContext, key RouterObjectKey) (svrId uint64, svrVer uint64, result cd.RpcResult)
	GetBaseCache(key RouterObjectKey) RouterObject
	Size() int
}

func CreateRouterManagerBase(name string, typeID uint32) RouterManagerBase {
	return RouterManagerBase{
		name:   name,
		typeID: typeID,
	}
}

func (base *RouterManagerBase) Name() string {
	return base.name
}

func (base *RouterManagerBase) GetTypeID() uint32 {
	return base.typeID
}

func (base *RouterManagerBase) IsClosing() bool {
	return base.isClosing
}

func (base *RouterManagerBase) OnStop() {
	base.isClosing = true
}

func (base *RouterManagerBase) IsAutoMutableObject() bool {
	return false
}

func (base *RouterManagerBase) IsAutoMutableCache() bool {
	return true
}

func (base *RouterManagerBase) GetDefaultRouterServerID(_key RouterObjectKey) uint64 {
	return 0
}

func (base *RouterManagerBase) PullOnlineServer(ctx cd.AwaitableContext, key RouterObjectKey) (uint64, uint64, cd.RpcResult) {
	return 0, 0, cd.CreateRpcResultOk()
}

func (base *RouterManagerBase) MutableCacheWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData) (RouterObject, cd.RpcResult) {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return base.impl.MutableCache(ctx, key, privData, &guard)
}

func (base *RouterManagerBase) MutableObjectWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData) (RouterObject, cd.RpcResult) {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return base.impl.MutableObject(ctx, key, privData, &guard)
}

func (base *RouterManagerBase) RemoveCacheWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObject, privData RouterPrivateData) cd.RpcResult {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return base.impl.RemoveCache(ctx, key, obj, privData, &guard)
}

func (base *RouterManagerBase) RemoveObjectWithoutGuard(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObject, privData RouterPrivateData) cd.RpcResult {
	guard := IoTaskGuard{}
	defer guard.ResumeAwaitTask(ctx)
	return base.impl.RemoveObject(ctx, key, obj, privData, &guard)
}
