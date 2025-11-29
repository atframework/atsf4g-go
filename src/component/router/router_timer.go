package atframework_component_router

import (
	"container/list"
	"sync"
)

type noCopy struct{}

// RouterTimer 路由定时器
type RouterTimer struct {
	_          noCopy
	ObjWatcher RouterObjectImpl // 路由对象
	TypeID     uint32

	Timeout       int64  // 超时时间戳
	TimerSequence uint64 // 定时器序列号
	TimerList     *TimerList
	TimerElement  *TimerElement
}

type TimerElement struct {
	_       noCopy
	element *list.Element
}

type TimerList struct {
	_    noCopy
	list *list.List

	removeLock    sync.Mutex
	pendingRemove []*TimerElement

	insertLock    sync.Mutex
	pendingInsert *list.List
}

func NewTimerList() *TimerList {
	return &TimerList{
		list:          list.New(),
		pendingInsert: list.New(),
	}
}

func (tl *TimerList) DoPending() {
	tl.removeLock.Lock()
	tl.insertLock.Lock()
	defer tl.removeLock.Unlock()
	defer tl.insertLock.Unlock()

	// 先插入再删除
	for e := tl.pendingInsert.Front(); e != nil; e = e.Next() {
		timer, _ := e.Value.(*RouterTimer)
		if timer == nil {
			continue
		}
		elem := tl.list.PushBack(timer)
		timer.TimerElement.element = elem
	}
	tl.pendingInsert.Init()

	// 删除
	for _, elem := range tl.pendingRemove {
		tl.list.Remove(elem.element)
	}
	tl.pendingRemove = tl.pendingRemove[:0]
}

func (tl *TimerList) Remove(elem *TimerElement) {
	tl.removeLock.Lock()
	defer tl.removeLock.Unlock()
	tl.pendingRemove = append(tl.pendingRemove, elem)
}

func (tl *TimerList) PushBack(timer *RouterTimer) *TimerElement {
	tl.insertLock.Lock()
	defer tl.insertLock.Unlock()
	return &TimerElement{
		element: tl.pendingInsert.PushBack(timer),
	}
}

// TimerSet 定时器集合
type TimerSet struct {
	DefaultTimerList *TimerList
	FastTimerList    *TimerList
}
