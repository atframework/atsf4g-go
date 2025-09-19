package atframework_component_dispatcher

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"

	libatapp "github.com/atframework/libatapp-go"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
)

type MessageFilterHandler func(msg *DispatcherRawMessage) bool

// 调度器接口
type DispatcherImpl interface {
	libatapp.AppModuleImpl

	GetInstanceIdent() uint64
	IsClosing() bool

	AllocSequence() uint64

	OnSendMessageFailed(rpcContext *RpcContext, msg *DispatcherRawMessage, sequence uint64, err error)
	OnCreateTaskFailed(startData DispatcherStartData, err error)

	PickMessageTaskId(msg *DispatcherRawMessage) uint64
	PickMessageRpcName(msg *DispatcherRawMessage) string
	PickMessageOpType(msg *DispatcherRawMessage) MessageOpType

	CreateTask(startData DispatcherStartData) (TaskActionImpl, error)

	RegisterAction(ServiceDescriptor interface{}, rpcFullName string) error
	GetRegisteredService(serviceFullName string) interface{}
	GetRegisteredMethod(methodFullName string) interface{}

	PushFrontMessageFilter(handle MessageFilterHandler)
	PushBackMessageFilter(handle MessageFilterHandler)
}

type DispatcherBase struct {
	libatapp.AppModuleBase

	sequenceAllocator atomic.Uint64
	messageFilters    []MessageFilterHandler
}

func (dispatcher *DispatcherBase) Init(_initCtx context.Context) error {
	dispatcher.sequenceAllocator.Store(
		// 使用时间戳作为初始值, 避免与重启前的值冲突
		uint64(time.Now().Sub(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()),
	)
	return nil
}

func (dispatcher *DispatcherBase) GetInstanceIdent() uint64 {
	return uint64(uintptr(unsafe.Pointer(dispatcher)))
}

func (dispatcher *DispatcherBase) AllocSequence() uint64 {
	return dispatcher.sequenceAllocator.Add(1)
}

func (dispatcher *DispatcherBase) OnReceiveMessage(rpcContext *RpcContext, msg *DispatcherRawMessage, privateData interface{}, sequence uint64) error {
	return nil
}

func (dispatcher *DispatcherBase) PushFrontMessageFilter(handle MessageFilterHandler) {
	dispatcher.messageFilters = append([]MessageFilterHandler{handle}, dispatcher.messageFilters...)
}

func (dispatcher *DispatcherBase) PushBackMessageFilter(handle MessageFilterHandler) {
	dispatcher.messageFilters = append(dispatcher.messageFilters, handle)
}
