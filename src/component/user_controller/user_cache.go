package atframework_component_user_controller

import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
)

type UserImpl interface {
	component_dispatcher.TaskActionCSUser

	BindSession(session *Session)
	UnbindSession(session *Session)
}

type UserCache struct {
	zoneId uint64
	userId uint64

	session *Session

	actorExecutor *component_dispatcher.ActorExecutor
}

func (u *UserCache) GetUserId() uint64 {
	return u.userId
}

func (u *UserCache) GetZoneId() uint64 {
	return u.zoneId
}

func (u *UserCache) GetSession() component_dispatcher.TaskActionCSSession {
	return u.session
}

func (u *UserCache) GetActorExecutor() *component_dispatcher.ActorExecutor {
	return u.actorExecutor
}

func (u *UserCache) BindSession(session *Session) {
	if u.session == session {
		return
	}

	if session == nil {
		u.UnbindSession(u.session)
		return
	}

	u.session = session
}

func (u *UserCache) UnbindSession(session *Session) {
	if u.session != session {
		return
	}

	u.session = session

	// TODO: 触发登出保存
}
