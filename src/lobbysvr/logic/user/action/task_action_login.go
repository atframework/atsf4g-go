// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"
	"log/slog"
	"strconv"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type TaskActionLogin struct {
	cd.TaskActionCSBase[*service_protocol.CSLoginReq, *service_protocol.SCLoginRsp]

	isNewPlayer bool
}

func (t *TaskActionLogin) Name() string {
	return "TaskActionLogin"
}

func (t *TaskActionLogin) AllowNoActor() bool {
	return true
}

func (t *TaskActionLogin) Run(_startData *cd.DispatcherStartData) error {
	t.GetLogger().Info("TaskActionLogin Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", t.GetSession().GetSessionId()),
	)

	request_body := t.GetRequestBody()

	userId := request_body.GetUserId()
	if userId == 0 {
		userIdFromOpenId, err := strconv.ParseUint(request_body.GetOpenId(), 10, 64)
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
			t.GetLogger().Warn("invalid openid id", "open_id", request_body.GetOpenId(), "error", err)
			return nil
		}
		userId = userIdFromOpenId
	}

	zoneId := t.GetDispatcher().GetApp().GetLogicId()

	csSession := t.GetSession()
	if csSession == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session is required", "zone_id", zoneId, "user_id", userId)
		return nil
	}
	session := csSession.(*uc.Session)
	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session conversion failed", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	// 老用户登入锁
	user := uc.UserManagerFindUserAs[*data.User](uc.GlobalUserManager, zoneId, userId)
	if !t.checkExistedUser(user) {
		return nil
	}

	// 登入鉴权
	_, loginCode := uc.UserGetAuthDataFromFile(t.GetRpcContext(), zoneId, userId)
	if loginCode == "" || loginCode != request_body.GetLoginCode() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetLogger().Warn("invalid login code", "zone_id", zoneId, "user_id", userId, "code", loginCode, "req", request_body.GetLoginCode())
		return nil
	}

	// TODO: 登入鉴权Token有效期

	// 如果是在线用户，走替换Session流程
	if user != nil && user.IsWriteable() {
		t.replaceSession(user, session)
		return nil
	}

	loginTb, result := db.DatabaseTableLoginLoadWithZoneIdUserId(t.GetRpcContext(), zoneId, userId)
	if result.IsError() {
		if result.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			loginTb = &private_protocol_pbdesc.DatabaseTableLogin{
				OpenId:         request_body.GetOpenId(),
				UserId:         userId,
				ZoneId:         zoneId,
				RouterServerId: 0,
				RouterVersion:  0,
			}
		} else {
			if result.GetResponseCode() < 0 {
				t.SetResponseCode(result.GetResponseCode())
			} else {
				t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
			}
			if result.Error != nil {
				result.LogError(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
			} else {
				result.LogWarn(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
			}
			return nil
		}
	}
	loginTbVersion := loginTb.RouterVersion
	t.mergeLoginInfo(loginTb, userId)

	user, result = uc.UserManagerCreateUserAs(
		t.GetRpcContext(), uc.GlobalUserManager, zoneId, userId, request_body.GetOpenId(),
		loginTb, loginTbVersion, func(user *data.User) cd.RpcResult {
			if user == nil {
				return cd.CreateRpcResultError(
					fmt.Errorf("user is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
			}

			if !user.TryLockLoginTask(t.GetTaskId()) {
				return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
			}
			t.SetUser(user)

			return cd.CreateRpcResultOk()
		},
		// 不需要这里解锁，整个task执行完后会解锁的
		nil)
	if result.IsError() {
		if result.GetResponseCode() < 0 {
			t.SetResponseCode(result.GetResponseCode())
		} else {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
		}
		result.LogError(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	t.isNewPlayer = user.GetLoginVersion() <= 1

	// 数据复制
	// proto.Reset(user.MutableClientInfo())
	if request_body.GetClientInfo() != nil {
		*user.MutableClientInfo() = *request_body.GetClientInfo()
	}

	// session绑定
	t.GetSession().BindUser(t.GetRpcContext(), user)

	// 登入初始化
	user.LoginInit(t.GetRpcContext())

	return nil
}

func (t *TaskActionLogin) mergeLoginInfo(loginTb *private_protocol_pbdesc.DatabaseTableLogin, userId uint64) {
	request_body := t.GetRequestBody()

	loginTb.LoginCode = request_body.GetLoginCode()
	loginTb.LoginCodeExpired = t.GetNow().Unix() + int64(20*60)
	loginTb.Account = &private_protocol_pbdesc.AccountInformation{
		AccountType: request_body.GetAccount().GetAccountType(),
		// Access: request_body.Account.Access,
		Profile: &public_protocol_pbdesc.DUserProfile{
			OpenId: request_body.GetOpenId(),
			UserId: userId,
		},
		ChannelId: request_body.GetAccount().GetChannelId(),
	}
}

func (t *TaskActionLogin) checkExistedUser(user *data.User) bool {
	if user == nil {
		return true
	}

	if !user.TryLockLoginTask(t.GetTaskId()) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		t.GetLogger().Warn("user is logining in another task", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "login_task_id", user.GetLoginTaskId())
		return false
	}
	t.SetUser(user)

	if user.IsWriteable() && user.GetSession() == t.GetSession() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_ALREADY_ONLINE)
		t.GetLogger().Warn("user already login", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	return true
}

func (t *TaskActionLogin) replaceSession(user *data.User, session *uc.Session) bool {
	if user == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("user is required")
		return false
	}

	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session is required", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	// 先解锁旧的Session
	user.BindSession(t.GetRpcContext(), session)
	return true
}

func (t *TaskActionLogin) OnSuccess() {
	response_body := t.MutableResponseBody()

	userImpl := t.GetUser()
	if userImpl == nil {
		return
	}

	user, ok := userImpl.(*data.User)
	if !ok || user == nil {
		return
	}

	response_body.ZoneId = user.GetZoneId()
	response_body.VersionType = uint32(user.GetAccountInfo().GetVersionType())

	// TODO: 配置
	response_body.HeartbeatInterval = 120

	response_body.IsNewUser = t.isNewPlayer

	// 事件和刷新
	user.RefreshLimit(t.GetRpcContext(), t.GetNow())

	// 登入过程中产生的脏数据不需要推送
	user.CleanupClientDirtyCache(t.GetRpcContext())
}

func (t *TaskActionLogin) OnComplete() {
	userImpl := t.GetUser()
	if userImpl == nil {
		return
	}

	user, ok := userImpl.(*data.User)
	if !ok || user == nil {
		t.GetLogger().Warn("Task user can not convert to data.User", "task_id", t.GetTaskId(), "task_name", t.Name())
		return
	}

	user.UnlockLoginTask(t.GetTaskId())
}
