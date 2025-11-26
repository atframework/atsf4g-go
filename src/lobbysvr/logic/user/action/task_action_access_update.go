// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	db "github.com/atframework/atsf4g-go/component-db"
)

type TaskActionAccessUpdate struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.CSAccessUpdateReq, *service_protocol.SCAccessUpdateRsp]
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
		t.GetLogger().Warn("invalid new access secret", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return nil
	}

	authTable, _ := db.DatabaseTableAccessLoadWithZoneIdUserId(t.GetAwaitableContext(), user.GetZoneId(), user.GetUserId())
	accessSecret := ""
	if authTable != nil {
		accessSecret = authTable.GetAccessSecret()
	}
	if accessSecret == "" || accessSecret != request_body.GetOldAccess() {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE))
		t.GetLogger().Warn("invalid old access secret", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return nil
	}

	accessSecret = request_body.GetNewAccess()
	loginCode := user.GetLoginLockInfo().GetLoginCode()

	table := private_protocol_pbdesc.DatabaseTableAccess{
		ZoneId:       user.GetZoneId(),
		UserId:       user.GetUserId(),
		AccessSecret: accessSecret,
		LoginCode:    loginCode,
	}
	err := db.DatabaseTableAccessUpdateZoneIdUserId(t.GetAwaitableContext(), &table)
	if err.IsError() {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM))
		t.GetLogger().Warn("save access secret failed", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "error", err)
		return nil
	}

	return nil
}
