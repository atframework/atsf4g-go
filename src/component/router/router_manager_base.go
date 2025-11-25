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

	InnerMutableCache(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData) (RouterObjectImpl, cd.RpcResult)
	InnerMutableObject(ctx cd.AwaitableContext, key RouterObjectKey, privData RouterPrivateData) (RouterObjectImpl, cd.RpcResult)
	InnerRemoveCache(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData) cd.RpcResult
	InnerRemoveObject(ctx cd.AwaitableContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData) cd.RpcResult

	//////////////////////////// 待实现接口 ////////////////////////////
	OnRemoveObject(ctx cd.RpcContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData)
	OnRemoveCache(ctx cd.RpcContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData)
	OnObjectRemoved(ctx cd.RpcContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData)
	OnCacheRemoved(ctx cd.RpcContext, key RouterObjectKey, obj RouterObjectImpl, privData RouterPrivateData)
	OnPullObject(ctx cd.RpcContext, obj RouterObjectImpl, privData RouterPrivateData)
	OnPullCache(ctx cd.RpcContext, cache RouterObjectImpl, privData RouterPrivateData)

	////////////////////////////
	IsAutoMutableObject() bool
	IsAutoMutableCache() bool
	GetDefaultRouterServerID(RouterObjectKey) uint64
	PullOnlineServer(ctx cd.AwaitableContext, key RouterObjectKey) (svrId uint64, svrVer uint64, result cd.RpcResult)
	GetBaseCache(key RouterObjectKey) RouterObjectImpl
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
