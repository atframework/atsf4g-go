package atframework_component_dispatcher

import (
	"reflect"

	libatapp "github.com/atframework/libatapp-go"
)

var noMessageDispatcherReflectType reflect.Type

func init() {
	var _ libatapp.AppModuleImpl = (*NoMessageDispatcher)(nil)
	noMessageDispatcherReflectType = reflect.TypeOf((*NoMessageDispatcher)(nil)).Elem()
}

func GetReflectTypeNoMessageDispatcher() reflect.Type {
	return noMessageDispatcherReflectType
}

type NoMessageDispatcher struct {
	DispatcherBase
}

func CreateNoMessageDispatcher(owner libatapp.AppImpl) *NoMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &NoMessageDispatcher{
		DispatcherBase: CreateDispatcherBase(owner),
	}
	ret.DispatcherBase.impl = ret

	return ret
}

func (d *NoMessageDispatcher) Name() string { return "NoMessageDispatcher" }

func (m *NoMessageDispatcher) GetReflectType() reflect.Type {
	return noMessageDispatcherReflectType
}

func (d *NoMessageDispatcher) PickMessageRpcName(msg *DispatcherRawMessage) string {
	return ""
}

func (d *NoMessageDispatcher) PickMessageTaskId(msg *DispatcherRawMessage) uint64 {
	return 0
}
