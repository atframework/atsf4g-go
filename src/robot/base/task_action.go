package atsf4g_go_robot_protocol_base

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
)

type TaskActionImpl interface {
	AwaitTask(TaskActionImpl) error
	InitOnFinish(func())
	GetTaskId() uint64
	BeforeYield()
	AfterYield()
	Finish()
	InitTaskId(uint64)
	GetTimeoutDuration() time.Duration
	InitTimeoutTimer(*time.Timer)
	TimeoutKill()

	HookRun()
}

func AwaitTask(other TaskActionImpl) error {
	AwaitChannel := make(chan struct{}, 1)
	other.InitOnFinish(func() {
		AwaitChannel <- struct{}{}
	})
	<-AwaitChannel
	return nil
}

const (
	TaskActionAwaitTypeNone = iota
	TaskActionAwaitTypeNormal
	TaskActionAwaitTypeRPC
)

type TaskActionAwaitData struct {
	WaitingType uint32
	WaitingId   uint64
}

type TaskActionResumeData struct {
	Err  error
	Data interface{}
}

type TaskActionBase struct {
	Impl   TaskActionImpl
	TaskId uint64

	awaitData       TaskActionAwaitData
	AwaitChannel    chan *TaskActionResumeData
	timeoutDuration time.Duration
	Timeout         *time.Timer

	finishLock sync.Mutex
	finished   bool
	kill       atomic.Bool
	onFinish   []func()
}

func NewTaskActionBase(timeoutDuration time.Duration) *TaskActionBase {
	t := &TaskActionBase{
		timeoutDuration: timeoutDuration,
		AwaitChannel:    make(chan *TaskActionResumeData),
	}
	return t
}

func (t *TaskActionBase) Yield(awaitData TaskActionAwaitData) *TaskActionResumeData {
	if t.kill.Load() {
		return &TaskActionResumeData{
			Err: fmt.Errorf("task action killed"),
		}
	}
	t.awaitData = awaitData
	t.Impl.BeforeYield()
	ret := <-t.AwaitChannel
	t.awaitData.WaitingId = 0
	t.awaitData.WaitingType = 0
	t.Impl.AfterYield()
	return ret
}

func (t *TaskActionBase) Resume(awaitData *TaskActionAwaitData, resumeData *TaskActionResumeData) {
	if t.awaitData.WaitingId == awaitData.WaitingId && t.awaitData.WaitingType == awaitData.WaitingType {
		t.AwaitChannel <- resumeData
	}
}

func (t *TaskActionBase) TimeoutKill() {
	t.kill.Store(true)
	if t.awaitData.WaitingId != 0 && t.awaitData.WaitingType != TaskActionAwaitTypeNone {
		t.AwaitChannel <- &TaskActionResumeData{
			Err: fmt.Errorf("sys timeout"),
		}
	}
}

func (t *TaskActionBase) Finish() {
	t.Timeout.Stop()
	t.finishLock.Lock()
	defer t.finishLock.Unlock()
	t.finished = true
	for _, fn := range t.onFinish {
		fn()
	}
}

func (t *TaskActionBase) InitOnFinish(fn func()) {
	t.finishLock.Lock()
	defer t.finishLock.Unlock()
	if t.finished {
		fn()
		return
	}
	t.onFinish = append(t.onFinish, fn)
}

func (t *TaskActionBase) GetTaskId() uint64 {
	return t.TaskId
}

func (t *TaskActionBase) AwaitTask(other TaskActionImpl) error {
	if lu.IsNil(other) {
		return fmt.Errorf("task nil")
	}
	other.InitOnFinish(func() {
		t.Resume(&TaskActionAwaitData{
			WaitingType: TaskActionAwaitTypeNormal,
			WaitingId:   other.GetTaskId(),
		}, &TaskActionResumeData{
			Err:  nil,
			Data: nil,
		})
	})
	t.Yield(TaskActionAwaitData{
		WaitingType: TaskActionAwaitTypeNormal,
		WaitingId:   other.GetTaskId(),
	})
	return nil
}

func (t *TaskActionBase) BeforeYield() {
	// do nothing
}

func (t *TaskActionBase) AfterYield() {
	// do nothing
}

func (t *TaskActionBase) InitTaskId(id uint64) {
	t.TaskId = id
}

func (t *TaskActionBase) GetTimeoutDuration() time.Duration {
	return t.timeoutDuration
}

func (t *TaskActionBase) InitTimeoutTimer(timer *time.Timer) {
	t.Timeout = timer
}

type TaskActionManager struct {
	taskIdMap   sync.Map
	taskIdAlloc atomic.Uint64
}

func NewTaskActionManager() *TaskActionManager {
	ret := &TaskActionManager{}
	ret.taskIdAlloc.Store(
		uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()))
	return ret
}

func (m *TaskActionManager) allocTaskId() uint64 {
	id := m.taskIdAlloc.Add(1)
	return id
}

func (m *TaskActionManager) RunTaskAction(taskAction TaskActionImpl) {
	taskAction.InitTaskId(m.allocTaskId())
	m.taskIdMap.Store(taskAction.GetTaskId(), taskAction)

	if taskAction.GetTimeoutDuration() > 0 {
		timeoutTimer := time.AfterFunc(taskAction.GetTimeoutDuration(), func() {
			taskAction.TimeoutKill()
		})
		taskAction.InitTimeoutTimer(timeoutTimer)
	}
	go func() {
		taskAction.HookRun()
		taskAction.Finish()
		m.taskIdMap.Delete(taskAction.GetTaskId())
	}()
}
