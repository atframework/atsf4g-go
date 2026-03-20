// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	"github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"

	user_auth "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user/auth"
)

type TaskActionAccessUpdate struct {
	user_controller.TaskActionCSBase[*service_protocol.CSAccessUpdateReq, *service_protocol.SCAccessUpdateRsp]
}

func (t *TaskActionAccessUpdate) Name() string {
	return "TaskActionAccessUpdate"
}

func (t *TaskActionAccessUpdate) Run(_startData *component_dispatcher.DispatcherStartData) error {
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	request_body := t.GetRequestBody()

	if request_body.GetNewAccess() == "" {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		t.GetRpcContext().LogWarn("invalid new access secret")
		return nil
	}

	authTable, _ := db.DatabaseTableAccessLoadWithZoneIdUserId(t.GetAwaitableContext(), user.GetZoneId(), user.GetUserId())
	accessSecret := ""
	if authTable != nil {
		accessSecret = authTable.GetAccessSecret()
	} else {
		authTable = &pbdesc.DatabaseTableAccess{
			ZoneId:       user.GetZoneId(),
			UserId:       user.GetUserId(),
			AccessSecret: accessSecret,
			LoginCode:    user.GetLoginLockInfo().GetLoginCode(),
		}
	}

	lobbySvrCfg := config.GetServerConfig[*private_protocol_config.Readonly_LobbyServerCfg](config.GetConfigManager().GetCurrentConfigGroup())
	if lobbySvrCfg == nil {
		t.LogError("Lobby server config is nil")
	}

	// TODO: 临时的密码模式要走密码验证算法
	if lobbySvrCfg.GetAuth().GetAllowNoAuth() {
		// SHA256(password) should be 64 characters, so this is a simple check to prevent plaintext password bypassing auth
		if len(request_body.GetNewAccess()) > 64 {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
			t.GetRpcContext().LogWarn("invalid new access secret")
			return nil
		}
		checkPassed, err := user_auth.VerifyPassword(request_body.GetOldAccess(), accessSecret)
		if err != nil || !checkPassed {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
			t.GetRpcContext().LogWarn("verify password failed", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "error", err)
			return nil
		}

		newHash, err := user_auth.GeneratePasswordHash(t.GetRpcContext(), request_body.GetNewAccess())
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			t.GetRpcContext().LogWarn("generate password hash failed", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "error", err)
			return nil
		}
		authTable.AccessSecret = newHash
	} else {
		if accessSecret == "" || accessSecret != request_body.GetOldAccess() {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE))
			t.GetRpcContext().LogWarn("invalid old access secret")
			return nil
		}
		authTable.AccessSecret = request_body.GetNewAccess()
	}

	accessSecret = request_body.GetNewAccess()
	authTable.LoginCode = user.GetLoginLockInfo().GetLoginCode()

	err := db.DatabaseTableAccessUpdateZoneIdUserId(t.GetAwaitableContext(), authTable)
	if err.IsError() {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		t.GetRpcContext().LogWarn("save access secret failed", "error", err)
		return nil
	}

	return nil
}
