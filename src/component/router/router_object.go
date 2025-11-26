package atframework_component_router

import (
	"log/slog"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type RouterObjectImpl interface {
	PullCache(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult // 存在默认实现
	PullObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	SaveObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	LogValue() slog.Value
	GetActorExecutor() *cd.ActorExecutor
	CheckActorExecutor(ctx cd.RpcContext) bool
	RouterObjectBaseImpl
}
