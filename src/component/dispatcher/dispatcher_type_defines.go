package atframework_component_dispatcher

import (
	"time"
)

type DispatcherOptions struct {
	// TODO: 使用协议里的结构
}

type DispatcherRawMessage struct {
	Type     uint64
	Instance interface{}
}

type DispatcherResumeData struct {
	Message     *DispatcherRawMessage
	Sequence    uint64
	Result      RpcResult
	PrivateData interface{}

	MessageRpcContext AwaitableContext
}

type DispatcherStartData struct {
	Message     *DispatcherRawMessage
	PrivateData interface{}

	// TODO: options
	MessageRpcContext AwaitableContext
}

type RpcResult struct {
	Error        error
	ResponseCode int32
}

type DispatcherAwaitOptions struct {
	Type     uint64
	Sequence uint64
	Timeout  time.Duration
}

type ActorAction struct {
	action   TaskActionImpl
	callback func() error
}

type TraceInheritOption struct{}

type TraceStartOption struct{}
