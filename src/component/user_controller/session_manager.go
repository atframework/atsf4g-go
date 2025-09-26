package atframework_component_user_controller

import (
	"sync"
)

type noCopy struct{}

type SessionManager struct {
	_ noCopy

	sessionLock sync.RWMutex
	sessions    map[uint64]*map[uint64]*Session
}

func createSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[uint64]*map[uint64]*Session),
	}
}

var GlobalSessionManager = createSessionManager()

func (sm *SessionManager) GetSession(key *SessionKey) *Session {
	if key == nil {
		return nil
	}

	sm.sessionLock.RLock()
	defer sm.sessionLock.RUnlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if session, ok := (*nodeSessions)[key.SessionId]; ok {
			return session
		}
	}

	return nil
}

func (sm *SessionManager) CreateSession(key SessionKey, handle SessionNetworkHandleImpl) *Session {
	if handle == nil {
		return nil
	}

	sm.sessionLock.RLock()
	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if session, ok := (*nodeSessions)[key.SessionId]; ok {
			sm.sessionLock.RUnlock()
			return session
		}
	}

	sm.sessionLock.RUnlock()

	session := CreateSession(key, handle)

	sm.sessionLock.Lock()
	defer sm.sessionLock.Unlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		(*nodeSessions)[key.SessionId] = session
	} else {
		sm.sessions[key.NodeId] = &map[uint64]*Session{
			key.SessionId: session,
		}
	}

	return session
}

func (sm *SessionManager) RemoveSession(key *SessionKey, reason int32, reasonMessage string) {
	if key == nil {
		return
	}

	session := sm.GetSession(key)
	if session == nil {
		return
	}

	session.Close(reason, reasonMessage)

	sm.sessionLock.Lock()
	defer sm.sessionLock.Unlock()

	if nodeSessions, ok := sm.sessions[key.NodeId]; ok {
		if _, ok := (*nodeSessions)[key.SessionId]; ok {
			delete(*nodeSessions, key.SessionId)
			if len(*nodeSessions) == 0 {
				delete(sm.sessions, key.NodeId)
			}
		}
	}
}
