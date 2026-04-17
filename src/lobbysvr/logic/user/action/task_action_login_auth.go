package lobbysvr_logic_user_action

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	config "github.com/atframework/atsf4g-go/component/config"
	db "github.com/atframework/atsf4g-go/component/db"
	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	"github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	logic_uuid "github.com/atframework/atsf4g-go/component/uuid"
	libatapp "github.com/atframework/libatapp-go"

	component_dispatcher "github.com/atframework/atsf4g-go/component/dispatcher"
	component_open_platform "github.com/atframework/atsf4g-go/component/open_platform"
	user_controller "github.com/atframework/atsf4g-go/component/user_controller"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	user_auth "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user/auth"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"

	uc "github.com/atframework/atsf4g-go/component/user_controller"
)

var (
	loginAuthLock    sync.Mutex
	loginAuthUserSet = make(map[string]struct{})
)

func tryEnterLoginAuthUser(openId string) bool {
	loginAuthLock.Lock()
	defer loginAuthLock.Unlock()

	if _, exist := loginAuthUserSet[openId]; exist {
		return false
	}

	loginAuthUserSet[openId] = struct{}{}
	return true
}

func leaveLoginAuthUser(openId string) {
	loginAuthLock.Lock()
	delete(loginAuthUserSet, openId)
	loginAuthLock.Unlock()
}

type TaskActionLoginAuth struct {
	user_controller.TaskActionCSBase[*service_protocol.CSLoginAuthReq, *service_protocol.SCLoginAuthRsp]
}

func (t *TaskActionLoginAuth) Name() string {
	return "TaskActionLoginAuth"
}

func (t *TaskActionLoginAuth) AllowNoActor() bool {
	return true
}

func (t *TaskActionLoginAuth) Run(_startData *component_dispatcher.DispatcherStartData) error {
	session := t.GetSession()
	t.GetRpcContext().LogInfo("TaskActionLoginAuth Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", session.GetSessionId()),
	)

	request_body := t.GetRequestBody()
	response_body := t.MutableResponseBody()

	if request_body.GetOpenId() == "" {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		t.GetRpcContext().LogWarn("open_id is empty")
		session.SetUnflushActorLogName("NO_OPENID")
		return nil
	}
	session.SetUnflushActorLogName(request_body.GetOpenId())

	// 认证锁
	if !tryEnterLoginAuthUser(request_body.GetOpenId()) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		t.GetRpcContext().LogWarn("user is logining in another task", "open_id", request_body.GetOpenId())
		return nil
	}

	defer func() {
		leaveLoginAuthUser(request_body.GetOpenId())
	}()

	authTable, result := db.DatabaseTableAccessLoadWithOpenId(t.GetAwaitableContext(), request_body.GetOpenId())
	if result.IsError() && result.GetResponseCode() != int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
		result.LogError(t.GetRpcContext(), "load access table failed", "open_id", request_body.GetOpenId())
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return nil
	}

	// 新用户
	if authTable == nil {
		authTable = &pbdesc.DatabaseTableAccess{
			OpenId: request_body.GetOpenId(),
			ZoneId: 0,
			UserId: 0,
		}
	}
	if authTable.GetZoneId() == 0 {
		authTable.ZoneId = config.GetConfigManager().GetLogicId()
	}

	isNewUser := false
	if authTable.GetUserId() == 0 {
		authTable.UserId, result = logic_uuid.GenerateGlobalUniqueID(t.GetAwaitableContext(),
			private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_ACCOUNT_ID,
			private_protocol_pbdesc.EnGlobalUUIDMinorType_EN_GLOBAL_UUID_MIT_LIMIT_GLOBAL,
			private_protocol_pbdesc.EnGlobalUUIDPatchType_EN_GLOBAL_UUID_PT_DEFAULT,
		)
		if result.IsError() {
			result.LogError(t.GetRpcContext(), "generate user id failed")
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			return nil
		}
		isNewUser = true
	}

	user := uc.UserManagerFindUserAs[*data.User](t.GetRpcContext(), t.GetDispatcher().GetApp(), authTable.GetZoneId(), authTable.GetUserId())
	if !t.checkExistedUser(user) {
		return nil
	}

	// 已登入用户的登入互斥
	defer func() {
		if user != nil {
			user.UnlockLoginTask(t.GetTaskId())
		}
	}()

	lobbySvrCfg := config.GetServerConfig[*private_protocol_config.Readonly_LobbyServerCfg](config.GetConfigManager().GetCurrentConfigGroup())
	if lobbySvrCfg == nil {
		t.LogError("Lobby server config is nil")
	}

	// 认证验证

	switch request_body.GetAccount().GetAccountType() {
	case uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_INTERNAL):
		if !lobbySvrCfg.GetAuth().GetAllowNoAuth() {
			if !t.validateAccessDataPassword(authTable) {
				return nil
			}
		}
	case uint32(public_protocol_pbdesc.EnAccountTypeID_EN_ATI_ACCOUNT_TAPTAP):
		if !t.validateAccessDataOpenPlatform(authTable) {
			return nil
		}
	default:
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetRpcContext().LogWarn("unsupported account type", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(), "account_type", request_body.GetAccount().GetAccountType())
		return nil
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetRpcContext().LogWarn("generate login code failed", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(), "error", err)
		return nil
	}
	loginCode := strings.Replace(uuid.String(), "-", "", -1)

	// 复制本次认证信息
	authTable.LoginCode = loginCode
	authTable.MutableLastAccessData().AccessData = request_body.GetAuthData()
	authTable.MutableLastAccessData().Account = request_body.GetAccount()

	if !lobbySvrCfg.GetAuth().GetAllowNoAuth() {
		weakToken := strings.Replace(uuid.String(), "-", "", -1)
		weakTokenExpireTime := t.GetNow().Add(lobbySvrCfg.GetAuth().GetWeakTokenTimeout().AsDuration())
		user_auth.AddWeakToken(t.GetRpcContext(), weakToken, weakTokenExpireTime, authTable.MutableWeakData())

		response_body.AuthWeak = &public_protocol_pbdesc.DClientAuthWeakData{
			Type: &public_protocol_pbdesc.DClientAuthWeakData_Password{
				Password: &public_protocol_pbdesc.DClientAuthWeakPassword{
					WeakToken:        weakToken,
					ExpiredTimepoint: timestamppb.New(weakTokenExpireTime),
				},
			},
		}
	}
	rpcErr := db.DatabaseTableAccessUpdateOpenId(t.GetAwaitableContext(), authTable)
	if rpcErr.IsError() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetRpcContext().LogWarn("update login code failed", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(), "error", rpcErr)
		return nil
	}

	response_body.LoginCode = loginCode
	response_body.OpenId = request_body.GetOpenId()
	response_body.UserId = authTable.GetUserId()
	response_body.ZoneId = authTable.GetZoneId()
	response_body.IsNewUser = isNewUser
	response_body.VersionType = uint32(public_protocol_common.EnVersionType_EN_VERSION_DEFAULT)

	return nil
}

