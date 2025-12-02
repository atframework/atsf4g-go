// Copyright 2025 atframework

package lobbysvr_logic_user_action

import (
	"fmt"
	"log/slog"
	"strconv"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"

	config "github.com/atframework/atsf4g-go/component-config"
	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	router "github.com/atframework/atsf4g-go/component-router"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type TaskActionLogin struct {
	cd.TaskActionCSBase[*service_protocol.CSLoginReq, *service_protocol.SCLoginRsp]

	isNewPlayer bool
}

func (t *TaskActionLogin) Name() string {
	return "TaskActionLogin"
}

func (t *TaskActionLogin) AllowNoActor() bool {
	return true
}

func (t *TaskActionLogin) Run(_startData *cd.DispatcherStartData) error {
	t.GetLogger().Info("TaskActionLogin Run",
		slog.Uint64("task_id", t.GetTaskId()),
		slog.Uint64("session_id", t.GetSession().GetSessionId()),
	)

	request_body := t.GetRequestBody()

	userId := request_body.GetUserId()
	if userId == 0 {
		userIdFromOpenId, err := strconv.ParseUint(request_body.GetOpenId(), 10, 64)
		if err != nil {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
			t.GetLogger().Warn("invalid openid id", "open_id", request_body.GetOpenId(), "error", err)
			return nil
		}
		userId = userIdFromOpenId
	}

	zoneId := t.GetDispatcher().GetApp().GetLogicId()

	csSession := t.GetSession()
	if csSession == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session is required", "zone_id", zoneId, "user_id", userId)
		return nil
	}
	session := csSession.(*uc.Session)
	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session conversion failed", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	// 先查找用户缓存
	user := uc.UserManagerFindUserAs[*data.User](t.GetRpcContext(), t.GetDispatcher().GetApp(), zoneId, userId)
	// 检查创建状态
	if !t.checkExistedUser(user) {
		return nil
	}
	// 这里开始 如有user 则创建加锁成功

	// 登入鉴权
	authTable, _ := db.DatabaseTableAccessLoadWithZoneIdUserId(t.GetAwaitableContext(), zoneId, userId)
	loginCode := ""
	if authTable != nil {
		loginCode = authTable.GetLoginCode()
	}
	if loginCode == "" || loginCode != request_body.GetLoginCode() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
		t.GetLogger().Warn("invalid login code", "zone_id", zoneId, "user_id", userId, "code", loginCode, "req", request_body.GetLoginCode())
		return nil
	}

	// TODO: 登入鉴权Token有效期

	// 如果正在登出则要等登出结束重新获取
	if user != nil && user.IsWriteable() {
		result := t.awaitLogoutIoTask(t.GetAwaitableContext(), user)
		if result.IsError() {
			result.LogError(t.GetRpcContext(), "await logout io task failed", "zone_id", zoneId, "user_id", userId)
			if result.GetResponseCode() < 0 {
				t.SetResponseCode(result.GetResponseCode())
			} else {
				t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			}
			return nil
		}
		user = uc.UserManagerFindUserAs[*data.User](t.GetRpcContext(), t.GetDispatcher().GetApp(), zoneId, userId)
	}

	// 如果是在线用户，走替换Session流程
	if user != nil && user.IsWriteable() {
		t.replaceSession(user, session)
		return nil
	}

	// 如果有缓存要强制失效，因为可能其他地方登入了，这时候也不能复用缓存
	libatapp.AtappGetModule[*uc.UserManager](t.GetAwaitableContext().GetApp()).Remove(t.GetAwaitableContext(), zoneId, userId, nil, true)

	loginTb, loginCASVersion, result := t.kickoffOtherSession(t.GetAwaitableContext(), zoneId, userId)
	if result.IsError() {
		if result.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			// 初次全区服登录
			loginTb = &private_protocol_pbdesc.DatabaseTableLoginLock{
				UserId: userId,
			}
			loginCASVersion = 0
			result = db.DatabaseTableLoginLockUpdateUserId(t.GetAwaitableContext(), loginTb, &loginCASVersion, false)
			if result.IsError() {
				if result.GetResponseCode() < 0 {
					t.SetResponseCode(result.GetResponseCode())
				} else {
					t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
				}
				return nil
			}
		} else {
			if result.GetResponseCode() < 0 {
				t.SetResponseCode(result.GetResponseCode())
			} else {
				t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
			}
			if result.Error != nil {
				result.LogError(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
			} else {
				result.LogWarn(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
			}
			return nil
		}
	}

	// 可以开始尝试登录了
	routerVersion := loginTb.RouterVersion
	// 填充LoginCode
	loginTb.LoginCode = request_body.GetLoginCode()

	user, result = uc.UserManagerCreateUserAs(
		t.GetAwaitableContext().GetApp(), t.GetAwaitableContext(), zoneId, userId, request_body.GetOpenId(),
		loginTb, routerVersion, loginCASVersion, func(user *data.User) {
			// 填充客户端数据
			accountInfo := user.MutableAccountInfo()
			accountInfo.AccountType = request_body.GetAccount().GetAccountType()
			// Access: request_body.Account.Access,
			accountInfo.Profile = &public_protocol_pbdesc.DUserProfile{
				OpenId: request_body.GetOpenId(),
				UserId: userId,
			}
			accountInfo.ChannelId = request_body.GetAccount().GetChannelId()
		}, func(user *data.User) cd.RpcResult {
			if user == nil {
				return cd.CreateRpcResultError(
					fmt.Errorf("user is nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
			}

			if !user.TryLockLoginTask(t.GetTaskId()) {
				return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
			}
			t.SetUser(user)

			return cd.CreateRpcResultOk()
		},
		// 不需要这里解锁，整个task执行完后会解锁的
		nil)
	if result.IsError() {
		if result.GetResponseCode() < 0 {
			t.SetResponseCode(result.GetResponseCode())
		} else {
			t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
		}
		result.LogError(t.GetRpcContext(), "create user failed", "zone_id", zoneId, "user_id", userId)
		return nil
	}

	t.isNewPlayer = user.IsNewUser()

	// 数据复制
	if request_body.GetClientInfo() != nil {
		*user.MutableClientInfo() = *request_body.GetClientInfo()
	}

	// session绑定
	t.GetSession().BindUser(t.GetRpcContext(), user)

	// 登入初始化
	user.LoginInit(t.GetRpcContext())

	return nil
}

func (t *TaskActionLogin) checkExistedUser(user *data.User) bool {
	if user == nil {
		return true
	}

	if !user.TryLockLoginTask(t.GetTaskId()) {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		t.GetLogger().Warn("user is logining in another task", "zone_id", user.GetZoneId(), "user_id", user.GetUserId(), "login_task_id", user.GetLoginTaskId())
		return false
	}
	t.SetUser(user)

	if user.IsWriteable() && user.GetSession() == t.GetSession() {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_ALREADY_ONLINE)
		t.GetLogger().Warn("user already login", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	return true
}

func (t *TaskActionLogin) replaceSession(user *data.User, session *uc.Session) bool {
	if user == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("user is required")
		return false
	}

	if session == nil {
		t.SetResponseError(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		t.GetLogger().Error("session is required", "zone_id", user.GetZoneId(), "user_id", user.GetUserId())
		return false
	}

	// 先解锁旧的Session
	user.BindSession(t.GetRpcContext(), session)
	return true
}

func (t *TaskActionLogin) awaitLogoutIoTask(ctx cd.AwaitableContext, user *data.User) cd.RpcResult {
	cache := uc.GetUserRouterManager(ctx.GetApp()).GetCache(router.RouterObjectKey{
		TypeID:   uint32(public_protocol_pbdesc.EnRouterObjectType_EN_ROT_PLAYER),
		ZoneID:   user.GetZoneId(),
		ObjectID: user.GetUserId(),
	})
	if cache == nil {
		return cd.CreateRpcResultOk()
	}
	if cache.CheckFlag(router.FlagRemovingObject) && cache.IsIORunning() {
		return cache.AwaitIOTask(ctx)
	}
	return cd.CreateRpcResultOk()
}

func (t *TaskActionLogin) kickoffOtherSession(ctx cd.AwaitableContext, zoneId uint32, userId uint64) (table *private_protocol_pbdesc.DatabaseTableLoginLock, CASVersion uint64, retResult cd.RpcResult) {
	table, CASVersion, retResult = db.DatabaseTableLoginLockLoadWithUserId(ctx, userId)
	if retResult.IsError() {
		return
	}

	// 踢线检查
	if table.GetRouterServerId() == 0 {
		return
	}

	// 被占用中 尝试是否可以抢占
	retResult = t.playerKickoff(ctx)
	if retResult.IsError() {
		ctx.LogError("player kickoff failed", "zone_id", zoneId, "user_id", userId,
			"router_server_id", table.GetRouterServerId())
		// 不同登入进程占用 检查是否过期
		if ctx.GetApp().GetSysNow().Unix() < table.GetLoginExpired() {
			// 未过期 则拒绝本次登入
			retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ALREADY_LOGIN)
			return
		} else {
			// 已过期 可以继续往下走
			ctx.LogWarn("login lock expired, can continue login", "zone_id", zoneId, "user_id", userId,
				"router_server_id", table.GetRouterServerId(), "login_expired", table.GetLoginExpired(),
				"now", ctx.GetApp().GetSysNow().Unix())
			table.RouterServerId = 0
			retResult = cd.CreateRpcResultOk()
		}
	} else {
		oldServerId := table.GetRouterServerId()
		// 踢人成功 再次拉取数据
		table, CASVersion, retResult = db.DatabaseTableLoginLockLoadWithUserId(ctx, userId)
		if retResult.IsError() {
			ctx.LogError("reload login lock failed after kickoff", "zone_id", zoneId, "user_id", userId)
			retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ALREADY_LOGIN)
			return
		}

		// 可能目标服务器重启后没有这个玩家的数据而直接忽略请求直接返回成功
		// 这时候走故障恢复流程，直接把router_server_id设成0即可
		if table.GetRouterServerId() != 0 && oldServerId != table.GetRouterServerId() {
			// 被其他服务器顶掉了
			ctx.LogWarn("player has been kicked off by other server", "zone_id", zoneId, "user_id", userId,
				"old_router_server_id", oldServerId, "new_router_server_id", table.GetRouterServerId())
			retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ALREADY_LOGIN)
			return
		}
		table.RouterServerId = 0
	}
	return
}

