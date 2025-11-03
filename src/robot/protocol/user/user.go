package atsf4g_go_robot_protocol_user

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	user "github.com/atframework/atsf4g-go/robot/data"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func LoginAuthRpc(user user.UserBase) error {
	if user.GetLoginCode() != "" {
		return fmt.Errorf("already login auth")
	}

	csMsg, csBody := makeLoginAuthMessage(user)
	return user.SendReq(csMsg, csBody, true)
}

func LoginRpc(user user.UserBase) error {
	if user.GetLoginCode() == "" {
		return fmt.Errorf("need login auth")
	}

	if user.GetLogined() {
		return fmt.Errorf("already login")
	}

	csMsg, csBody := makeLoginMessage(user)
	return user.SendReq(csMsg, csBody, true)
}

func PingRpc(user user.UserBase) error {
	if !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makePingMessage(user)
	return user.SendReq(csMsg, csBody, false)
}

func GetInfoRpc(user user.UserBase, args []string) error {
	if !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGetInfoMessage(user, args)
	return user.SendReq(csMsg, csBody, false)
}

func GMRpc(user user.UserBase, args []string) error {
	if !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGMMessage(user, args)
	return user.SendReq(csMsg, csBody, false)
}

func makeLoginAuthMessage(user user.UserBase) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSLoginAuthReq{
		OpenId: user.GetOpenId(),
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.GetAccessToken(),
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		SystemId:        public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS,
		PackageVersion:  "0.0.0.1",
		ResourceVersion: "0.0.0.1",
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.login_auth", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeLoginMessage(user user.UserBase) (*public_protocol_extension.CSMsg, proto.Message) {
	vmem, _ := mem.VirtualMemory()
	cpuInfo, _ := cpu.Info()

	csBody := &lobysvr_protocol_pbdesc.CSLoginReq{
		LoginCode: user.GetLoginCode(),
		OpenId:    user.GetOpenId(),
		UserId:    user.GetUserId(),
		ZoneId:    user.GetZoneId(),
		Account: &public_protocol_pbdesc.DAccountData{
			AccountType: uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INNER),
			Access:      user.GetAccessToken(),
			ChannelId:   uint32(public_protocol_pbdesc.EnPlatformChannelID_EN_PCI_NONE),
		},
		ClientInfo: &public_protocol_pbdesc.DClientDeviceInfo{
			SystemId:       public_protocol_pbdesc.EnSystemID_EN_OS_WINDOWS,
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
		Head: user.MakeMessageHead("proy.LobbyClientService.login", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)
	return &csMsg, csBody
}

func makePingMessage(user user.UserBase) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.ping", string(proto.MessageName(csBody))),
	}

	csMsg.BodyBin, _ = proto.Marshal(csBody)

	csMsg.BodyBin, _ = proto.Marshal(&lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	})

	return &csMsg, csBody
}

func makeUserGetInfoMessage(user user.UserBase, args []string) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGetInfoReq{}

	ref := csBody.ProtoReflect()
	fields := ref.Descriptor().Fields()

	needFields := make(map[string]struct{})
	for _, arg := range args {
		needFields[arg] = struct{}{}
	}

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		// 检查字段名前缀和类型
		if strings.HasPrefix(fieldName, "need_") && field.Kind() == protoreflect.BoolKind {
			if len(needFields) > 0 {
				_, ok := needFields[fieldName]
				if !ok {
					_, ok = needFields[strings.TrimPrefix(fieldName, "need_")]
					if !ok {
						continue
					}
				}
			}

			ref.Set(field, protoreflect.ValueOfBool(true))
		}
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.user_get_info", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func makeUserGMMessage(user user.UserBase, args []string) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: args,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.user_send_gm_command", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func ProcessLoginAuthResponse(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
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
		user.SetLoginCode(body.GetLoginCode())
	}
	if body.GetUserId() != 0 {
		user.SetUserId(body.GetUserId())
	}
	if body.GetZoneId() != 0 {
		user.SetZoneId(body.GetZoneId())
	}
}

func ProcessLoginResponse(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
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
		user.SetZoneId(body.GetZoneId())
	}
	user.SetLogined(true)
	if body.GetHeartbeatInterval() > 0 {
		user.SetHeartbeatInterval(time.Duration(body.GetHeartbeatInterval()) * time.Second)
	}
}

func ProcessPongResponse(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.SetLastPingTime(time.Now())
}

func ProcessGetInfoResponse(user user.UserBase, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	_, ok := rawBody.(*lobysvr_protocol_pbdesc.SCUserGetInfoRsp)
	if !ok {
		log.Println("Can not convert to SCUserGetInfoRsp")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.SetHasGetInfo(true)
}
