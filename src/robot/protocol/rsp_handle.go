package atsf4g_go_robot_protocol

import (
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	user "github.com/atframework/atsf4g-go/robot/data"
	"google.golang.org/protobuf/proto"

	protocol_user "github.com/atframework/atsf4g-go/robot/protocol/user"
)

var ProcessResponseHandles = buildProcessResponseHandles()

func buildProcessResponseHandles() map[string]func(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	handles := make(map[string]func(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message))
	handles["proy.LobbyClientService.login_auth"] = protocol_user.ProcessLoginAuthResponse
	handles["proy.LobbyClientService.login"] = protocol_user.ProcessLoginResponse
	handles["proy.LobbyClientService.ping"] = protocol_user.ProcessPongResponse
	handles["proy.LobbyClientService.user_get_info"] = protocol_user.ProcessGetInfoResponse
	return handles
}
