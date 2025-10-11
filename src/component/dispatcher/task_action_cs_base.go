package atframework_component_dispatcher

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type TaskActionCSSession interface {
	GetSessionId() uint64
	GetSessionNodeId() uint64
	AllocSessionSequence() uint64

	GetUser() TaskActionCSUser
	BindUser(ctx *RpcContext, user TaskActionCSUser)

	SendMessage(*public_protocol_extension.CSMsg) error
}

type TaskActionCSUser interface {
	GetUserId() uint64
	GetZoneId() uint32

	GetSession() TaskActionCSSession

	GetActorExecutor() *ActorExecutor

	SendAllSyncData() error

	// 每次执行任务前刷新
	RefreshLimit(*RpcContext, time.Time)
}

type TaskActionCSBase[RequestType proto.Message, ResponseType proto.Message] struct {
	TaskActionBase

	session TaskActionCSSession
	user    TaskActionCSUser

	rpcDescriptor protoreflect.MethodDescriptor
	requestHead   *public_protocol_extension.CSMsgHead
	requestBody   RequestType
	responseBody  ResponseType
}

func CreateTaskActionCSBase[RequestType proto.Message, ResponseType proto.Message](
	rd DispatcherImpl,
	session TaskActionCSSession,
	rpcDescriptor protoreflect.MethodDescriptor,
) TaskActionCSBase[RequestType, ResponseType] {
	var user TaskActionCSUser = nil
	var actor *ActorExecutor = nil
	if session != nil {
		user = session.GetUser()
	}
	if user != nil {
		actor = user.GetActorExecutor()
	}

	// 创建RequestType的零值实例
	requestBodyType := reflect.TypeOf((*RequestType)(nil)).Elem().Elem()

	return TaskActionCSBase[RequestType, ResponseType]{
		TaskActionBase: CreateTaskActionBase(rd, actor),
		session:        session,
		user:           user,
		rpcDescriptor:  rpcDescriptor,
		requestHead:    nil,
		requestBody:    reflect.New(requestBodyType).Interface().(RequestType),
	}
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetLogger() *slog.Logger {
	return t.GetDispatcher().GetApp().GetLogger()
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetUser(user TaskActionCSUser) {
	if user == nil {
		t.user = nil
		return
	}

	t.user = user
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetUser() TaskActionCSUser {
	if t.user == nil {
		if t.session != nil {
			t.user = t.session.GetUser()
		}
	}

	return t.user
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetSession() TaskActionCSSession {
	return t.session
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetSession(session TaskActionCSSession) {
	if session == nil {
		if t.session != nil {
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
	if t.rpcDescriptor == nil {
		return false
	}

	return t.rpcDescriptor.IsStreamingClient() || t.rpcDescriptor.IsStreamingServer()
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestHead() *public_protocol_extension.CSMsgHead {
	if t.requestHead == nil {
		return &public_protocol_extension.CSMsgHead{}
	}

	return t.requestHead
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestBody() RequestType {
	return t.requestBody
}

func (t *TaskActionCSBase[RequestType, ResponseType]) MutableResponseBody() ResponseType {
	// 检查responseBody是否为nil
	if reflect.ValueOf(t.responseBody).IsNil() {
		// 使用反射创建ResponseType的新实例
		responseType := reflect.TypeOf(t.responseBody).Elem()
		newInstance := reflect.New(responseType)
		t.responseBody = newInstance.Interface().(ResponseType)
	}

	return t.responseBody
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

	if session != nil {
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
		rd.GetApp().GetLogger().Error("Failed to marshal response body",
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
	if t.requestHead != nil {
		clientSequence = t.requestHead.ClientSequence
	}

	// 构造响应消息
	// TODO: 使用全局时间戳 Timestamp
	now := t.GetNow()
	responseMsg, err := CreateCSMessage(t.GetResponseCode(), now, clientSequence, t.GetDispatcher(), t.session,
		&public_protocol_extension.RpcResponseMeta{
			// TODO: 配置模块加载
			Version:         "0.1.0",
			RpcName:         string(t.rpcDescriptor.FullName()),
			TypeUrl:         string(t.rpcDescriptor.Output().FullName()),
			CallerNodeId:    t.GetDispatcher().GetApp().GetAppId(),
			CallerNodeName:  t.GetDispatcher().GetApp().GetAppName(),
			CallerTimestamp: timestamppb.New(now),
		},
		t.responseBody)
	if err != nil {
		return err
	}

	// 实际发送逻辑需要根据具体的网络层实现
	if t.GetDispatcher() != nil && t.GetDispatcher().GetApp() != nil {
		t.GetDispatcher().GetApp().GetLogger().Info("Sending CS response",
			"session_id", responseMsg.Head.SessionId,
			"client_sequence", responseMsg.Head.ClientSequence,
			"response_code", t.GetResponseCode())
		err = t.session.SendMessage(responseMsg)
		if err != nil {
			t.GetDispatcher().GetApp().GetLogger().Error("Failed to send CS response",
				"session_id", responseMsg.Head.SessionId,
				"client_sequence", responseMsg.Head.ClientSequence,
				"response_code", t.GetResponseCode(),
				"error", err.Error())

			t.GetDispatcher().OnSendMessageFailed(t.GetDispatcher(), t.GetRpcContext(), &DispatcherRawMessage{
				Type:     t.GetDispatcher().GetInstanceIdent(),
				Instance: responseMsg,
			}, responseMsg.Head.ServerSequence, err)
			return err
		}
	}

	return nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) CheckPermission(action TaskActionImpl) (int32, error) {
	if !action.AllowNoActor() && t.GetUser() == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_LOGIN), nil
	}

	return 0, nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) AllowNoActor() bool {
	return false
}

func (t *TaskActionCSBase[RequestType, ResponseType]) HookRun(action TaskActionImpl, startData *DispatcherStartData) error {
	t.PrepareHookRun(action, startData)

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

	user := t.GetUser()
	if user != nil {
		// 每次执行任务前刷新
		user.RefreshLimit(t.rpcContext, t.GetNow())
	}

	err := t.TaskActionBase.HookRun(action, startData)

	// 脏数据自动推送
	if user == nil {
		user = t.GetUser()
	}
	if user != nil {
		user.SendAllSyncData()
	}

	return err
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetTypeName() string {
	return "CS Task Action"
}

type TaskActionCSTest struct {
	TaskActionCSBase[*public_protocol_pbdesc.DClientDeviceInfo, *public_protocol_pbdesc.DClientDeviceInfo]
}

func (t *TaskActionCSTest) Run(startData DispatcherStartData) error {
	body := t.GetRequestBody()
	_ = body // 使用变量避免编译错误

	// TODO: 实现具体的业务逻辑
	// 示例：处理设备信息
	// if body != nil {
	//     // 处理设备信息...
	// }

	return nil
}
