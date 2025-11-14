package atframework_component_dispatcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	libatapp "github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type MessageFilterHandler func(rd DispatcherImpl, msg *DispatcherRawMessage) bool

type TaskActionCreator = func(rd DispatcherImpl, startData *DispatcherStartData) (TaskActionImpl, error)

// 调度器接口
type DispatcherImpl interface {
	libatapp.AppModuleImpl

	GetInstanceIdent() uint64
	IsClosing() bool

	AllocTaskId() uint64
	AllocSequence() uint64
	GetNow() time.Time
	GetLogger() *slog.Logger

	OnSendMessageFailed(rpcContext *RpcContext, msg *DispatcherRawMessage, sequence uint64, err error)
	OnCreateTaskFailed(startData *DispatcherStartData, err error)

	OnReceiveMessage(parentContext context.Context, msg *DispatcherRawMessage, privateData interface{}, sequence uint64) error

	PickMessageTaskId(msg *DispatcherRawMessage) uint64
	PickMessageRpcName(msg *DispatcherRawMessage) string

	CreateTask(startData *DispatcherStartData) (TaskActionImpl, error)

	RegisterAction(ServiceDescriptor protoreflect.ServiceDescriptor, rpcFullName string, creator TaskActionCreator) error
	GetRegisteredService(serviceFullName string) protoreflect.ServiceDescriptor
	GetRegisteredMethod(methodFullName string) protoreflect.MethodDescriptor

	PushFrontMessageFilter(handle MessageFilterHandler)
	PushBackMessageFilter(handle MessageFilterHandler)

	CreateRpcContext() *RpcContext
}

type taskActionCreatorData struct {
	service protoreflect.ServiceDescriptor
	method  protoreflect.MethodDescriptor
	creator TaskActionCreator
	options *public_protocol_extension.DispatcherOptions
}

type DispatcherBase struct {
	libatapp.AppModuleBase
	impl DispatcherImpl

	sequenceAllocator atomic.Uint64
	messageFilters    []MessageFilterHandler

	registeredService map[string]protoreflect.ServiceDescriptor
	registeredMethod  map[string]protoreflect.MethodDescriptor
	registeredCreator map[string]taskActionCreatorData
}

func CreateDispatcherBase(owner libatapp.AppImpl) DispatcherBase {
	return DispatcherBase{
		AppModuleBase:     libatapp.CreateAppModuleBase(owner),
		sequenceAllocator: atomic.Uint64{},
		messageFilters:    make([]MessageFilterHandler, 0),
		registeredService: make(map[string]protoreflect.ServiceDescriptor),
		registeredMethod:  make(map[string]protoreflect.MethodDescriptor),
		registeredCreator: make(map[string]taskActionCreatorData),
	}
}

func (dispatcher *DispatcherBase) Init(_initCtx context.Context) error {
	dispatcher.sequenceAllocator.Store(
		// 使用时间戳作为初始值, 避免与重启前的值冲突
		uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()),
	)
	return nil
}

func (dispatcher *DispatcherBase) GetInstanceIdent() uint64 {
	return uint64(uintptr(unsafe.Pointer(dispatcher)))
}

func (dispatcher *DispatcherBase) IsClosing() bool {
	return dispatcher.GetApp().IsClosing()
}

var taskIdAllocator = atomic.Uint64{}

func (dispatcher *DispatcherBase) AllocTaskId() uint64 {
	ret := taskIdAllocator.Add(1)
	if ret <= 1 {
		taskIdAllocator.Store(
			// 使用时间戳作为初始值, 避免与重启前的值冲突
			uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()),
		)
		ret = taskIdAllocator.Add(1)
	}

	return ret
}

func (dispatcher *DispatcherBase) AllocSequence() uint64 {
	return dispatcher.sequenceAllocator.Add(1)
}

func (dispatcher *DispatcherBase) GetNow() time.Time {
	// TODO: 使用逻辑时间戳 Timestamp
	return time.Now()
}

func (dispatcher *DispatcherBase) GetLogger() *slog.Logger {
	app := dispatcher.GetApp()
	if lu.IsNil(app) {
		return slog.Default()
	}

	return app.GetDefaultLogger()
}

func (dispatcher *DispatcherBase) OnSendMessageFailed(rpcContext *RpcContext, msg *DispatcherRawMessage, sequence uint64, err error) {
	dispatcher.impl.GetLogger().Error("OnSendMessageFailed", "error", err, "sequence", sequence, "message_type", msg.Type)
}

func (dispatcher *DispatcherBase) OnCreateTaskFailed(startData *DispatcherStartData, err error) {
	dispatcher.impl.GetLogger().Error("OnCreateTaskFailed", "error", err, "message_type", startData.Message.Type, "rpc_name", dispatcher.impl.PickMessageRpcName(startData.Message))
}

