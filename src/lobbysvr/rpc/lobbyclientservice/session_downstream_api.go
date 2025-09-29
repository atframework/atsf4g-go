// Copyright 2025 atframework
// @brief Created by mako-generator.py for proy.LobbyClientService, please don't edit it

package lobbysvr_rpc_lobbyclientservice


import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"

	ppe "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"

	sp "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
)

func sendMessage(responseCode int32, session *uc.Session,
	rd cd.DispatcherImpl, now time.Time,
	rpcType interface{}, body proto.Message,
) error {
	msg, err := cd.CreateCSMessage(responseCode, now, 0,
		rd, session,
		rpcType, body)
	if err != nil {
		return err
	}

	return session.SendMessage(msg)
}


func SendUserDirtyChgSync(session *uc.Session, body *sp.SCUserDirtyChgSync, responseCode int32) error {
	if session == nil || body == nil {
		return fmt.Errorf("session or message body is nil")
	}

	rd := session.GetNetworkHandle().GetDispatcher()
	if rd == nil {
		return fmt.Errorf("session dispatcher is nil")
	}

	now := rd.GetNow()

	return sendMessage(responseCode, session, rd, now, &ppe.RpcStreamMeta{
		Version:         "0.1.0",  // TODO: make it configurable
		RpcName:         "proy.LobbyClientService.user_dirty_chg_sync",
		TypeUrl:         "proy.SCUserDirtyChgSync",
		Caller:          rd.GetApp().GetTypeName(),
		CallerTimestamp: timestamppb.New(now),
	}, body)
}
