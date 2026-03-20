package atframework_component_user_controller

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	pu "github.com/atframework/atframe-utils-go/proto_utility"

	log "github.com/atframework/atframe-utils-go/log"
	config "github.com/atframework/atsf4g-go/component/config"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_extension "github.com/atframework/atsf4g-go/component/protocol/public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component/router"
	libatapp "github.com/atframework/libatapp-go"
)

type TaskActionCSBase[RequestType proto.Message, ResponseType proto.Message] struct {
	cd.TaskActionBase

	session SessionImpl
	user    UserImpl

	rpcDescriptor   protoreflect.MethodDescriptor
	requestHead     *public_protocol_extension.CSMsgHead
	requestBody     RequestType
	responseBody    ResponseType
	responseFactory func() ResponseType
}

func CreateCSTaskAction(
	ctx cd.RpcContext,
	rd cd.DispatcherImpl,
	session SessionImpl,
	rpcDescriptor protoreflect.MethodDescriptor,
	createFn func(cd.RpcContext, cd.DispatcherImpl, SessionImpl, protoreflect.MethodDescriptor) cd.TaskActionImpl,
) cd.TaskActionImpl {
	ret := createFn(ctx, rd, session, rpcDescriptor)
	ret.SetImplementation(ret)
	libatapp.AtappGetModule[*cd.TaskManager](rd.GetApp()).InsertTaskAction(ctx, ret)
	return ret
}