func (dispatcher *DispatcherBase) OnReceiveMessage(parentContext context.Context, msg *DispatcherRawMessage, privateData interface{}, sequence uint64) error {
	if msg == nil || lu.IsNil(msg.Instance) {
		dispatcher.GetLogger().Error("OnReceiveMessage message can not be nil", "sequence", sequence)
		return fmt.Errorf("OnReceiveMessage message can not be nil")
	}

	if msg.Type != dispatcher.impl.GetInstanceIdent() {
		dispatcher.GetLogger().Error("OnReceiveMessage message type mismatch", "expect", dispatcher.impl.GetInstanceIdent(), "got", msg.Type, "sequence", sequence)
		return fmt.Errorf("OnReceiveMessage message type mismatch, expect %d, got %d", dispatcher.impl.GetInstanceIdent(), msg.Type)
	}

	if len(dispatcher.messageFilters) > 0 {
		for _, filter := range dispatcher.messageFilters {
			if !filter(dispatcher.impl, msg) {
				// 被过滤掉了
				return nil
			}
		}
	}

	resumeTaskId := dispatcher.impl.PickMessageTaskId(msg)
	if resumeTaskId != 0 {
		// TODO: 处理恢复任务
		return nil
	}

	rpcContext := dispatcher.CreateRpcContext()
	if parentContext != nil {
		rpcContext.Context, rpcContext.CancelFn = context.WithCancel(parentContext)
	}

	startData := &DispatcherStartData{
		Message:           msg,
		PrivateData:       privateData,
		MessageRpcContext: rpcContext,
	}

	action, err := dispatcher.impl.CreateTask(startData)
	if err != nil {
		dispatcher.GetLogger().Error("OnReceiveMessage CreateTask failed", slog.String("error", err.Error()), "sequence", sequence, "rpc_name", dispatcher.impl.PickMessageRpcName(msg))
		dispatcher.OnCreateTaskFailed(startData, err)

		if rpcContext.CancelFn != nil {
			cancelFn := rpcContext.CancelFn
			rpcContext.CancelFn = nil
			cancelFn()
		}
		return err
	}
	rpcContext.taskAction = action

	err = RunTaskAction(dispatcher.impl.GetApp(), action, startData)
	if err != nil {
		dispatcher.GetLogger().Error("OnReceiveMessage RunTaskAction failed", slog.String("error", err.Error()), "sequence", sequence, "rpc_name", dispatcher.impl.PickMessageRpcName(msg), "task_id", action.GetTaskId(), "task_name", action.GetTypeName())
		if rpcContext.CancelFn != nil {
			cancelFn := rpcContext.CancelFn
			rpcContext.CancelFn = nil
			cancelFn()
		}
		return err
	}
	return nil
}

func (dispatcher *DispatcherBase) CreateTask(startData *DispatcherStartData) (TaskActionImpl, error) {
	rpcFullName := dispatcher.impl.PickMessageRpcName(startData.Message)
	if rpcFullName == "" {
		return nil, fmt.Errorf("CreateTask rpc name can not be empty")
	}

	creator, ok := dispatcher.registeredCreator[rpcFullName]
	if !ok || lu.IsNil(creator.creator) {
		return nil, fmt.Errorf("CreateTask rpc %s not registered", rpcFullName)
	}

	return creator.creator(dispatcher.impl, startData)
}

func (dispatcher *DispatcherBase) RegisterAction(serviceDescriptor protoreflect.ServiceDescriptor, rpcFullName string, creator TaskActionCreator) error {
	// 实现注册逻辑
	if lu.IsNil(serviceDescriptor) {
		return fmt.Errorf("RegisterAction ServiceDescriptor can not be nil")
	}

	if lu.IsNil(creator) {
		return fmt.Errorf("RegisterAction creator can not be nil")
	}

	serviceFullName := string(serviceDescriptor.FullName())
	dispatcher.registeredService[serviceFullName] = serviceDescriptor

	rpcShortName := rpcFullName[strings.LastIndex(rpcFullName, ".")+1:]
	methodDescriptor := serviceDescriptor.Methods().ByName(protoreflect.Name(rpcShortName))
	if lu.IsNil(methodDescriptor) {
		dispatcher.GetLogger().Error("RegisterAction method not found", "method", rpcShortName, "service", serviceFullName)
		return fmt.Errorf("RegisterAction method %s not found in service %s", rpcShortName, serviceFullName)
	}

	dispatcher.registeredMethod[string(methodDescriptor.FullName())] = methodDescriptor

	methodOpts := methodDescriptor.Options().(*descriptorpb.MethodOptions)

	var options *public_protocol_extension.DispatcherOptions = nil
	if proto.HasExtension(methodOpts, public_protocol_extension.E_RpcOptions) {
		options = proto.GetExtension(methodOpts, public_protocol_extension.E_RpcOptions).(*public_protocol_extension.DispatcherOptions)
	}

	dispatcher.registeredCreator[string(methodDescriptor.FullName())] = taskActionCreatorData{
		service: serviceDescriptor,
		method:  methodDescriptor,
		creator: creator,
		options: options,
	}
	return nil
}

func (dispatcher *DispatcherBase) GetRegisteredService(serviceFullName string) protoreflect.ServiceDescriptor {
	return dispatcher.registeredService[serviceFullName]
}

