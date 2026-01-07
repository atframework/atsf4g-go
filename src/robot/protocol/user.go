package atsf4g_go_robot_protocol

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	user_data "github.com/atframework/atsf4g-go/robot/data"

	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func LoginAuthRpc(action *user_data.TaskActionUser) (int32, *lobysvr_protocol_pbdesc.SCLoginAuthRsp, error) {
	user := action.User
	if user.GetLoginCode() != "" {
		return 0, nil, fmt.Errorf("already login auth")
	}

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
	return user_data.SendLoginAuth(action, csBody, false)
}

func LoginRpc(action *user_data.TaskActionUser) (int32, *lobysvr_protocol_pbdesc.SCLoginRsp, error) {
	user := action.User
	if user.GetLoginCode() == "" {
		return 0, nil, fmt.Errorf("need login auth")
	}

	if user.GetLogined() {
		return 0, nil, fmt.Errorf("already login")
	}

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
				}
				return "unknown"
			}(),
			Memory: uint32(vmem.Total / (1024 * 1024)),
		},
	}

	return user_data.SendLogin(action, csBody, false)
}

func PingRpc(action *user_data.TaskActionUser) error {
	csBody := &lobysvr_protocol_pbdesc.CSPingReq{
		Timepoint: time.Now().UnixNano(),
	}

	errCode, _, err := user_data.SendPing(action, csBody, true)
	if err != nil {
		return err
	}
	if errCode < 0 {
		return fmt.Errorf("ping failed, errCode: %d", errCode)
	}
	action.User.SetLastPingTime(time.Now())
	return nil
}

func GetInfoRpc(action *user_data.TaskActionUser, args []string) (int32, *lobysvr_protocol_pbdesc.SCUserGetInfoRsp, error) {
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

	return user_data.SendUserGetInfo(action, csBody, true)
}

func GMRpc(action *user_data.TaskActionUser, args []string) (int32, *lobysvr_protocol_pbdesc.SCUserGMCommandRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: args,
	}
	return user_data.SendUserSendGmCommand(action, csBody, true)
}

func UsingItemRpc(action *user_data.TaskActionUser, item *public_protocol_common.DItemBasic, param *public_protocol_common.DItemUseParam) (int32, *lobysvr_protocol_pbdesc.SCUserUseItemRsp, error) {
	csBody := &lobysvr_protocol_pbdesc.CSUserUseItemReq{
		Item:     item,
		UseParam: param,
	}
	return user_data.SendUserUseItem(action, csBody, true)
}
