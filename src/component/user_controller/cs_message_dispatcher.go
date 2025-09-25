package atframework_component_user_controller

import (
	"fmt"
	"strings"

	libatapp "github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type SessionNetworkWebsocketHandle struct {
	dispatcher     *component_dispatcher.WebSocketMessageDispatcher
	networkSession *component_dispatcher.WebSocketSession

	cacheRemoteAddr string
}

func (h *SessionNetworkWebsocketHandle) SendMessage(msg *public_protocol_extension.CSMsg) error {
	// Implement the logic to send a message over the WebSocket
	return h.dispatcher.WriteMessage(h.networkSession, msg)
}

func (h *SessionNetworkWebsocketHandle) SetAuthorized(authorized bool) {
	// Implement the logic to set the authorization state
	h.networkSession.Authorized = authorized
}

func (h *SessionNetworkWebsocketHandle) Close(reason int32, reasonMessage string) {
	// Implement the logic to close the WebSocket session
	h.dispatcher.Close(h.networkSession, int(reason), reasonMessage)
}

func (h *SessionNetworkWebsocketHandle) GetRemoteAddr() string {
	if h.cacheRemoteAddr != "" {
		return h.cacheRemoteAddr
	}

	if h.networkSession != nil && h.networkSession.Connection != nil {
		h.cacheRemoteAddr = h.networkSession.Connection.RemoteAddr().String()
	}

	return h.cacheRemoteAddr
}

func WebsocketDispatcherCreateCSMessage(owner libatapp.AppImpl) *component_dispatcher.WebSocketMessageDispatcher {
	d := component_dispatcher.CreateCSMessageWebsocketDispatcher(owner)
	if d == nil {
		return nil
	}

	d.SetOnNewSession(func(session *component_dispatcher.WebSocketSession) error {
		// WS消息都是本地监听，所以NodeId都是自己的AppId
		sessionKey := CreateSessionKey(owner.GetAppId(), session.SessionId)
		session.PrivateData = GlobalSessionManager.CreateSession(sessionKey, &SessionNetworkWebsocketHandle{
			dispatcher:     d,
			networkSession: session,
		})

		return nil
	})

	d.SetOnRemoveSession(func(session *component_dispatcher.WebSocketSession) {
		// WS消息都是本地监听，所以NodeId都是自己的AppId
		sessionKey := CreateSessionKey(owner.GetAppId(), session.SessionId)

		// TODO: 接入gateway层reason
		GlobalSessionManager.RemoveSession(&sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_SESSION_NOT_FOUND), "closed by client")
	})

	return d
}

func WebsocketDispatcherFindSessionFromMessage(
	rd component_dispatcher.DispatcherImpl, msg *component_dispatcher.DispatcherRawMessage,
	privateData interface{},
) *Session {
	if privateData != nil {
		switch privateData.(type) {
		case *component_dispatcher.WebSocketSession:
			s := privateData.(*component_dispatcher.WebSocketSession).PrivateData
			if s == nil {
				return nil
			}

			return s.(*Session)
		}
	}

	return nil
}

type FindCSMessageSession = func(
	rd component_dispatcher.DispatcherImpl, msg *component_dispatcher.DispatcherRawMessage,
	privateData interface{},
) *Session

func RegisterCSMessageAction[RequestType proto.Message, ResponseType proto.Message](
	rd component_dispatcher.DispatcherImpl, findSessionFn FindCSMessageSession,
	serviceDescriptor protoreflect.ServiceDescriptor, rpcFullName string,
	requestBody RequestType,
	responseBody ResponseType,
) error {
	if serviceDescriptor == nil {
		rd.GetApp().GetLogger().Error("service descriptor is nil", "rpc_name", rpcFullName)
		return fmt.Errorf("service descriptor not match rpc full name")
	}

	lastIndex := strings.LastIndex(rpcFullName, ".")
	if lastIndex < 0 || string(serviceDescriptor.FullName()) != rpcFullName[:lastIndex] {
		rd.GetApp().GetLogger().Error("service descriptor not match rpc full name", "rpc_name", rpcFullName, "service_name", serviceDescriptor.FullName())
		return fmt.Errorf("service descriptor not match rpc full name")
	}

	methodDesc := serviceDescriptor.Methods().ByName(protoreflect.Name(rpcFullName[lastIndex+1:]))
	if methodDesc == nil {
		rd.GetApp().GetLogger().Error("method descriptor not found in service", "rpc_name", rpcFullName, "service_name", serviceDescriptor.FullName())
		return fmt.Errorf("method descriptor not found in service")
	}

	creator := func(rd component_dispatcher.DispatcherImpl, startData *component_dispatcher.DispatcherStartData) (component_dispatcher.TaskActionImpl, error) {
		session := findSessionFn(rd, startData.Message, startData.PrivateData)
		if session == nil {
			rd.GetApp().GetLogger().Warn("session not found for CS message", "rpc_name", rpcFullName)
			return nil, fmt.Errorf("session not found for CS message")
		}

		// TODO: 创建实际类型
		// component_dispatcher.CreateTaskActionCSBase[RequestType, ResponseType](rd, session, methodDesc, requestBody, responseBody)
		return nil, nil
	}

	return rd.RegisterAction(serviceDescriptor, rpcFullName, creator)
}