func (dispatcher *DispatcherBase) GetRegisteredMethod(methodFullName string) protoreflect.MethodDescriptor {
	return dispatcher.registeredMethod[methodFullName]
}

func (dispatcher *DispatcherBase) PushFrontMessageFilter(handle MessageFilterHandler) {
	dispatcher.messageFilters = append([]MessageFilterHandler{handle}, dispatcher.messageFilters...)
}

func (dispatcher *DispatcherBase) PushBackMessageFilter(handle MessageFilterHandler) {
	dispatcher.messageFilters = append(dispatcher.messageFilters, handle)
}

func (dispatcher *DispatcherBase) CreateRpcContext() *RpcContext {
	return &RpcContext{
		app:        dispatcher.GetApp(),
		dispatcher: dispatcher.impl,
	}
}

func CreateRpcResultOk() RpcResult {
	return RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func CreateRpcResultOkResponse(responseCode int32) RpcResult {
	return RpcResult{
		Error:        nil,
		ResponseCode: responseCode,
	}
}

func CreateRpcResultError(err error, responseCode public_protocol_pbdesc.EnErrorCode) RpcResult {
	return RpcResult{
		Error:        err,
		ResponseCode: int32(responseCode),
	}
}

func (der *RpcResult) IsOK() bool {
	return der.Error == nil && der.ResponseCode >= 0
}

func (der *RpcResult) IsError() bool {
	return der != nil && (der.Error != nil || der.ResponseCode < 0)
}

func (der *RpcResult) GetStandardError() error {
	if der == nil {
		return nil
	}

	return der.Error
}

func (der *RpcResult) GetResponseCode() int32 {
	if der.IsOK() {
		return der.ResponseCode
	}
	if der.ResponseCode < 0 {
		return der.ResponseCode
	}

	return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_UNKNOWN)
}

func (der *RpcResult) GetResponseMessage() string {
	code := der.GetResponseCode()
	if code >= 0 {
		return ""
	}

	name, ok := public_protocol_pbdesc.EnErrorCode_name[code]
	if !ok {
		return "UnknownError"
	}

	ec := public_protocol_pbdesc.EnErrorCode(0).Descriptor().Values().ByNumber(protoreflect.EnumNumber(code))
	if desc := proto.GetExtension(ec.Options(), public_protocol_extension.E_Description); desc != nil {
		descStr, ok := desc.(string)
		if ok && descStr != "" {
			return fmt.Sprintf("%s(%s)", descStr, name)
		}
	}

	return name
}

func (der *RpcResult) GetErrorString() string {
	if der.Error != nil {
		return der.Error.Error()
	}

	return der.GetResponseMessage()
}

func (der *RpcResult) LogWithLevelContextWithCaller(pc uintptr, c context.Context, level slog.Level, ctx *RpcContext, msg string, args ...any) {
	if der.IsOK() {
		if ctx != nil {
			ctx.LogWithLevelContextWithCaller(pc, c, level, msg, args...)
		} else {
			libatapp.LogInner(slog.Default(), pc, c, level, msg, args...)
		}
		return
	}

	if der.Error != nil {
		args = append(args, slog.String("error", der.Error.Error()))
	}
	if der.ResponseCode < 0 {
		args = append(args, slog.Int64("response_code", int64(der.ResponseCode)), slog.String("response_message", der.GetResponseMessage()))
	}

	if ctx != nil {
		ctx.LogWithLevelContextWithCaller(pc, c, level, msg, args...)
	} else {
		libatapp.LogInner(slog.Default(), pc, c, level, msg, args...)
	}
}

func (der *RpcResult) LogWithLevelWithCaller(pc uintptr, level slog.Level, ctx *RpcContext, msg string, args ...any) {
	if ctx == nil || ctx.Context == nil {
		der.LogWithLevelContextWithCaller(pc, context.Background(), level, ctx, msg, args...)
		return
	}
	der.LogWithLevelContextWithCaller(pc, ctx.Context, level, ctx, msg, args...)
}

// ====================== 业务日志接口 =========================

func (der *RpcResult) LogErrorContext(c context.Context, ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelError, ctx, msg, args...)
}

func (der *RpcResult) LogError(ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelError, ctx, msg, args...)
}

func (der *RpcResult) LogWarnContext(c context.Context, ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelWarn, ctx, msg, args...)
}

func (der *RpcResult) LogWarn(ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelWarn, ctx, msg, args...)
}

func (der *RpcResult) LogInfoContext(c context.Context, ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelInfo, ctx, msg, args...)
}

func (der *RpcResult) LogInfo(ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelInfo, ctx, msg, args...)
}

func (der *RpcResult) LogDebugContext(c context.Context, ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelContextWithCaller(libatapp.GetCaller(1), c, slog.LevelDebug, ctx, msg, args...)
}

func (der *RpcResult) LogDebug(ctx *RpcContext, msg string, args ...any) {
	der.LogWithLevelWithCaller(libatapp.GetCaller(1), slog.LevelDebug, ctx, msg, args...)
}
