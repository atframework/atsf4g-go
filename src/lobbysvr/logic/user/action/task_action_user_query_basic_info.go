// Copyright 2026 atframework

package lobbysvr_logic_user_action

import (
	"fmt"

	db "github.com/atframework/atsf4g-go/component/db"
	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

const maxUserBasicInfoQueryCount = 30

type TaskActionUserQueryBasicInfo struct {
	user_controller.TaskActionCSBase[*service_protocol.CSQueryUsersBasicInfoReq, *service_protocol.SCQueryUsersBasicInfoRsp]
}

func (t *TaskActionUserQueryBasicInfo) Name() string {
	return "TaskActionUserQueryBasicInfo"
}

func (t *TaskActionUserQueryBasicInfo) Run(_startData *component_dispatcher.DispatcherStartData) error {
	requestBody := t.GetRequestBody()
	responseBody := t.MutableResponseBody()

	if len(requestBody.GetUserZoneIdMap()) == 0 {
		return nil
	}
	if len(requestBody.GetUserZoneIdMap()) > maxUserBasicInfoQueryCount {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM))
		return fmt.Errorf("query users basic info exceed limit, max=%d, got=%d",
			maxUserBasicInfoQueryCount, len(requestBody.GetUserZoneIdMap()))
	}

	loadKeys := make([]db.DatabaseTableUserTableKey, 0, len(requestBody.GetUserZoneIdMap()))
	for userID, zoneID := range requestBody.GetUserZoneIdMap() {
		loadKeys = append(loadKeys, db.DatabaseTableUserTableKey{ZoneId: zoneID, UserId: userID})
	}

	loadResult, batchRet := db.DatabaseTableUserBatchLoadWithZoneIdUserIdPartlyGetBasicInfo(t.GetAwaitableContext(), loadKeys)
	if batchRet.IsError() {
		t.SetResponseCode(batchRet.GetResponseCode())
		return batchRet.GetStandardError()
	}

	rspUserInfo := make(map[uint64]*public_protocol_pbdesc.DUserBasicInfo)
	for index, loaded := range loadResult {
		if loaded.Result.IsError() {
			if loaded.Result.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
				continue
			}

			t.SetResponseCode(loaded.Result.GetResponseCode())
			return loaded.Result.GetStandardError()
		}

		if loaded.Table == nil {
			continue
		}

		profile := loaded.Table.GetAccountData().GetProfile()
		rspUserInfo[loadKeys[index].UserId] = &public_protocol_pbdesc.DUserBasicInfo{
			UserId:      loadKeys[index].UserId,
			NickName:    profile.GetNickName(),
			ProfileCard: profile.GetProfileCard(),
			Avatar:      profile.GetAvatar(),

			PlatformNickName: profile.GetPlatformNickName(),
			PlatformAvatar:   profile.GetPlatformAvatar(),
		}
	}

	responseBody.UserInfo = rspUserInfo
	return nil
}