func CreateCSTaskActionBase[RequestType proto.Message, ResponseType proto.Message](
	ctx cd.RpcContext,
	rd cd.DispatcherImpl,
	session SessionImpl,
	rpcDescriptor protoreflect.MethodDescriptor,
	zeroRequest RequestType,
	responseFactory func() ResponseType,
) TaskActionCSBase[RequestType, ResponseType] {
	var user UserImpl = nil
	var actor *cd.ActorExecutor = nil
	if !lu.IsNil(session) {
		user = session.GetUser()
	}
	if !lu.IsNil(user) {
		actor = user.GetActorExecutor()
	}

	return TaskActionCSBase[RequestType, ResponseType]{
		TaskActionBase:  cd.CreateTaskActionBase(rd, actor, config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetTask().GetCsmsg().GetTimeout().AsDuration()),
		session:         session,
		user:            user,
		rpcDescriptor:   rpcDescriptor,
		requestHead:     nil,
		requestBody:     zeroRequest,
		responseFactory: responseFactory,
	}
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetUser(user UserImpl) {
	if lu.IsNil(user) {
		t.user = nil
		return
	}

	t.user = user
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetUser() UserImpl {
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

func (t *TaskActionCSBase[RequestType, ResponseType]) GetSession() SessionImpl {
	return t.session
}

func (t *TaskActionCSBase[RequestType, ResponseType]) SetSession(session SessionImpl) {
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

func (t *TaskActionCSBase[RequestType, ResponseType]) GetActorLogWriter() log.LogWriter {
	sess := t.GetSession()
	if lu.IsNil(sess) {
		return nil
	}
	return sess.GetActorLogWriter()
}

func CreateCSMessage(responseCode int32, timestamp time.Time, clientSequence uint64,
	rd cd.DispatcherImpl, session SessionImpl,
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
		rd.GetLogger().LogError("Failed to marshal response body",
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
	now := t.GetNow()
	responseMsg, err := CreateCSMessage(t.GetResponseCode(), now, clientSequence, t.GetDispatcher(), t.session,
		&public_protocol_extension.RpcResponseMeta{
			// TODO: 配置模块加载
			Version:         "0.1.0",
			RpcName:         string(t.rpcDescriptor.FullName()),
			TypeUrl:         string(t.rpcDescriptor.Output().FullName()),
			CallerNodeId:    t.GetDispatcher().GetApp().GetId(),
			CallerNodeName:  t.GetDispatcher().GetApp().GetAppName(),
			CallerTimestamp: timestamppb.New(t.GetSysNow()),
		},
		t.responseBody)
	if err != nil {
		return err
	}

	// 实际发送逻辑需要根据具体的网络层实现
	if !lu.IsNil(t.GetDispatcher()) && !lu.IsNil(t.GetDispatcher().GetApp()) {
		t.GetRpcContext().LogInfo("Sending CS response",
			"session_id", responseMsg.Head.SessionId,
			"client_sequence", responseMsg.Head.ClientSequence,
			"response_code", t.GetResponseCode())
		// 输出CSLOG
		logWriter := t.GetActorLogWriter()
		if logWriter != nil {
			fmt.Fprintf(logWriter, "%s >>>>>>>>>>>>>>>>>>>> Session: %d Sending: %s\nHead:%s\nBody:%s",
				now.Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.rpcDescriptor.Output().FullName(),
				pu.MessageReadableTextIndent(responseMsg.Head),
				pu.MessageReadableTextIndent(t.responseBody))
		} else if t.session.IsEnableActorLog() {
			t.session.InsertPendingActorLog(fmt.Sprintf("%s >>>>>>>>>>>>>>>>>>>> Session: %d Sending: %s\nHead:%s\nBody:%s",
				now.Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.rpcDescriptor.Output().FullName(),
				pu.MessageReadableTextIndent(responseMsg.Head),
				pu.MessageReadableTextIndent(t.responseBody)))
		}

		err = t.session.SendMessage(responseMsg)
		if err != nil {
			t.GetDispatcher().GetLogger().LogError("Failed to send CS response",
				"session_id", responseMsg.Head.SessionId,
				"client_sequence", responseMsg.Head.ClientSequence,
				"response_code", t.GetResponseCode(),
				"error", err.Error())

			t.GetDispatcher().OnSendMessageFailed(t.GetRpcContext(), &cd.DispatcherRawMessage{
				Type:     t.GetDispatcher().GetInstanceIdent(),
				Instance: responseMsg,
			}, responseMsg.Head.ServerSequence, err)
			return err
		}
	}

	return nil
}

func (t *TaskActionCSBase[RequestType, ResponseType]) CheckPermission() (int32, error) {
	if !t.GetImpl().AllowNoActor() && lu.IsNil(t.GetUser()) {
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

func (t *TaskActionCSBase[RequestType, ResponseType]) HookRun(startData *cd.DispatcherStartData) error {
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
	logWriter := t.GetActorLogWriter()
	if logWriter != nil {
		fmt.Fprintf(logWriter, "%s <<<<<<<<<<<<<<<<<<<< Session: %d Received: %s\nHead:%s\nBody:%s",
			t.GetSysNow().Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.requestHead.GetRpcRequest().GetTypeUrl(),
			pu.MessageReadableTextIndent(t.requestHead),
			pu.MessageReadableTextIndent(t.requestBody))
	} else if t.session.IsEnableActorLog() {
		t.session.InsertPendingActorLog(fmt.Sprintf("%s <<<<<<<<<<<<<<<<<<<< Session: %d Received: %s\nHead:%s\nBody:%s",
			t.GetSysNow().Format("2006-01-02 15:04:05.000"), t.session.GetSessionId(), t.requestHead.GetRpcRequest().GetTypeUrl(),
			pu.MessageReadableTextIndent(t.requestHead),
			pu.MessageReadableTextIndent(t.requestBody)))
	}

	user := t.GetUser()
	if !lu.IsNil(user) {
		// 每次执行任务前刷新
		user.RefreshLimit(t.GetRpcContext(), t.GetNow())
	}

	managerSet := libatapp.AtappGetModule[*router.RouterManagerSet](t.GetAwaitableContext().GetApp())
	userRouterManager := managerSet.GetManager(uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER)).(*UserRouterManager)
	var routerCache *UserRouterCache
	if !lu.IsNil(user) {
		userRouterManager = managerSet.GetManager(uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER)).(*UserRouterManager)
		routerCache = userRouterManager.GetCache(router.RouterObjectKey{
			TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
			ZoneID:   user.GetZoneId(),
			ObjectID: user.GetUserId(),
		})
		if routerCache != nil && (!routerCache.IsWritable() || routerCache.GetUserImpl() != user) {
			routerCache = nil
		}
	}

	// 限频
	var dispatcherOption *public_protocol_extension.DispatcherOptions = nil
	if startData.Option != nil && startData.Option.Option != nil {
		dispatcherOption = startData.Option.Option
	}
	for {
		if dispatcherOption == nil || lu.IsNil(user) {
			break
		}

		if dispatcherOption.GetDisable() {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_CS_PROTOCOL_FREQUENCY_LIMIT))
			return nil
		}

		if dispatcherOption.GetFrequencyLimit().GetFrequencyLimitMs() <= 0 ||
			dispatcherOption.GetFrequencyLimit().GetFrequencyLimitCount() <= 0 {
			break
		}

		frequencyLimitMs := dispatcherOption.GetFrequencyLimit().GetFrequencyLimitMs()
		frequencyLimitCount := dispatcherOption.GetFrequencyLimit().GetFrequencyLimitCount()
		frequencyLimitData := user.GetCSProtocolFrequencyLimit()
		rpcName := t.GetImpl().Name()
		currentTime := t.GetNow().UnixMilli()

		ring := frequencyLimitData[rpcName]
		if ring == nil {
			ring = &CSProtocolFrequencyRingBuffer{
				timestamps: make([]int64, frequencyLimitCount),
			}
			frequencyLimitData[rpcName] = ring
		}

		if ring.count < frequencyLimitCount {
			// 未满，写入 (index + count) % cap 位置
			pos := (ring.index + ring.count) % frequencyLimitCount
			ring.timestamps[pos] = currentTime
			ring.count++
			break
		}

		// 已满，index 指向最老的记录
		oldestTime := ring.timestamps[ring.index]
		if currentTime-oldestTime < 0 {
			// 时间回退，重置
			ring.timestamps[0] = currentTime
			ring.index = 0
			ring.count = 1
		} else if currentTime-oldestTime >= int64(frequencyLimitMs) {
			// 窗口已过期，覆盖最老记录并移动 index
			ring.timestamps[ring.index] = currentTime
			ring.index = (ring.index + 1) % frequencyLimitCount
		} else {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_CS_PROTOCOL_FREQUENCY_LIMIT))
			return nil
		}

		break
	}

	err := t.TaskActionBase.HookRun(startData)

	for {
		if dispatcherOption == nil {
			break
		}
		if !dispatcherOption.GetMarkFastSave() && !dispatcherOption.GetMarkWaitSave() {
			break
		}
		user = t.GetUser()
		if lu.IsNil(user) || routerCache == nil {
			break
		}
		user.RefreshLimit(t.GetRpcContext(), t.GetNow())
		if dispatcherOption.GetMarkWaitSave() {
			// 等待保存
			result := libatapp.AtappGetModule[*UserManager](t.GetAwaitableContext().GetApp()).Save(t.GetAwaitableContext(), user)
			if result.IsError() {
				t.GetDispatcher().GetLogger().LogError("Failed to save user data",
					"session_id", t.session.GetSessionId(),
					"error", result)
				return fmt.Errorf("failed to save user data: %v", result)
			}
		} else {
			// 快速保存
			managerSet.MarkFastSave(t.GetAwaitableContext(), userRouterManager, routerCache)
		}
		break
	}

	return err
}

func (t *TaskActionCSBase[RequestType, ResponseType]) GetTypeName() string {
	return "CS Task Action"
}
