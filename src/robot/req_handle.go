// client.go
package main

import (
	"fmt"
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

func makeMessageHead(user *User, rpcName string, typeName string) *public_protocol_extension.CSMsgHead {
	user.connectionSequence++
	return &public_protocol_extension.CSMsgHead{
		Timestamp:      time.Now().Unix(),
		ClientSequence: user.connectionSequence,
		RpcType: &public_protocol_extension.CSMsgHead_RpcRequest{
			RpcRequest: &public_protocol_extension.RpcRequestMeta{
				RpcName: rpcName,
				TypeUrl: typeName,
			},
		},
	}
}

func processMakeRequest(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	user.dispatcherLock.Lock()
	defer user.dispatcherLock.Unlock()

	var csMsg *public_protocol_extension.CSMsg
	var csBody proto.Message
	if user.LoginCode == "" {
		csMsg, csBody = makeLoginAuthMessage(user)
	} else if !user.Logined {
		csMsg, csBody = makeLoginMessage(user)
	} else if user.LastPingTime.Add(user.HeartbeatInterval).Before(time.Now()) {
		csMsg, csBody = makePingMessage(user)
	} else if !user.HasGetInfo {
		// Here could add more messages after login
		csMsg, csBody = makeUserGetInfoMessage(user)
	} else {
		csMsg, csBody = makeUserGMMessage(user)
	}

	return csMsg, csBody
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
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_get_info", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeUserGMMessage(user *User) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: []string{"add-item", "1001", "1"},
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: makeMessageHead(user, "proy.LobbyClientService.user_send_gm_command", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}