func (t *TaskActionLoginAuth) checkExistedUser(user *data.User) bool {
	if user == nil {
		return true
	}

	if !user.TryLockLoginTask(t.GetTaskId()) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		t.GetRpcContext().LogWarn("user is logining in another task", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "login_task_id", user.GetLoginTaskId())
		return false
	}
	t.SetUser(user)

	if user.IsWriteable() && user.GetSession() == t.GetSession() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_ALREADY_ONLINE)
		t.GetRpcContext().LogWarn("user already login", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	return true
}

func (t *TaskActionLoginAuth) OnComplete() {
	userImpl := t.GetUser()
	if userImpl == nil {
		return
	}

	user, ok := userImpl.(*data.User)
	if !ok || user == nil {
		t.GetRpcContext().LogWarn("Task user can not convert to data.User", "task_id", t.GetTaskId(), "task_name", t.Name())
		return
	}

	user.UnlockLoginTask(t.GetTaskId())
}

func (t *TaskActionLoginAuth) validateAccessDataPassword(authTable *private_protocol_pbdesc.DatabaseTableAccess) bool {
	request_body := t.GetRequestBody()

	accessSecret := authTable.GetAccessSecret()

	requestPasswordHash := request_body.GetAuthData().GetPassword().GetPasswordHash()
	requestPasswordWeakToken := request_body.GetAuthData().GetPassword().GetWeakToken()
	if requestPasswordHash == "" && requestPasswordWeakToken == "" {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetRpcContext().LogWarn("user login requires password", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId())
		return false
	}

	checkPassed := false
	if requestPasswordWeakToken != "" {
		checkPassed = user_auth.HasPasswordWeakToken(t.GetRpcContext(), requestPasswordWeakToken, authTable.GetWeakData())
	}
	if !checkPassed && requestPasswordHash != "" && accessSecret != "" {
		var err error
		checkPassed, err = user_auth.VerifyPassword(requestPasswordHash, accessSecret)
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
			t.GetRpcContext().LogWarn("verify password failed", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(), "error", err)
		}
	}

	if !checkPassed {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetRpcContext().LogWarn("user password auth failed", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId())
		return false
	}

	return true
}

func (t *TaskActionLoginAuth) validateAccessDataOpenPlatform(authTable *private_protocol_pbdesc.DatabaseTableAccess) bool {
	request_body := t.GetRequestBody()

	opMgr := libatapp.AtappGetModule[component_open_platform.OpenPlatformManager](t.GetRpcContext().GetApp())
	if opMgr == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_INVALID_CHANNEL)
		t.GetRpcContext().LogWarn("OpenPlatformManager is not setup", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId())
		return false
	}

	channelDelegate := opMgr.CreateChannelDelegate(component_open_platform.OpenPlatformAccountType(request_body.GetAccount().GetAccountType()))
	if channelDelegate == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_INVALID_CHANNEL)
		t.GetRpcContext().LogWarn("unsupported open platform account type", "zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(), "account_type", request_body.GetAccount().GetAccountType())
		return false
	}

	_, platformError, rpcResult := channelDelegate.ValidateAuthorization(
		t.GetAwaitableContext(),
		opMgr,
		component_open_platform.MakeOpenPlatformUserKey(request_body.GetOpenId()),
		request_body.GetAccount(),
		request_body.GetAuthData(),
	)

	if rpcResult.IsError() {
		t.SetResponseCode(rpcResult.GetResponseCode())
		t.GetRpcContext().LogWarn("open platform authorization failed",
			"zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(),
			"open_id", request_body.GetOpenId(),
			"account_type", request_body.GetAccount().GetAccountType(),
			"response_code", rpcResult.GetResponseCode(),
			"response_message", rpcResult.GetResponseMessage(),
			"error", rpcResult.GetErrorString(),
		)
		return false
	}

	if channelDelegate.IsError(platformError) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetRpcContext().LogWarn("open platform authorization failed",
			"zone_id", authTable.GetZoneId(), "user_id", authTable.GetUserId(),
			"account_type", request_body.GetAccount().GetAccountType(),
			"platform_error_code", platformError.GetErrorCode(),
			"platform_error_message", platformError.GetErrorMessage(),
			"platform_error_description", platformError.GetErrorDescription(),
		)
		return false
	}

	return true
}
