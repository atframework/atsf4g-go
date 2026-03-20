package atframework_component_router

import (
	"log/slog"

	cd "github.com/atframework/atsf4g-go/component/dispatcher"
)

type RouterObjectImpl interface {
	// 不要直接调用 Pull Save, 由路由系统内层调用
	PullCache(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult // 存在默认实现
	PullObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	SaveObject(ctx cd.AwaitableContext, privateData RouterPrivateData) cd.RpcResult
	LogValue() slog.Value
	RouterObjectBaseImpl
}
