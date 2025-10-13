// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"log/slog"
	"strconv"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskActionLogin struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSLoginReq, *service_protocol.SCLoginRsp]

	isNewPlayer bool
}

func (t *TaskActionLogin) Name() string {
	return "TaskActionLogin"
}

func (t *TaskActionLogin) AllowNoActor() bool {
	return true
}

func (t *TaskActionLogin) Run(_startData *component_dispatcher.DispatcherStartData) error {
	t.GetDispatcher().GetApp().GetDefaultLogger().Info("TaskActionLoginAuth Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", t.GetSession().GetSessionId()),
	)

	request_body := t.GetRequestBody()

	userId := request_body.GetUserId()
	if userId == 0 {
		userIdFromOpenId, err := strconv.ParseUint(request_body.GetOpenId(), 10, 64)
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
			t.GetDefaultLogger().Warn("invalid openid id", "open_id", request_body.GetOpenId(), "error", err)
			return nil
		}
		userId = userIdFromOpenId
	}

	zoneId := request_body.GetZoneId()
	if zoneId == 0 {
		zoneId = uint32(1)
	}

	csSession := t.GetSession()
	if csSession == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetDefaultLogger().Error("session is required", "zone_id", zoneId, "user_id", userId)
		return nil
	}
	session := csSession.(*uc.Session)
	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetDefaultLogger().Error("session conversion failed", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	// 老用户登入锁
	user := uc.UserManagerFindUserAs[*data.User](uc.GlobalUserManager, zoneId, userId)
	if !t.checkExistedUser(user) {
		return nil
	}

	// 已登入用户的登入互斥
	defer func() {
		if user != nil {
			user.UnlockLoginTask(t.GetTaskId())
		}
	}()

	// 登入鉴权
	_, loginCode := uc.UserGetAuthDataFromFile(t.GetRpcContext(), zoneId, userId)
	if loginCode == "" || loginCode != request_body.GetLoginCode() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetDefaultLogger().Warn("invalid login code", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	// TODO: 登入鉴权Token有效期

	// 如果是在线用户，走替换Session流程
	if user != nil && user.IsWriteable() {
		t.replaceSession(user, session)
		return nil
	}

	// TODO: 临时信任session，后续加入token验证和User绑定
	t.GetSession().BindUser(t.GetRpcContext(), &data.User{})

	return nil
}

func (t *TaskActionLogin) checkExistedUser(user *data.User) bool {
	if user == nil {
		return true
	}

	if !user.TryLockLoginTask(t.GetTaskId()) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		t.GetDefaultLogger().Warn("user is logining in another task", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "login_task_id", user.GetLoginTaskId())
		return false
	}

	if user.IsWriteable() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_ALREADY_ONLINE)
		t.GetDefaultLogger().Warn("user already login", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	return true
}

func (t *TaskActionLogin) replaceSession(user *data.User, session *uc.Session) bool {
	if user == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetDefaultLogger().Error("user is required")
		return false
	}

	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetDefaultLogger().Error("session is required", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	// 先解锁旧的Session
	user.BindSession(user, t.GetRpcContext(), session)
	return true
}

func (t *TaskActionLogin) OnSuccess() {
	response_body := t.MutableResponseBody()

	user := t.GetUser()
	if user == nil {
		return
	}

	response_body.ZoneId = user.GetZoneId()
	response_body.VersionType = uint32(public_protocol_pbdesc.EnVersionType_EN_VERSION_DEFAULT)

	// TODO: 配置
	response_body.HeartbeatInterval = 120

	response_body.IsNewUser = t.isNewPlayer
}
