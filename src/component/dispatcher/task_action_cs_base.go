package atframework_component_dispatcher

import (
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	pu "github.com/atframework/atframe-utils-go/proto_utility"

	config "github.com/atframework/atsf4g-go/component-config"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionCSSession interface {
	GetSessionId() uint64
	GetSessionNodeId() uint64
	AllocSessionSequence() uint64

	GetActorLogWriter() libatapp.LogWriter

	GetUser() TaskActionCSUser
	BindUser(ctx RpcContext, user TaskActionCSUser)

	GetDispatcher() DispatcherImpl
	SendMessage(*public_protocol_extension.CSMsg) error

	IsEnableActorLog() bool
	InsertPendingActorLog(string)
}

type TaskActionCSUser interface {
	GetUserId() uint64
	GetZoneId() uint32
	GetOpenId() string

	GetSession() TaskActionCSSession
	GetCsActorLogWriter() libatapp.LogWriter
	GetActorExecutor() *ActorExecutor

	OnSendResponse(ctx RpcContext) error

	// 每次执行任务前刷新
	RefreshLimit(RpcContext, time.Time)
}

type TaskActionCSBase[RequestType proto.Message, ResponseType proto.Message] struct {
	TaskActionBase

	session TaskActionCSSession
	user    TaskActionCSUser

	rpcDescriptor   protoreflect.MethodDescriptor
	requestHead     *public_protocol_extension.CSMsgHead
	requestBody     RequestType
	responseBody    ResponseType
	responseFactory func() ResponseType
}

func CreateCSTaskAction(
	ctx RpcContext,
	rd DispatcherImpl,
	session TaskActionCSSession,
	rpcDescriptor protoreflect.MethodDescriptor,
	createFn func(RpcContext, DispatcherImpl, TaskActionCSSession, protoreflect.MethodDescriptor) TaskActionImpl,
) TaskActionImpl {
	ret := createFn(ctx, rd, session, rpcDescriptor)
	ret.SetImplementation(ret)
	libatapp.AtappGetModule[*TaskManager](rd.GetApp()).InsertTaskAction(ctx, ret)
	return ret
}

func CreateCSTaskActionBase[RequestType proto.Message, ResponseType proto.Message](
	ctx RpcContext,
	rd DispatcherImpl,
	session TaskActionCSSession,
	rpcDescriptor protoreflect.MethodDescriptor,
	zeroRequest RequestType,
	responseFactory func() ResponseType,
) TaskActionCSBase[RequestType, ResponseType] {
	var user TaskActionCSUser = nil
	var actor *ActorExecutor = nil
	if !lu.IsNil(session) {
		user = session.GetUser()
	}
	if !lu.IsNil(user) {
		actor = user.GetActorExecutor()
	}

	return TaskActionCSBase[RequestType, ResponseType]{
		TaskActionBase:  CreateTaskActionBase(rd, actor, config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetTask().GetCsmsg().GetTimeout().AsDuration()),
		session:         session,
		user:            user,
		rpcDescriptor:   rpcDescriptor,
		requestHead:     nil,
		requestBody:     zeroRequest,
		responseFactory: responseFactory,
	}
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetLogger() *slog.Logger {
	return t.GetDispatcher().GetLogger()
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetUser(user TaskActionCSUser) {
	if lu.IsNil(user) {
		t.user = nil
		return
	}

	t.user = user
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetUser() TaskActionCSUser {
	if lu.IsNil(t.user) {
		if !lu.IsNil(t.session) {
			t.user = t.session.GetUser()
		}
	}

	if lu.IsNil(t.user) {
		return nil
	}

	return t.user
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetSession() TaskActionCSSession {
	return t.session
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetSession(session TaskActionCSSession) {
	if lu.IsNil(session) {
		if !lu.IsNil(t.session) {
			if t.user == t.session.GetUser() {
				t.user = nil
			}
			t.session = nil
		}

		return
	}

	// 换绑session也要换绑定user
	t.session = session
	t.user = nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) IsStreamRpc() bool {
	if lu.IsNil(t.rpcDescriptor) {
		return false
	}

	return t.rpcDescriptor.IsStreamingClient() || t.rpcDescriptor.IsStreamingServer()
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestHead() *public_protocol_extension.CSMsgHead {
	if lu.IsNil(t.requestHead) {
		return &public_protocol_extension.CSMsgHead{}
	}

	return t.requestHead
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestBody() RequestType {
	return t.requestBody
}

func (t *TaskActionCSBase[RequestType, ResponseType]) MutableResponseBody() ResponseType {
	// 检查responseBody是否为nil
	if lu.IsNil(t.responseBody) {
		// 使用反射创建ResponseType的新实例
		t.responseBody = t.responseFactory()
	}

	return t.responseBody
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetCsActorLogWriter() libatapp.LogWriter {
	user := t.GetUser()
	if lu.IsNil(user) {
		return nil
	}
	return user.GetCsActorLogWriter()
}

func CreateCSMessage(responseCode int32, timestamp time.Time, clientSequence uint64,
	rd DispatcherImpl, session TaskActionCSSession,
	rpcType interface{}, body proto.Message,
) (*public_protocol_extension.CSMsg, error) {
	responseMsg := &public_protocol_extension.CSMsg{
		Head: &public_protocol_extension.CSMsgHead{
			// 复制请求头的一些信息
			ErrorCode:       responseCode,
			Timestamp:       timestamp.Unix(),
			ClientSequence:  clientSequence,
			ServerSequence:  rd.AllocSequence(),
			SessionSequence: 0,
			SessionId:       0,
			SessionNodeId:   0,
			SessionNodeName: "",
			RpcType:         nil,
		},
	}

	switch v := rpcType.(type) {
	case *public_protocol_extension.RpcResponseMeta:
		responseMsg.Head.RpcType = &public_protocol_extension.CSMsgHead_RpcResponse{
			RpcResponse: v,
		}
	case *public_protocol_extension.RpcRequestMeta:
		responseMsg.Head.RpcType = &public_protocol_extension.CSMsgHead_RpcRequest{
			RpcRequest: v,
		}
	case *public_protocol_extension.RpcStreamMeta:
		responseMsg.Head.RpcType = &public_protocol_extension.CSMsgHead_RpcStream{
			RpcStream: v,
		}
	default:
		return nil, fmt.Errorf("invalid RpcType for CSMsg: %T", rpcType)
	}

	if !lu.IsNil(session) {
		responseMsg.Head.SessionSequence = session.AllocSessionSequence()
		responseMsg.Head.SessionId = session.GetSessionId()
		responseMsg.Head.SessionNodeId = session.GetSessionNodeId()
		// TODO: 是否需要 SessionNodeName?
		// responseMsg.Head.SessionNodeName = FindNodeById(t.session.GetSessionNodeId()).Name()
	}

	// 序列化响应体 - 需要检查是否为零值
	var responseBodyBytes []byte
	var err error

	// 由于 ResponseType 是泛型，我们不能直接与 nil 比较
	// 需要使用反射或者其他方式检查，这里先尝试序列化
	if responseBodyBytes, err = proto.Marshal(body); err != nil {
		rd.GetLogger().Error("Failed to marshal response body",
			"session_id", responseMsg.Head.SessionId,
			"client_sequence", responseMsg.Head.ClientSequence,
			"response_code", responseCode,
			"error", err.Error())
		return nil, fmt.Errorf("failed to marshal response body: %w", err)
	}
	responseMsg.BodyBin = responseBodyBytes

	return responseMsg, nil
}

// SendResponse 实现响应发送逻辑
func (t *TaskActionCSBase[RequestType, ResponseType]) SendResponse() error {
	if t.IsResponseDisabled() || t.IsStreamRpc() {
		return nil
	}

	var clientSequence uint64 = 0
	if !lu.IsNil(t.requestHead) {
		clientSequence = t.requestHead.ClientSequence
	}

	// 构造响应消息
	now := t.GetSysNow()
	responseMsg, err := CreateCSMessage(t.GetResponseCode(), now, clientSequence, t.GetDispatcher(), t.session,
		&public_protocol_extension.RpcResponseMeta{
			// TODO: 配置模块加载
			Version:         "0.1.0",
			RpcName:         string(t.rpcDescriptor.FullName()),
			TypeUrl:         string(t.rpcDescriptor.Output().FullName()),
			CallerNodeId:    t.GetDispatcher().GetApp().GetId(),
			CallerNodeName:  t.GetDispatcher().GetApp().GetAppName(),
			CallerTimestamp: timestamppb.New(now),
		},
		t.responseBody)
	if err != nil {
		return err
	}

	// 实际发送逻辑需要根据具体的网络层实现
	if !lu.IsNil(t.GetDispatcher()) && !lu.IsNil(t.GetDispatcher().GetApp()) {
		t.GetDispatcher().GetLogger().Info("Sending CS response",
			"session_id", responseMsg.Head.SessionId,
			"client_sequence", responseMsg.Head.ClientSequence,
			"response_code", t.GetResponseCode())
		// 输出CSLOG
		logWriter := t.GetCsActorLogWriter()
		if logWriter != nil {
			fmt.Fprintf(logWriter, "%s >>>>>>>>>>>>>>>>>>>> Session: %d Sending: %s\n", now.Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.rpcDescriptor.Output().FullName())
			fmt.Fprintf(logWriter, "Head:{\n%s}\n", pu.MessageReadableText(responseMsg.Head))
			fmt.Fprintf(logWriter, "Body:{\n%s}\n\n", pu.MessageReadableText(t.responseBody))
		} else if t.session.IsEnableActorLog() {
			t.session.InsertPendingActorLog(fmt.Sprintf("%s >>>>>>>>>>>>>>>>>>>> Session: %d Sending: %s\n", now.Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.rpcDescriptor.Output().FullName()))
			t.session.InsertPendingActorLog(fmt.Sprintf("Head:{\n%s}\n", pu.MessageReadableText(responseMsg.Head)))
			t.session.InsertPendingActorLog(fmt.Sprintf("Body:{\n%s}\n\n", pu.MessageReadableText(t.responseBody)))
		}

		err = t.session.SendMessage(responseMsg)
		if err != nil {
			t.GetDispatcher().GetLogger().Error("Failed to send CS response",
				"session_id", responseMsg.Head.SessionId,
				"client_sequence", responseMsg.Head.ClientSequence,
				"response_code", t.GetResponseCode(),
				"error", err.Error())

			t.GetDispatcher().OnSendMessageFailed(t.GetRpcContext(), &DispatcherRawMessage{
				Type:     t.GetDispatcher().GetInstanceIdent(),
				Instance: responseMsg,
			}, responseMsg.Head.ServerSequence, err)
			return err
		}
	}

	return nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) CheckPermission() (int32, error) {
	if !t.impl.AllowNoActor() && lu.IsNil(t.GetUser()) {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_LOGIN), nil
	}

	return 0, nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) AllowNoActor() bool {
	return false
}

func (t *TaskActionCSBase[RequestType, ResponseType]) OnSendResponse() {
	// 脏数据自动推送
	user := t.GetUser()
	if !lu.IsNil(user) {
		user.OnSendResponse(t.GetRpcContext())
	}
}

func (t *TaskActionCSBase[RequestType, ResponseType]) HookRun(startData *DispatcherStartData) error {
	t.PrepareHookRun(startData)

	csMsg, ok := startData.Message.Instance.(*public_protocol_extension.CSMsg)
	if !ok {
		return fmt.Errorf("TaskActionCSBase: invalid message type %v", startData.Message.Type)
	}

	// 保存原始消息数据
	bodyData := csMsg.BodyBin

	t.requestHead = csMsg.Head

	// 创建请求体实例
	if len(bodyData) > 0 {
		if err := proto.Unmarshal(bodyData, t.requestBody); err != nil {
			return fmt.Errorf("failed to parse request body: %w", err)
		}
	}

	// 清空 CSMsg 的 BodyBin，因为已经解析到了 requestBody
	csMsg.BodyBin = []byte{}

	// 输出CSLOG
	logWriter := t.GetCsActorLogWriter()
	if logWriter != nil {
		fmt.Fprintf(logWriter, "%s <<<<<<<<<<<<<<<<<<<< Session: %d Received: %s\n", t.GetSysNow().Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.requestHead.GetRpcRequest().GetTypeUrl())
		fmt.Fprintf(logWriter, "Head:{\n%s}\n", pu.MessageReadableText(t.requestHead))
		fmt.Fprintf(logWriter, "Body:{\n%s}\n\n", pu.MessageReadableText(t.requestBody))
	} else if t.session.IsEnableActorLog() {
		t.session.InsertPendingActorLog(fmt.Sprintf("%s <<<<<<<<<<<<<<<<<<<< Session: %d Received: %s\n", t.GetSysNow().Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.requestHead.GetRpcRequest().GetTypeUrl()))
		t.session.InsertPendingActorLog(fmt.Sprintf("Head:{\n%s}\n", pu.MessageReadableText(t.requestHead)))
		t.session.InsertPendingActorLog(fmt.Sprintf("Body:{\n%s}\n\n", pu.MessageReadableText(t.requestBody)))
	}

	user := t.GetUser()
	if !lu.IsNil(user) {
		// 每次执行任务前刷新
		user.RefreshLimit(t.GetRpcContext(), t.GetNow())
	}

	err := t.TaskActionBase.HookRun(startData)

	return err
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetTypeName() string {
	return "CS Task Action"
}
