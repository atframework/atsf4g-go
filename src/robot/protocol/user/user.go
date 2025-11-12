package atsf4g_go_robot_protocol_user

import (
	"fmt"
	"log"
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

	csMsg, csBody := makeLoginAuthMessage(user)
	return user.SendReq(csMsg, csBody)
}

func LoginRpc(user user_data.User) error {
	if user.GetLoginCode() == "" {
		return fmt.Errorf("need login auth")
	}

	if user.GetLogined() {
		return fmt.Errorf("already login")
	}

	csMsg, csBody := makeLoginMessage(user)
	return user.SendReq(csMsg, csBody)
}

func PingRpc(user user_data.User) error {
	if user == nil || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makePingMessage(user)
	return user.SendReq(csMsg, csBody)
}

func GetInfoRpc(user user_data.User, args []string) error {
	if user == nil || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGetInfoMessage(user, args)
	return user.SendReq(csMsg, csBody)
}

func GMRpc(user user_data.User, args []string) error {
	if user == nil || !user.IsLogin() {
		return fmt.Errorf("need login")
	}

	csMsg, csBody := makeUserGMMessage(user, args)
	return user.SendReq(csMsg, csBody)
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
		log.Println("Can not convert to SCLoginAuthRsp")
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
		log.Println("user is nil in ProcessLoginAuthResponse")
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
		log.Println("user login failed", "error", err, "open_id", user.GetOpenId(),
			"user_id", user.GetUserId(), "zone_id", user.GetZoneId())
		return
	}
}

func ProcessLoginResponse(user user_data.User, rpcName string, msg *public_protocol_extension.CSMsg, rawBody proto.Message) {
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
	user_data.AddLoginUser(user)

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
		log.Println("Can not convert to SCUserGetInfoRsp")
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
	utils.RegisterCommand([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议")
	utils.RegisterCommand([]string{"user", "logout"}, LogoutCmd, "", "登出协议")
	utils.RegisterCommand([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息")
	utils.RegisterCommand([]string{"gm"}, GMCmd, "<args...>", "GM")

	user_data.RegisterResponseHandle("proy.LobbyClientService.login_auth", ProcessLoginAuthResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.login", ProcessLoginResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.ping", ProcessPongResponse)
	user_data.RegisterResponseHandle("proy.LobbyClientService.user_get_info", ProcessGetInfoResponse)
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
		log.Fatal("Error connecting to Websocket Server:", err)
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
		log.Println("Remove User:", openId)

		if user_data.GetCurrentUser() == user {
			user_data.SetCurrentUser(nil)
		}
	})
	go ret.ActionHandler()
	go ret.ReceiveHandler()

	log.Println("Create User:", openId)

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
