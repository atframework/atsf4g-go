// client.go
package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/proto"
)

var processResponseHandles = buildProcessResponseHandles()

func buildProcessResponseHandles() map[string]func(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	handles := make(map[string]func(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message))
	handles["proy.LobbyClientService.login_auth"] = processLoginAuthResponse
	handles["proy.LobbyClientService.login"] = processLoginResponse
	handles["proy.LobbyClientService.ping"] = processPongResponse
	handles["proy.LobbyClientService.user_get_info"] = processGetInfoResponse
	return handles
}

func LoginAuthRpc(user *User) error {
	if user.LoginCode != "" {
		return fmt.Errorf("already login auth")
	}

	csMsg, csBody := makeLoginAuthMessage(user)
	return sendReq(user, csMsg, csBody, true)
}

func LoginRpc(user *User) error {
	if user.LoginCode == "" {
		return fmt.Errorf("need login auth")
	}

	if user.Logined {
		return fmt.Errorf("already login")
	}

	csMsg, csBody := makeLoginMessage(user)
	return sendReq(user, csMsg, csBody, true)
}

func PingRpc(user *User) error {
	if !user.Logined {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makePingMessage(user)
	return sendReq(user, csMsg, csBody, false)
}

func GetInfoRpc(user *User) error {
	if !user.Logined {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGetInfoMessage(user)
	return sendReq(user, csMsg, csBody, false)
}

func GMRpc(user *User, args []string) error {
	if !user.Logined {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGMMessage(user, args)
	return sendReq(user, csMsg, csBody, false)
}

func makeLoginAuthMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSLoginAuthReq{
		OpenId: user.OpenId,
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.AccessToken,
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		SystemId:        public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS,
		PackageVersion:  "0.0.0.1",
		ResourceVersion: "0.0.0.1",
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.login_auth", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeLoginMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	vmem, _ := mem.VirtualMemory()
	cpuInfo, _ := cpu.Info()

	csBody := &lobysvr_protocol_pbdesc.CSLoginReq{
		LoginCode: user.LoginCode,
		OpenId:    user.OpenId,
		UserId:    user.UserId,
		ZoneId:    user.ZoneId,
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.AccessToken,
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		ClientInfo: &public_protocol_pbdesc.DClientDeviceInfo{
			SystemId:       uint32(public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS),
			ClientVersion:  "0.0.0.1",
			SystemSoftware: runtime.GOOS,
			SystemHardware: runtime.GOARCH,
			CpuInfo: func() string {
				if len(cpuInfo) > 0 {
					return fmt.Sprintf("%s - %gMHz", strings.TrimSpace(cpuInfo[0].ModelName), cpuInfo[0].Mhz)
				} else {
					return "unknown"
				}
			}(),
			Memory: uint32(vmem.Total / (1024 * 1024)),
		},
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.login", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)
	return &csMsg, csBody
}

func makePingMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.ping", string(proto.MessageName(csBody))),
	}

	csMsg.BodyBin, _ = proto.Marshal(csBody)

	csMsg.BodyBin, _ = proto.Marshal(&lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	})

	return &csMsg, csBody
}

func makeUserGetInfoMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGetInfoReq{
		NeedUserInfo:      true,
		NeedUserOptions:   true,
		NeedUserInventory: true,
		NeedUserBuilding:  true,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_get_info", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeUserGMMessage(user *User, args []string) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: args,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_send_gm_command", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func processLoginAuthResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginAuthRsp)
	if !ok {
		log.Println("Can not convert to SCLoginAuthRsp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	if body.GetLoginCode() != "" {
		user.LoginCode = body.GetLoginCode()
	}
	if body.GetUserId() != 0 {
		user.UserId = body.GetUserId()
	}
	if body.GetZoneId() != 0 {
		user.ZoneId = body.GetZoneId()
	}
}

func processLoginResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginRsp)
	if !ok {
		log.Println("Can not convert to SCLoginResp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	if body.GetZoneId() != 0 {
		user.ZoneId = body.GetZoneId()
	}
	user.Logined = true
	if body.GetHeartbeatInterval() > 0 {
		user.HeartbeatInterval = time.Duration(body.GetHeartbeatInterval()) * time.Second
	}
}

func processPongResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.LastPingTime = time.Now()
}

func processGetInfoResponse(user *User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	_, ok := rawBody.(*lobysvr_protocol_pbdesc.SCUserGetInfoRsp)
	if !ok {
		log.Println("Can not convert to SCUserGetInfoRsp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.HasGetInfo = true
}
