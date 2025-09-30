package atframework_component_dispatcher

import (
	"context"
	"log/slog"
	"time"
)

type RpcContext struct {
	Logger *slog.Logger

	Context  context.Context
	CancelFn context.CancelFunc
}

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

type DispatcherErrorResult struct {
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
