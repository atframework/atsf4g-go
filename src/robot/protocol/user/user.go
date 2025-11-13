package atsf4g_go_robot_protocol_user

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	user_impl "github.com/atframework/atsf4g-go/robot/data/impl"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func LoginAuthRpc(user user_data.User) error {
	if user.GetLoginCode() != "" {
		return fmt.Errorf("already login auth")
	}

	return user.SendReq(makeLoginAuthMessage(user))
}

func LoginRpc(user user_data.User) error {
	if user.GetLoginCode() == "" {
		return fmt.Errorf("need login auth")
	}

	if user.GetLogined() {
		return fmt.Errorf("already login")
	}

	return user.SendReq(makeLoginMessage(user))
}

func PingRpc(user user_data.User) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	return user.SendReq(makePingMessage(user))
}

func GetInfoRpc(user user_data.User, args []string) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	return user.SendReq(makeUserGetInfoMessage(user, args))
}

func GMRpc(user user_data.User, args []string) error {
	if lu.IsNil(user) || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	return user.SendReq(makeUserGMMessage(user, args))
}

func makeLoginAuthMessage(user user_data.User) (*public_protocol_extension.CSMsg, proto.Message) {
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

func makeLoginMessage(user user_data.User) (*public_protocol_extension.CSMsg, proto.Message) {
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

func makePingMessage(user user_data.User) (*public_protocol_extension.CSMsg, proto.Message) {
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

func makeUserGetInfoMessage(user user_data.User, args []string) (*public_protocol_extension.CSMsg, proto.Message) {
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

func makeUserGMMessage(user user_data.User, args []string) (*public_protocol_extension.CSMsg, proto.Message) {
	csBody := &lobysvr_protocol_pbdesc.CSUserGMCommandReq{
		Args: args,
	}

	csMsg := public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("proy.LobbyClientService.user_send_gm_command", string(proto.MessageName(csBody))),
	}
	csMsg.BodyBin, _ = proto.Marshal(csBody)

	return &csMsg, csBody
}

func ProcessLoginAuthResponse(user user_data.User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginAuthRsp)
	if !ok {
		utils.StdoutLog("Can not convert to SCLoginAuthRsp\n")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	if user == nil {
		userMapLock.Lock()
		defer userMapLock.Unlock()

		userInst, ok := mutableUserMapContainer()[body.GetOpenId()]
		if !ok || userInst == nil {
			user = userInst
			user = mutableUserMapContainer()[strconv.FormatUint(body.GetUserId(), 10)]
		}
	}

	if user == nil {
		utils.StdoutLog("user is nil in ProcessLoginAuthResponse\n")
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

	err := LoginRpc(user)
	if err != nil {
		utils.StdoutLog(fmt.Sprintf("user login failed, error: %v, open_id: %s, user_id: %d, zone_id: %d", err, user.GetOpenId(), user.GetUserId(), user.GetZoneId()))
		return
	}
}

func ProcessLoginResponse(user user_data.User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	body, ok := rawBody.(*lobysvr_protocol_pbdesc.SCLoginRsp)
	if !ok {
		utils.StdoutLog("Can not convert to SCLoginResp\n")
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

	user_data.SetCurrentUser(user)

	// 创建Ping流程
	user.CheckPingTask()
}

func ProcessPongResponse(user user_data.User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.SetLastPingTime(time.Now())
}

func ProcessGetInfoResponse(user user_data.User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
	_, ok := rawBody.(*lobysvr_protocol_pbdesc.SCUserGetInfoRsp)
	if !ok {
		utils.StdoutLog("Can not convert to SCUserGetInfoRsp\n")
		return
	}

	head := msg.Head
	if head.ErrorCode < 0 {
		return
	}

	user.SetHasGetInfo(true)
}

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommand([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议", nil)
	utils.RegisterCommand([]string{"user", "logout"}, LogoutCmd, "", "登出协议", nil)
	utils.RegisterCommand([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息", nil)
	utils.RegisterCommand([]string{"gm"}, GMCmd, "<args...>", "GM", nil)

	user_data.RegisterResponseHandle("proy.LobbyClientService.login_auth", ProcessLoginAuthResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.login", ProcessLoginResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.ping", ProcessPongResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.user_get_info", ProcessGetInfoResponse)

	utils.RegisterCommand([]string{"user", "show_all_login_user"}, func([]string) string {
		userMapLock.Lock()
		defer userMapLock.Unlock()
		for _, v := range userMapContainer {
			fmt.Printf("%d\n", v.GetUserId())
		}
		return ""
	}, "", "显示所有登录User", nil)
	utils.RegisterCommand([]string{"user", "switch"}, func(cmd []string) string {
		if len(cmd) < 1 {
			return "Need User Id"
		}

		userId, err := strconv.ParseInt(cmd[0], 10, 64)
		if err != nil {
			return err.Error()
		}

		userMapLock.Lock()
		v, ok := userMapContainer[strconv.FormatUint(uint64(userId), 10)]
		userMapLock.Unlock()
		if !ok {
			return "not found user"
		}

		user_data.SetCurrentUser(v)
		return ""
	}, "<userId>", "切换登录User", AutoCompleteUseIdWithoutCurrent)
}

var (
	userMapContainer map[string]*user_impl.User
	userMapLock      sync.Mutex
)

func mutableUserMapContainer() map[string]*user_impl.User {
	if userMapContainer == nil {
		userMapContainer = make(map[string]*user_impl.User)
	}
	return userMapContainer
}

func AutoCompleteUseId(string) []string {
	userMapLock.Lock()
	defer userMapLock.Unlock()
	var res []string
	for _, k := range userMapContainer {
		res = append(res, strconv.FormatUint(k.GetUserId(), 10))
	}
	return res
}

func AutoCompleteUseIdWithoutCurrent(string) []string {
	userMapLock.Lock()
	defer userMapLock.Unlock()
	var res []string
	for _, k := range userMapContainer {
		if k.GetUserId() == user_data.GetCurrentUser().GetUserId() {
			continue
		}
		res = append(res, strconv.FormatUint(k.GetUserId(), 10))
	}
	return res
}

func CreateUser(openId string, socketUrl string) *user_impl.User {
	userMapLock.Lock()

	if existingUser, ok := mutableUserMapContainer()[openId]; ok && existingUser != nil {
		userMapLock.Unlock()
		return existingUser
	}
	userMapLock.Unlock()

	bufferWriter, _ := libatapp.NewlogBufferedRotatingWriter(
		"../log", openId, 20*1024*1024, 3, time.Second*3, false, false)
	runtime.SetFinalizer(bufferWriter, func(writer *libatapp.LogBufferedRotatingWriter) {
		writer.Close()
	})

	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		utils.StdoutLog(fmt.Sprintf("Error connecting to Websocket Server: %v", err))
		return nil
	}

	var ret *user_impl.User = nil
	userMapLock.Lock()
	defer userMapLock.Unlock()
	ret = user_impl.CreateUser(openId, conn, bufferWriter, PingRpc)

	mutableUserMapContainer()[openId] = ret

	ret.AddOnClosedHandler(func(user user_data.User) {
		userMapLock.Lock()
		defer userMapLock.Unlock()

		u, ok := mutableUserMapContainer()[openId]
		if !ok || u != user {
			return
		}
		delete(mutableUserMapContainer(), openId)
		utils.StdoutLog(fmt.Sprintf("Remove User: %s", openId))

		if user_data.GetCurrentUser() == user {
			user_data.SetCurrentUser(nil)
		}
	})
	go ret.ActionHandler()
	go ret.ReceiveHandler()

	utils.StdoutLog(fmt.Sprintf("Create User: %s\n", openId))

	return ret
}

func LogoutCmd(cmd []string) string {
	if user_data.GetCurrentUser() != nil {
		fmt.Printf("user %s logout\n", user_data.GetCurrentUser().GetOpenId())
		user_data.GetCurrentUser().Logout()
		user_data.SetCurrentUser(nil)
	}
	return ""
}

func LoginCmd(cmd []string) string {
	if len(cmd) < 1 {
		return "Need OpenId"
	}
	// 创建角色
	u := CreateUser(cmd[0], config.SocketUrl)
	if u == nil {
		return "Failed to create user"
	}

	// 发送登录请求
	err := LoginAuthRpc(u)
	if err != nil {
		return err.Error()
	}

	return ""
}

func GetInfoCmd(cmd []string) string {
	// 发送登录请求
	err := GetInfoRpc(user_data.GetCurrentUser(), cmd)
	if err != nil {
		return err.Error()
	}
	return ""
}

func GMCmd(cmd []string) string {
	// 发送登录请求
	err := GMRpc(user_data.GetCurrentUser(), cmd)
	if err != nil {
		return err.Error()
	}
	return ""
}
