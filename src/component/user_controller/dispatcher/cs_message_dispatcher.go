package atframework_component_user_controller

import (
	"fmt"
	"strings"

	lu "github.com/atframework/atframe-utils-go/lang_utility"

	libatapp "github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"

	uc "github.com/atframework/atsf4g-go/component-user_controller"
	uc_act "github.com/atframework/atsf4g-go/component-user_controller/action"
)

type SessionNetworkWebsocketHandle struct {
	dispatcher     *cd.WebSocketMessageDispatcher
	networkSession *cd.WebSocketSession

	cacheRemoteAddr string
}

func (h *SessionNetworkWebsocketHandle) GetDispatcher() cd.DispatcherImpl {
	if h == nil {
		return nil
	}
	return h.dispatcher
}

func (h *SessionNetworkWebsocketHandle) SendMessage(msg *public_protocol_extension.CSMsg) error {
	if h == nil {
		return fmt.Errorf("session network handle is nil")
	}
	// Implement the logic to send a message over the WebSocket
	return h.dispatcher.WriteMessage(h.networkSession, msg)
}

func (h *SessionNetworkWebsocketHandle) SetAuthorized(authorized bool) {
	if h == nil {
		return
	}

	if lu.IsNil(h.networkSession) {
		return
	}

	// Implement the logic to set the authorization state
	h.networkSession.Authorized = authorized
}

func (h *SessionNetworkWebsocketHandle) Close(ctx cd.RpcContext, reason int32, reasonMessage string) {
	if h == nil {
		ctx.LogError("SessionNetworkWebsocketHandle is nil")
		return
	}

	if h.dispatcher == nil {
		ctx.LogError("SessionNetworkWebsocketHandle.dispatcher is nil")
		return
	}

	// Implement the logic to close the WebSocket session
	h.dispatcher.Close(ctx, h.networkSession, int(reason), reasonMessage)
}

func (h *SessionNetworkWebsocketHandle) GetRemoteAddr() string {
	if h == nil {
		return ""
	}

	if h.cacheRemoteAddr != "" {
		return h.cacheRemoteAddr
	}

	if h.networkSession != nil && h.networkSession.Connection != nil {
		h.cacheRemoteAddr = h.networkSession.Connection.RemoteAddr().String()
	}

	return h.cacheRemoteAddr
}

func WebsocketDispatcherCreateCSMessage(owner libatapp.AppImpl, webServerConfigurePath string, webSocketServerConfigurePath string) *cd.WebSocketMessageDispatcher {
	d := cd.CreateCSMessageWebsocketDispatcher(owner, webServerConfigurePath, webSocketServerConfigurePath)
	if lu.IsNil(d) {
		return nil
	}

	d.SetOnNewSession(func(ctx cd.RpcContext, session *cd.WebSocketSession) error {
		// WS消息都是本地监听，所以NodeId都是自己的AppId
		sessionKey := uc.CreateSessionKey(owner.GetAppId(), session.SessionId)
		session.PrivateData = uc.GlobalSessionManager.CreateSession(ctx, sessionKey, &SessionNetworkWebsocketHandle{
			dispatcher:     d,
			networkSession: session,
		})

		return nil
	})

	d.SetOnRemoveSession(func(ctx cd.RpcContext, session *cd.WebSocketSession) {
		// WS消息都是本地监听，所以NodeId都是自己的AppId
		sessionKey := uc.CreateSessionKey(owner.GetAppId(), session.SessionId)

		uc_act.RemoveSessionAndMaybeLogoutUser(d, ctx, &sessionKey)
	})

	return d
}

func WebsocketDispatcherFindSessionFromMessage(
	rd cd.DispatcherImpl, msg *cd.DispatcherRawMessage,
	privateData interface{},
) *uc.Session {
	if !lu.IsNil(privateData) {
		switch privateData.(type) {
		case *cd.WebSocketSession:
			s := privateData.(*cd.WebSocketSession).PrivateData
			if lu.IsNil(s) {
				return nil
			}

			return s.(*uc.Session)
		}
	}

	return nil
}

type FindCSMessageSession = func(
	rd cd.DispatcherImpl, msg *cd.DispatcherRawMessage,
	privateData interface{},
) *uc.Session

func RegisterCSMessageAction[RequestType proto.Message, ResponseType proto.Message](
	rd cd.DispatcherImpl, findSessionFn FindCSMessageSession,
	serviceDescriptor protoreflect.ServiceDescriptor, rpcFullName string,
	createFn func(cd.TaskActionCSBase[RequestType, ResponseType]) cd.TaskActionImpl,
) error {
	if lu.IsNil(serviceDescriptor) {
		rd.GetLogger().Error("service descriptor is nil", "rpc_name", rpcFullName)
		return fmt.Errorf("service descriptor not match rpc full name")
	}

	lastIndex := strings.LastIndex(rpcFullName, ".")
	if lastIndex < 0 || string(serviceDescriptor.FullName()) != rpcFullName[:lastIndex] {
		rd.GetLogger().Error("service descriptor not match rpc full name", "rpc_name", rpcFullName, "service_name", serviceDescriptor.FullName())
		return fmt.Errorf("service descriptor not match rpc full name")
	}

	methodDesc := serviceDescriptor.Methods().ByName(protoreflect.Name(rpcFullName[lastIndex+1:]))
	if lu.IsNil(methodDesc) {
		rd.GetLogger().Error("method descriptor not found in service", "rpc_name", rpcFullName, "service_name", serviceDescriptor.FullName())
		return fmt.Errorf("method descriptor not found in service")
	}

	creator := func(rd cd.DispatcherImpl, startData *cd.DispatcherStartData) (cd.TaskActionImpl, error) {
		session := findSessionFn(rd, startData.Message, startData.PrivateData)
		if lu.IsNil(session) {
			rd.GetLogger().Warn("session not found for CS message", "rpc_name", rpcFullName)
			return nil, fmt.Errorf("session not found for CS message")
		}

		// 创建实际类型
		return createFn(cd.CreateTaskActionCSBase[RequestType, ResponseType](rd, session, methodDesc)), nil
	}

	return rd.RegisterAction(serviceDescriptor, rpcFullName, creator)
}
