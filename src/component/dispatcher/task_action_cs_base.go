package atframework_component_dispatcher

import (
	"fmt"
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
}

type TaskActionCSUser interface {
	GetUserId() uint64
	GetZoneId() uint64
}

type TaskActionCSImpl interface {
	TaskActionImpl

	AllowNewSession() bool
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

func (t *TaskActionCSBase[RequestType, ResponseType]) IsStreamRpc() bool {
	if t.rpcDescriptor == nil {
		return false
	}

	return t.rpcDescriptor.IsStreamingClient() || t.rpcDescriptor.IsStreamingServer()
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestHead() *public_protocol_extension.CSMsgHead {
	return t.requestHead
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetRequestBody() RequestType {
	return t.requestBody
}

func (t *TaskActionCSBase[RequestType, ResponseType]) MutableResponseBody() ResponseType {
	return t.responseBody
}

// SendResponse 实现响应发送逻辑
func (t *TaskActionCSBase[RequestType, ResponseType]) SendResponse() error {
	if t.IsResponseDisabled() || t.IsStreamRpc() {
		return nil
	}

	// 构造响应消息
	// TODO: 使用全局时间戳 Timestamp
	now := time.Now()
	responseMsg := &public_protocol_extension.CSMsg{
		Head: &public_protocol_extension.CSMsgHead{
			// 复制请求头的一些信息
			ErrorCode:       t.GetResponseCode(),
			Timestamp:       now.Unix(),
			ClientSequence:  t.requestHead.ClientSequence,
			ServerSequence:  t.GetDispatcher().AllocSequence(),
			SessionSequence: 0,
			SessionId:       0,
			SessionNodeId:   0,
			SessionNodeName: "",
			RpcType: &public_protocol_extension.CSMsgHead_RpcResponse{
				RpcResponse: &public_protocol_extension.RpcResponseMeta{
					// TODO: 配置模块加载
					Version:         "0.1.0",
					RpcName:         string(t.rpcDescriptor.FullName()),
					TypeUrl:         string(t.rpcDescriptor.Output().FullName()),
					CallerNodeId:    t.GetDispatcher().GetApp().GetAppId(),
					CallerNodeName:  t.GetDispatcher().GetApp().GetAppName(),
					CallerTimestamp: timestamppb.New(now),
				},
			},
		},
	}

	if t.session != nil {
		responseMsg.Head.SessionSequence = t.session.AllocSessionSequence()
		responseMsg.Head.SessionId = t.session.GetSessionId()
		responseMsg.Head.SessionNodeId = t.session.GetSessionNodeId()
		// TODO: 是否需要 SessionNodeName
		// responseMsg.Head.SessionNodeName = FindNodeById(t.session.GetSessionNodeId()).Name()
	}

	// 序列化响应体 - 需要检查是否为零值
	var responseBodyBytes []byte
	var err error

	// 由于 ResponseType 是泛型，我们不能直接与 nil 比较
	// 需要使用反射或者其他方式检查，这里先尝试序列化
	if responseBodyBytes, err = proto.Marshal(t.responseBody); err != nil {
		return fmt.Errorf("failed to marshal response body: %w", err)
	}
	responseMsg.BodyBin = responseBodyBytes

	// TODO: 实际发送逻辑需要根据具体的网络层实现
	// 这里只是示例框架，实际应该通过 dispatcher 或 app 来发送
	if t.GetDispatcher() != nil && t.GetDispatcher().GetApp() != nil {
		t.GetDispatcher().GetApp().GetLogger().Info("Sending CS response",
			"session_id", responseMsg.Head.SessionId,
			"client_sequence", responseMsg.Head.ClientSequence,
			"response_code", t.GetResponseCode())
	}

	return nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) CheckPermission(_action TaskActionImpl) (int32, error) {
	if t.session == nil || t.user == nil {
		return int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_NOT_LOGIN), nil
	}

	return 0, nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) HookRun(action TaskActionImpl, startData DispatcherStartData) error {
	csMsg, ok := startData.Message.Instance.(*public_protocol_extension.CSMsg)
	if !ok {
		return fmt.Errorf("TaskActionCSBase: invalid message type %v", startData.Message.Type)
	}

	// 保存原始消息数据
	bodyData := csMsg.BodyBin

	t.requestHead = csMsg.Head

	// 创建请求体实例
	var reqBody RequestType
	if len(bodyData) > 0 {
		if err := proto.Unmarshal(bodyData, reqBody); err != nil {
			return fmt.Errorf("failed to parse request body: %w", err)
		}
	}
	t.requestBody = reqBody

	// 清空 CSMsg 的 BodyBin，因为已经解析到了 requestBody
	csMsg.BodyBin = []byte{}

	// TODO: 是否自动创建Session
	// TODO: 是否允许无Session

	return t.TaskActionBase.HookRun(action, startData)
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
