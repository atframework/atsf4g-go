// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"

	logic_open_platform "github.com/atframework/atsf4g-go/service-lobbysvr/logic/open_platform"
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

	needPasswordUpdate := request_body.GetNewAccess() != ""
	needOpenPlatformAccessUpdate := request_body.GetAccount() != nil &&
		request_body.GetAccount().GetAccess() != "" &&
		request_body.GetAccount().GetAccountType() == user.GetAccountInfo().GetAccountType() &&
		request_body.GetAccount().GetAccountType() != uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INTERNAL)

	if !needPasswordUpdate && !needOpenPlatformAccessUpdate {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		t.GetRpcContext().LogWarn("no access update data provided",
			"zone_id", user.GetZoneId(),
			"user_id", user.GetUserId(),
			"open_id", user.GetOpenId())
		return nil
	}

	authTable, _ := db.DatabaseTableAccessLoadWithOpenId(t.GetAwaitableContext(), user.GetOpenId())
	accessSecret := ""
	if authTable != nil {
		accessSecret = authTable.GetAccessSecret()
	} else {
		authTable = &private_protocol_pbdesc.DatabaseTableAccess{
			OpenId:       user.GetOpenId(),
			ZoneId:       user.GetZoneId(),
			UserId:       user.GetUserId(),
			AccessSecret: accessSecret,
			LoginCode:    user.GetLoginLockInfo().GetLoginCode(),
		}
	}

	// 更新密码认证方式的密码
	if needPasswordUpdate {
		if !t.updateAccessDataPassword(authTable) {
			return nil
		}
	}

	if needOpenPlatformAccessUpdate {
		if !t.updateAccessDataOpenPlatform(authTable) {
			return nil
		}
	}

	authTable.LoginCode = user.GetLoginLockInfo().GetLoginCode()
	// 任意鉴权方式都可以刷新AccessData表
	if request_body.GetAuthData() != nil {
		authTable.MutableLastAccessData().AccessData = request_body.GetAuthData()
	}

	err := db.DatabaseTableAccessUpdateOpenId(t.GetAwaitableContext(), authTable)
	if err.IsError() {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		t.GetRpcContext().LogWarn("save access secret failed", "error", err)
		return nil
	}

	return nil
}

func (t *TaskActionAccessUpdate) updateAccessDataPassword(authTable *private_protocol_pbdesc.DatabaseTableAccess) bool {
	request_body := t.GetRequestBody()

	accessSecret := ""
	if authTable != nil {
		accessSecret = authTable.GetAccessSecret()
	} else {
		authTable = &private_protocol_pbdesc.DatabaseTableAccess{
			OpenId:       t.GetUser().GetOpenId(),
			ZoneId:       t.GetUser().GetZoneId(),
			UserId:       t.GetUser().GetUserId(),
			AccessSecret: accessSecret,
			LoginCode:    t.GetUser().GetLoginLockInfo().GetLoginCode(),
		}
	}

	lobbySvrCfg := config.GetServerConfig[*private_protocol_config.Readonly_LobbyServerCfg](config.GetConfigManager().GetCurrentConfigGroup())
	if lobbySvrCfg == nil {
		t.LogError("Lobby server config is nil")
	}

	if lobbySvrCfg.GetAuth().GetAllowNoAuth() {
		// SHA256(password) should be 64 characters, so this is a simple check to prevent plaintext password bypassing auth
		if len(request_body.GetNewAccess()) > 64 {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
			t.GetRpcContext().LogWarn("invalid new access secret")
			return false
		}
		checkPassed, err := user_auth.VerifyPassword(request_body.GetOldAccess(), accessSecret)
		if err != nil || !checkPassed {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
			t.GetRpcContext().LogWarn("verify password failed", "zone_id", t.GetUser().GetZoneId(), "user_id", t.GetUser().GetUserId(), "error", err)
			return false
		}

		newHash, err := user_auth.GeneratePasswordHash(t.GetRpcContext(), request_body.GetNewAccess())
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			t.GetRpcContext().LogWarn("generate password hash failed", "zone_id", t.GetUser().GetZoneId(), "user_id", t.GetUser().GetUserId(), "error", err)
			return false
		}
		authTable.AccessSecret = newHash
	} else {
		if accessSecret == "" || accessSecret != request_body.GetOldAccess() {
			t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE))
			t.GetRpcContext().LogWarn("invalid old access secret", "zone_id", t.GetUser().GetZoneId(), "user_id", t.GetUser().GetUserId())
			return false
		}
		newHash, err := user_auth.GeneratePasswordHash(t.GetRpcContext(), request_body.GetNewAccess())
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			t.GetRpcContext().LogWarn("generate password hash failed", "zone_id", t.GetUser().GetZoneId(), "user_id", t.GetUser().GetUserId(), "error", err)
			return false
		}
		authTable.AccessSecret = newHash
	}

	return true
}

func (t *TaskActionAccessUpdate) updateAccessDataOpenPlatform(authTable *private_protocol_pbdesc.DatabaseTableAccess) bool {
	request_body := t.GetRequestBody()
	if request_body.GetAccount() != nil {
		user, _ := t.GetUser().(*data.User)
		mgr := data.UserGetModuleManager[logic_open_platform.UserOpenPlatformManager](user)
		if mgr != nil {
			mgr.UpdateAccessToken(t.GetRpcContext(),
				request_body.GetAccount().GetAccess(), request_body.GetAuthData())
		}

		authTable.MutableLastAccessData().MutableAccount().Access = request_body.GetAccount().GetAccess()
	}

	return true
}
