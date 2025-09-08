package atframework_component_dispatcher

import (
	"context"
	"unsafe"

	libatapp "github.com/atframework/libatapp-go"
)

type MessageFilterHandler func(msg *DispatcherRawMessage) bool

// 调度器接口
type DispatcherImpl interface {
	libatapp.AppModuleImpl

	GetInstanceIdent() uint64
	IsClosing() bool

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

	messageFilters []MessageFilterHandler
}

func (dispatcher *DispatcherBase) Init(_parent context.Context) error {
	return nil
}

func (dispatcher *DispatcherBase) GetInstanceIdent() uint64 {
	return uint64(uintptr(unsafe.Pointer(dispatcher)))
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
