package atframework_component_dispatcher

import (
	libatapp "github.com/atframework/libatapp-go"
)

type NoMessageDispatcher struct {
	DispatcherBase
}

func CreateNoMessageDispatcher(owner libatapp.AppImpl) *NoMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &NoMessageDispatcher{
		DispatcherBase: CreateDispatcherBase(owner),
	}

	return ret
}

func (d *NoMessageDispatcher) Name() string { return "NoMessageDispatcher" }
