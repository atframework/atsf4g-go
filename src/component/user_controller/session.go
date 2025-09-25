package atframework_component_user_controller

import (
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
)

type SessionNetworkHandleImpl interface {
	SendMessage(*public_protocol_extension.CSMsg) error
	SetAuthorized(bool)
	Close(reason int32, reasonMessage string)
	GetRemoteAddr() string
}

type SessionKey struct {
	NodeId    uint64
	SessionId uint64
}

type Session struct {
	key SessionKey

	user *UserCache

	networkHandle SessionNetworkHandleImpl
	networkClosed bool
}

func CreateSessionKey(nodeId uint64, sessionId uint64) SessionKey {
	return SessionKey{
		NodeId:    nodeId,
		SessionId: sessionId,
	}
}

func CreateSession(key SessionKey, handle SessionNetworkHandleImpl) *Session {
	return &Session{
		key:           key,
		networkHandle: handle,
	}
}

func (s *Session) GetKey() SessionKey {
	return s.key
}

func (s *Session) Close(reason int32, reasonMessage string) {
	if !s.networkClosed {
		s.networkClosed = true
		s.networkHandle.Close(reason, reasonMessage)
	}

	// TODO: 解绑User
}
