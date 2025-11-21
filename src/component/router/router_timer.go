package atframework_component_router

import (
	"container/list"
)

// RouterTimer 路由定时器
type RouterTimer struct {
	ObjWatcher RouterObject // 路由对象
	TypeID     uint32

	Timeout       int64  // 超时时间戳
	TimerSequence uint64 // 定时器序列号
	TimerList     *list.List
	TimerElement  *list.Element
}

// TimerSet 定时器集合
type TimerSet struct {
	DefaultTimerList *list.List
	FastTimerList    *list.List
}
