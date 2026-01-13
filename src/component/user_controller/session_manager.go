package atframework_component_user_controller

import (
	"context"
	"fmt"
	"sync"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	libatapp "github.com/atframework/libatapp-go"
)

type noCopy struct{}

type SessionManager struct {
	libatapp.AppModuleBase

	sessionLock sync.RWMutex
	sessions    map[uint64]map[uint64]*Session
}

func init() {
	var _ libatapp.AppModuleImpl = (*SessionManager)(nil)
}

func (m *SessionManager) Init(parent context.Context) error {
	return nil
}

func (m *SessionManager) Name() string {
	return "SessionManager"
}

func CreateSessionManager(app libatapp.AppImpl) *SessionManager {
	return &SessionManager{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
		sessions:      make(map[uint64]map[uint64]*Session),
	}
}

func (sm *SessionManager) GetSession(key *SessionKey) *Session {
	if key == nil {
		return nil
	}

	sm.sessionLock.RLock()
	defer sm.sessionLock.RUnlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if session, ok := nodeSessions[key.SessionId]; ok {
			return session
		}
	}

	return nil
}

func (sm *SessionManager) CreateSession(ctx cd.RpcContext, key SessionKey, handle SessionNetworkHandleImpl) *Session {
	if handle == nil {
		return nil
	}

	sm.sessionLock.RLock()
	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if session, ok := nodeSessions[key.SessionId]; ok {
			sm.sessionLock.RUnlock()
			return session
		}
	}

	sm.sessionLock.RUnlock()

	session := CreateSession(key, handle)

	sm.sessionLock.Lock()
	defer sm.sessionLock.Unlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		nodeSessions[key.SessionId] = session
	} else {
		sm.sessions[key.NodeId] = map[uint64]*Session{
			key.SessionId: session,
		}
	}

	outputLog := fmt.Sprintf("%s >>>>>>>>>>>>>>>>>>>> Create Session: %d", ctx.GetSysNow().Format("2006-01-02 15:04:05.000"), session.GetSessionId())
	if session.IsEnableActorLog() {
		session.InsertPendingActorLog(outputLog)
	}
	ctx.LogDebug(outputLog)

	ctx.LogInfo("session created", "session_node_id", key.NodeId, "session_id", key.SessionId)

	// TODO: 添加session超时检查

	return session
}

func (sm *SessionManager) RemoveSession(ctx cd.RpcContext, key *SessionKey, reason int32, reasonMessage string) {
	if key == nil {
		return
	}

	session := sm.GetSession(key)
	if session == nil {
		return
	}

	session.Close(ctx, reason, reasonMessage)

	sm.sessionLock.Lock()
	defer sm.sessionLock.Unlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if _, ok := nodeSessions[key.SessionId]; ok {
			ctx.LogInfo("session removed", "session_node_id", key.NodeId, "session_id", key.SessionId)

			delete(nodeSessions, key.SessionId)
			if len(nodeSessions) == 0 {
				delete(sm.sessions, key.NodeId)
			}
		}
	}

	// TODO: 移除session超时检查
}