func (t *TaskActionLogin) playerKickoff(ctx cd.AwaitableContext) cd.RpcResult {
	// TODO RPC 踢人
	return cd.CreateRpcResultOk()
}

func (t *TaskActionLogin) OnSuccess() {
	response_body := t.MutableResponseBody()

	userImpl := t.GetUser()
	if userImpl == nil {
		return
	}

	user, ok := userImpl.(*data.User)
	if !ok || user == nil {
		return
	}

	response_body.ZoneId = user.GetZoneId()
	response_body.VersionType = uint32(user.GetAccountInfo().GetVersionType())

	response_body.HeartbeatInterval = config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetHeartbeat().GetInterval().GetSeconds()

	response_body.IsNewUser = t.isNewPlayer

	// 事件和刷新
	user.RefreshLimit(t.GetRpcContext(), t.GetNow())
}

func (t *TaskActionLogin) OnComplete() {
	userImpl := t.GetUser()
	if userImpl == nil {
		return
	}

	user, ok := userImpl.(*data.User)
	if !ok || user == nil {
		t.GetLogger().Warn("Task user can not convert to data.User", "task_id", t.GetTaskId(), "task_name", t.Name())
		return
	}

	// 登入过程中产生的脏数据不需要推送
	user.CleanupClientDirtyCache(t.GetRpcContext())

	user.UnlockLoginTask(t.GetTaskId())
}
