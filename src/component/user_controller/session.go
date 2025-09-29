package atframework_component_user_controller

import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
)

type SessionNetworkHandleImpl interface {
	GetDispatcher() component_dispatcher.DispatcherImpl

	SendMessage(*public_protocol_extension.CSMsg) error
	SetAuthorized(bool)
	Close(reason int32, reasonMessage string)
	GetRemoteAddr() string
}

type SessionKey struct {
	NodeId    uint64
	SessionId uint64
}

type SessionImpl interface {
	component_dispatcher.TaskActionCSSession

	GetKey() SessionKey
}

type Session struct {
	key SessionKey

	user UserImpl

	networkHandle SessionNetworkHandleImpl
	networkClosed bool

	sessionSequenceAllocator uint64
}

func CreateSessionKey(nodeId uint64, sessionId uint64) SessionKey {
	return SessionKey{
		NodeId:    nodeId,
		SessionId: sessionId,
	}
}

func CreateSession(key SessionKey, handle SessionNetworkHandleImpl) *Session {
	return &Session{
		key:                      key,
		user:                     nil,
		networkHandle:            handle,
		networkClosed:            false,
		sessionSequenceAllocator: 0,
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

	// 解绑User
	if s.user != nil {
		s.user.UnbindSession(s)
	}
}

func (s *Session) GetSessionId() uint64 {
	return s.key.SessionId
}

func (s *Session) GetSessionNodeId() uint64 {
	return s.key.NodeId
}

func (s *Session) AllocSessionSequence() uint64 {
	s.sessionSequenceAllocator++
	return s.sessionSequenceAllocator
}

func (s *Session) GetUser() component_dispatcher.TaskActionCSUser {
	return s.user
}

func (s *Session) BindUser(bindUser component_dispatcher.TaskActionCSUser) {
	if s.user == bindUser {
		return
	}

	convertUser, ok := bindUser.(UserImpl)
	if !ok {
		return
	}

	if s.user != nil {
		s.user.UnbindSession(s)
	}

	s.user = convertUser
	convertUser.BindSession(s)
}

func (s *Session) SendMessage(msg *public_protocol_extension.CSMsg) error {
	return s.networkHandle.SendMessage(msg)
}

func (s *Session) GetNetworkHandle() SessionNetworkHandleImpl {
	return s.networkHandle
}
