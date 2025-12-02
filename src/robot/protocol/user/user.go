package atsf4g_go_robot_protocol_user

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	base "github.com/atframework/atsf4g-go/robot/base"
	config "github.com/atframework/atsf4g-go/robot/config"
	user_data "github.com/atframework/atsf4g-go/robot/data"
	user_impl "github.com/atframework/atsf4g-go/robot/data/impl"

	utils "github.com/atframework/atsf4g-go/robot/utils"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"github.com/gorilla/websocket"
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

func PingTask(user user_data.User) error {
	user.RunTaskDefaultTimeout(func(action *user_data.TaskActionUser) {
		PingRpc(action)
	})
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

// ========================= 注册指令 =========================
func init() {
	utils.RegisterCommand([]string{"user", "login"}, LoginCmd, "<openid>", "登录协议", nil)
	utils.RegisterCommand([]string{"user", "logout"}, LogoutCmd, "", "登出协议", nil)
	utils.RegisterCommand([]string{"user", "getInfo"}, GetInfoCmd, "", "拉取用户信息", nil)
	utils.RegisterCommand([]string{"user", "benchmark"}, BenchmarkCmd, "", "压测协议", nil)
	utils.RegisterCommand([]string{"gm"}, GMCmd, "<args...>", "GM", nil)
	utils.RegisterCommand([]string{"user", "show_all_login_user"}, func(action base.TaskActionImpl, cmd []string) string {
		userMapLock.Lock()
		defer userMapLock.Unlock()
		for _, v := range userMapContainer {
			fmt.Printf("%d\n", v.GetUserId())
		}
		return ""
	}, "", "显示所有登录User", nil)
	utils.RegisterCommand([]string{"user", "switch"}, func(action base.TaskActionImpl, cmd []string) string {
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

	bufferWriter, _ := libatapp.NewLogBufferedRotatingWriter(nil,
		fmt.Sprintf("../log/%s.%%N.log", openId), "", 20*1024*1024, 3, time.Second*3)
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
	ret = user_impl.CreateUser(openId, conn, bufferWriter, PingTask)

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
	go ret.ReceiveHandler()

	utils.StdoutLog(fmt.Sprintf("Create User: %s\n", openId))

	return ret
}

func LogoutCmd(action base.TaskActionImpl, cmd []string) string {
	if user_data.GetCurrentUser() != nil {
		fmt.Printf("user %s logout\n", user_data.GetCurrentUser().GetOpenId())
		user_data.GetCurrentUser().Logout()
		user_data.SetCurrentUser(nil)
	}
	return ""
}

func LoginCmd(action base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 1 {
		return "Need OpenId"
	}

	// 创建角色
	u := CreateUser(cmd[0], config.SocketUrl)
	if u == nil {
		return "Failed to create user"
	}

	var err error
	action.AwaitTask(u.RunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		errCode, rsp, rpcErr := LoginAuthRpc(task)
		if rpcErr != nil {
			err = rpcErr
			return
		}
		if errCode < 0 {
			err = fmt.Errorf("login auth failed, errCode: %d", errCode)
			return
		}

		user := task.User

		if rsp.GetLoginCode() != "" {
			user.SetLoginCode(rsp.GetLoginCode())
		}
		if rsp.GetUserId() != 0 {
			user.SetUserId(rsp.GetUserId())
		}
		if rsp.GetZoneId() != 0 {
			user.SetZoneId(rsp.GetZoneId())
		}

		var loginRsp *lobysvr_protocol_pbdesc.SCLoginRsp
		errCode, loginRsp, rpcErr = LoginRpc(task)
		if rpcErr != nil {
			utils.StdoutLog(fmt.Sprintf("user login failed, error: %v, open_id: %s, user_id: %d, zone_id: %d", err, user.GetOpenId(), user.GetUserId(), user.GetZoneId()))
			err = rpcErr
			return
		}
		if errCode < 0 {
			err = fmt.Errorf("login req failed, errCode: %d", errCode)
			return
		}

		if loginRsp.GetZoneId() != 0 {
			user.SetZoneId(loginRsp.GetZoneId())
		}
		user.SetLogined(true)

		if loginRsp.GetHeartbeatInterval() > 0 {
			user.SetHeartbeatInterval(time.Duration(loginRsp.GetHeartbeatInterval()) * time.Second)
		}

		user_data.SetCurrentUser(user)

		// 创建Ping流程
		user.CheckPingTask()
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}

func GetInfoCmd(action base.TaskActionImpl, cmd []string) string {
	// 发送登录请求
	var err error
	action.AwaitTask(user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		errCode, _, rpcErr := GetInfoRpc(task, cmd)
		if rpcErr != nil {
			err = rpcErr
			return
		}
		if errCode < 0 {
			return
		}
		task.User.SetHasGetInfo(true)
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}

func BenchmarkCmd(action base.TaskActionImpl, cmd []string) string {
	var count int64 = 1000
	if len(cmd) >= 1 {
		count, _ = strconv.ParseInt(cmd[0], 10, 32)
	}

	for range count {
		user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
			_ = PingRpc(task)
		})
	}
	return ""
}

func GMCmd(action base.TaskActionImpl, cmd []string) string {
	// 发送登录请求
	var err error
	action.AwaitTask(user_data.CurrentUserRunTaskDefaultTimeout(func(task *user_data.TaskActionUser) {
		_, _, rpcErr := GMRpc(task, cmd)
		if rpcErr != nil {
			err = rpcErr
			return
		}
	}))
	if err != nil {
		return err.Error()
	}
	return ""
}
