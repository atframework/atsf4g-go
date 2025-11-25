package lobbysvr_logic_user_action

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

	db "github.com/atframework/atsf4g-go/component-db"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type TaskActionLoginAuth struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSLoginAuthReq, *service_protocol.SCLoginAuthRsp]
}

func (t *TaskActionLoginAuth) Name() string {
	return "TaskActionLoginAuth"
}

func (t *TaskActionLoginAuth) AllowNoActor() bool {
	return true
}

func (t *TaskActionLoginAuth) Run(_startData *component_dispatcher.DispatcherStartData) error {
	t.GetDispatcher().GetLogger().Info("TaskActionLoginAuth Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", t.GetSession().GetSessionId()),
	)

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	userId, err := strconv.ParseUint(request_body.GetOpenId(), 10, 64)
	if err != nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		t.GetLogger().Warn("invalid openid id", "open_id", request_body.GetOpenId(), "error", err)
		return nil
	}

	zoneId := t.GetDispatcher().GetApp().GetLogicId()

	user := uc.UserManagerFindUserAs[*data.User](t.GetRpcContext(), t.GetDispatcher().GetApp(), zoneId, userId)
	if !t.checkExistedUser(user) {
		return nil
	}

	// 已登入用户的登入互斥
	defer func() {
		if user != nil {
			user.UnlockLoginTask(t.GetTaskId())
		}
	}()

	authTable, _ := db.DatabaseTableAccessLoadWithZoneIdUserId(t.GetAwaitableContext(), zoneId, userId)
	accessSecret := ""
	if authTable != nil {
		accessSecret = authTable.GetAccessSecret()
	}
	if accessSecret != "" && accessSecret != "*" && accessSecret != request_body.GetAccount().GetAccess() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetLogger().Warn("user already login", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Warn("generate login code failed", "zone_id", zoneId, "user_id", userId, "error", err)
		return nil
	}
	loginCode := strings.Replace(uuid.String(), "-", "", -1)

	if accessSecret == "" {
		accessSecret = "*"
	}

	table := private_protocol_pbdesc.DatabaseTableAccess{
		ZoneId:       zoneId,
		UserId:       userId,
		AccessSecret: accessSecret,
		LoginCode:    loginCode,
	}
	rpcErr := db.DatabaseTableAccessUpdateZoneIdUserId(t.GetAwaitableContext(), &table)
	if rpcErr.IsError() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Warn("update login code failed", "zone_id", zoneId, "user_id", userId, "error", rpcErr)
		return nil
	}

	response_body.LoginCode = loginCode
	response_body.OpenId = request_body.GetOpenId()
	response_body.UserId = userId
	response_body.ZoneId = zoneId
	response_body.IsNewUser = accessSecret == ""
	response_body.VersionType = uint32(public_protocol_common.EnVersionType_EN_VERSION_DEFAULT)

	return nil
}

func (t *TaskActionLoginAuth) checkExistedUser(user *data.User) bool {
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

func (t *TaskActionLoginAuth) OnComplete() {
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
