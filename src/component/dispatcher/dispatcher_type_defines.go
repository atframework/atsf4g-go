package atframework_component_dispatcher

import (
	"container/list"
	"sync"
	"time"
)

type MessageOpType int

const (
	MessageOpTypeMisup MessageOpType = iota // 0
	MessageOpTypeStream
	MessageOpTypeUnaryRequest
	MessageOpTypeUnaryResponse
)

type RpcContext struct{}

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
	PrivateData interface{}

	MessageRpcContext *RpcContext
}

type DispatcherStartData struct {
	Message     *DispatcherRawMessage
	PrivateData interface{}

	// TODO: options
	MessageRpcContext *RpcContext
}

type DispatcherAwaitOptions struct {
	Sequence uint64
	Timeout  time.Duration
}

type ActorExecutorStatus int

const (
	ActorExecutorStatusFree ActorExecutorStatus = iota // 0
	ActorExecutorStatusPending
	ActorExecutorStatusRunning
)

type ActorAction struct {
	action   TaskActionImpl
	callback func() error
}

type ActorExecutor struct {
	current_action TaskActionImpl

	action_status   ActorExecutorStatus
	action_lock     sync.Mutex
	pending_actions list.List

	Instance interface{}
}

type TraceInheritOption struct {
}

type TraceStartOption struct {
}
