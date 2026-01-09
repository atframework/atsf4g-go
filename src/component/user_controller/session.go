package atframework_component_user_controller

import (
	"fmt"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	libatapp "github.com/atframework/libatapp-go"

	config "github.com/atframework/atsf4g-go/component-config"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
)

type SessionNetworkHandleImpl interface {
	GetDispatcher() cd.DispatcherImpl

	SendMessage(*public_protocol_extension.CSMsg) error
	SetAuthorized(bool)
	Close(ctx cd.RpcContext, reason int32, reasonMessage string)
	GetRemoteAddr() string
}

type SessionKey struct {
	NodeId    uint64
	SessionId uint64
}

type SessionImpl interface {
	cd.TaskActionCSSession

	GetKey() *SessionKey
}

type Session struct {
	key SessionKey

	user UserImpl

	networkHandle SessionNetworkHandleImpl
	networkClosed bool

	sessionSequenceAllocator uint64

	enableActorLog bool
	pendingCsLog   []string
}

func CreateSessionKey(nodeId uint64, sessionId uint64) SessionKey {
	return SessionKey{
		NodeId:    nodeId,
		SessionId: sessionId,
	}
}

func CreateSession(key SessionKey, handle SessionNetworkHandleImpl) *Session {
	if handle != nil && lu.IsNil(handle) {
		handle = nil
	}
	return &Session{
		key:                      key,
		user:                     nil,
		networkHandle:            handle,
		networkClosed:            false,
		sessionSequenceAllocator: 0,
		enableActorLog:           config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetUser().GetEnableSessionActorLog(),
	}
}

func (s *Session) GetKey() *SessionKey {
	return &s.key
}

func (s *Session) Close(ctx cd.RpcContext, reason int32, reasonMessage string) {
	if !s.networkClosed {
		s.networkClosed = true
		s.networkHandle.Close(ctx, reason, reasonMessage)
	}

	// 解绑User
	if !lu.IsNil(s.user) {
		s.user.UnbindSession(ctx, s)
	}

	// 处理日志
	if len(s.pendingCsLog) > 0 {
		writer, _ := libatapp.NewLogBufferedRotatingWriter(ctx, fmt.Sprintf("%s/%%F/session-unflush/%d.%%N.log", config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetServer().GetLogPath(), s.key.SessionId),
			"", config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorLogSize(),
			uint32(config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorLogRotate()),
			config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetActorAutoFlush().AsDuration(), 0)
		if writer != nil {
			s.FlushPendingActorLog(writer)
		}
		writer.Close()
	}
}

func (s *Session) GetSessionId() uint64 {
	return s.key.SessionId
}

func (s *Session) GetSessionNodeId() uint64 {
	return s.key.NodeId
}

func (s *Session) AllocSessionSequence() uint64 {
	if !lu.IsNil(s.user) {
		seq := s.user.AllocSessionSequence()
		if seq > s.sessionSequenceAllocator {
			s.sessionSequenceAllocator = seq
		}
		return seq
	}

	s.sessionSequenceAllocator++
	return s.sessionSequenceAllocator
}

func (s *Session) GetUser() cd.TaskActionCSUser {
	if s.user == nil {
		return nil
	}

	return s.user
}

func (s *Session) BindUser(ctx cd.RpcContext, bindUser cd.TaskActionCSUser) {
	if s.user == bindUser {
		return
	}

	if lu.IsNil(bindUser) {
		if !lu.IsNil(s.user) {
			s.user.UnbindSession(ctx, s)
		}
		s.user = nil
		return
	}

	convertUser, ok := bindUser.(UserImpl)
	if !ok {
		return
	}

	oldUser := s.user

	// 覆盖旧绑定,必须先设置成员变量再触发关联绑定，以解决重入问题
	s.user = convertUser
	convertUser.BindSession(ctx, s)

	if !lu.IsNil(s.user) && !lu.IsNil(s.networkHandle) {
		s.networkHandle.SetAuthorized(true)
	}

	// 关联解绑
	if !lu.IsNil(oldUser) {
		oldUser.UnbindSession(ctx, s)
	}
}

func (s *Session) UnbindUser(ctx cd.RpcContext, bindUser cd.TaskActionCSUser) {
	if !lu.IsNil(bindUser) && s.user != bindUser {
		return
	}

	oldUser := s.user

	s.user = nil

	// 关联解绑
	if !lu.IsNil(oldUser) {
		oldUser.UnbindSession(ctx, s)
	}
}

func (s *Session) GetDispatcher() cd.DispatcherImpl {
	if lu.IsNil(s.networkHandle) {
		return nil
	}

	return s.networkHandle.GetDispatcher()
}

func (s *Session) SendMessage(msg *public_protocol_extension.CSMsg) error {
	if lu.IsNil(s.networkHandle) {
		return fmt.Errorf("network handle is already closed")
	}
	return s.networkHandle.SendMessage(msg)
}

func (s *Session) GetNetworkHandle() SessionNetworkHandleImpl {
	return s.networkHandle
}

func (s *Session) GetActorLogWriter() libatapp.LogWriter {
	user := s.GetUser()
	if lu.IsNil(user) {
		return nil
	}
	return user.GetCsActorLogWriter()
}

func (s *Session) IsEnableActorLog() bool {
	return s.enableActorLog
}

func (s *Session) InsertPendingActorLog(content string) {
	s.pendingCsLog = append(s.pendingCsLog, content)
}

func (s *Session) FlushPendingActorLog(logWriter libatapp.LogWriter) {
	for _, log := range s.pendingCsLog {
		fmt.Fprint(logWriter, log)
	}
	s.pendingCsLog = nil
}
