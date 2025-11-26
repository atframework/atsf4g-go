package atframework_component_user_controller

import (
	"fmt"
	"log/slog"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component-router"
	libatapp "github.com/atframework/libatapp-go"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type UserRouterCache struct {
	router.RouterObjectBase
	UserImpl
}

func CreateUserRouterCache(ctx cd.RpcContext, key router.RouterObjectKey) *UserRouterCache {
	// 这个时候openid无效，后面需要再init一次
	cache := CreateUserCache(ctx, key.ZoneID, key.ObjectID, "")
	return &UserRouterCache{
		RouterObjectBase: router.CreateRouterObjectBase(ctx, key),
		UserImpl:         &cache,
	}
}

type UserRouterPrivateData struct {
	loginLockTb     *private_protocol_pbdesc.DatabaseTableLoginLock
	loginCASVersion uint64
	routerVersion   uint64
	openId          string
}

func (p *UserRouterPrivateData) RouterPrivateDataImpl() {}

func (p *UserRouterCache) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("UserRouterCache Key %v", p.GetKey()))
}

func (p *UserRouterCache) PullCache(ctx cd.AwaitableContext, privateData router.RouterPrivateData) cd.RpcResult {
	if lu.IsNil(privateData) {
		var privateData UserRouterPrivateData
		return p.pullCache(ctx, &privateData)
	}
	return p.pullCache(ctx, privateData.(*UserRouterPrivateData))
}

func (p *UserRouterCache) pullCache(ctx cd.AwaitableContext, privateData *UserRouterPrivateData) cd.RpcResult {
	return p.RouterObjectBase.PullCache(ctx, privateData)
}

func (p *UserRouterCache) PullObject(ctx cd.AwaitableContext, privateData router.RouterPrivateData) cd.RpcResult {
	if lu.IsNil(privateData) {
		var privateData UserRouterPrivateData
		return p.pullObject(ctx, &privateData)
	}
	return p.pullObject(ctx, privateData.(*UserRouterPrivateData))
}

func (p *UserRouterCache) pullObject(ctx cd.AwaitableContext, privateData *UserRouterPrivateData) cd.RpcResult {
	// 完成后可写
	if privateData.loginLockTb == nil {
		return cd.CreateRpcResultError(fmt.Errorf("loginTb should not be nil, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if lu.IsNil(p) || !p.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	userTb, userCasVersion, err := db.DatabaseTableUserLoadWithZoneIdUserId(ctx, p.GetZoneId(), p.GetUserId())
	if err.IsError() {
		if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			// 新创建记录初始化
			userTb = new(private_protocol_pbdesc.DatabaseTableUser)
			userTb.OpenId = privateData.openId
			userTb.ZoneId = p.GetZoneId()
			userTb.UserId = p.GetUserId()
			userTb.DataVersion = UserDataCurrentVersion
		} else {
			err.LogError(ctx, "load user table from db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
			return err
		}
	}

	// 补全OpenId
	if p.GetOpenId() == "" {
		p.InitOpenId(userTb.GetOpenId())
	}
	p.SetUserCASVersion(userCasVersion)

	// 冲突检测
	expectVersion := privateData.loginLockTb.GetExpectTableUserDbVersion()
	if userCasVersion < expectVersion {
		// 版本不对
		ctx.LogWarn("user table version conflict", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "db_version", userCasVersion, "expect_version", expectVersion)
		if ctx.GetSysNow().UnixNano() < privateData.loginLockTb.GetExpectTableUserDbTimeout().AsTime().UnixNano() {
			ctx.LogWarn("need retry later for user table version conflict", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "db_version", userCasVersion, "expect_version", expectVersion)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_EAGAIN)
		}
	}

	ctx.LogInfo("load user table from db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "cas_version", userCasVersion)

	// TODO 判断USER表Version是否合法
	// TODO DB 版本防止回退校验

	// 设置路由版本
	p.SetRouterServerId(0, privateData.loginLockTb.GetRouterVersion())

	// LoginLock Table
	p.LoadLoginLockInfo(privateData.loginLockTb)
	p.SetLoginLockCASVersion(privateData.loginCASVersion)

	// Init from DB
	// TODO 版本升级
	result := p.InitFromDB(ctx, userTb)
	if result.IsError() {
		result.LogError(ctx, "init user from db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return result
	}

	// 更新LoginLock表路由信息
	oldRouterServerId := p.GetLoginLockInfo().GetRouterServerId()
	oldRouterServerVersion := p.GetLoginLockInfo().GetRouterVersion()

	// 更新登录锁信息
	p.GetLoginLockInfo().LoginExpired = ctx.GetSysNow().Unix() +
		config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetSession().GetLoginCodeValidSec().GetSeconds()
	p.GetLoginLockInfo().LoginZoneId = p.GetZoneId()

	p.GetLoginLockInfo().RouterServerId = uint64(ctx.GetApp().GetLogicId())
	// 手动版本更新
	p.GetLoginLockInfo().RouterVersion = oldRouterServerVersion + 1

	loginCASVersion := p.GetLoginLockCASVersion()
	loginTableResult := db.DatabaseTableLoginLockUpdateUserId(ctx, p.GetLoginLockInfo(), &loginCASVersion, false)
	if loginTableResult.IsError() {
		// 回滚
		p.GetLoginLockInfo().RouterServerId = oldRouterServerId
		p.GetLoginLockInfo().RouterVersion = oldRouterServerVersion
		result.LogError(ctx, "update login table router server id failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return loginTableResult
	}
	p.SetLoginLockCASVersion(loginCASVersion)
	// 更新路由版本
	p.SetRouterServerId(p.GetLoginLockInfo().GetRouterServerId(), p.GetLoginLockInfo().GetRouterVersion())

	result.LogInfo(ctx, "init user from db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())

	return cd.CreateRpcResultOk()
}

func (p *UserRouterCache) SaveObject(ctx cd.AwaitableContext, _ router.RouterPrivateData) cd.RpcResult {
	if lu.IsNil(p) || !p.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	tryTime := 2
	var err cd.RpcResult
	for tryTime > 0 {
		tryTime--
		if err.IsError() && err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
			// 再拉一次数据库
			loginLockTable, loginLockCASVersion, dbResult := db.DatabaseTableLoginLockLoadWithUserId(ctx, p.GetUserId())
			if dbResult.IsError() {
				dbResult.LogError(ctx, "reload login lock table from db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
				return dbResult
			}
			p.LoadLoginLockInfo(loginLockTable)
			p.SetLoginLockCASVersion(loginLockCASVersion)
			err.LogInfo(ctx, "reload login lock table from db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		}

		if p.GetLoginLockInfo().GetRouterServerId() != uint64(ctx.GetApp().GetLogicId()) {
			// 别的地方登录成功 尝试下线
			ctx.LogError("login lock occupied by other router server, cannot save user, need kick off", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "router_server_id", p.GetLoginLockInfo().GetRouterServerId())
			s := p.UserImpl.GetUserSession()
			// 在其他设备登入的要把这里的Session踢下线
			if s != nil {
				s.UnbindUser(ctx, p.UserImpl)
				libatapp.AtappGetModule[*SessionManager](GetReflectTypeSessionManager(), ctx.GetApp()).
					RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "other device login")
			}
			p.Downgrade(ctx)
			return cd.CreateRpcResultError(fmt.Errorf("login lock occupied by other router server, cannot save user, need kick off, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		}

		// 冲突检测的版本号设置
		p.GetLoginLockInfo().ExpectTableUserDbVersion = p.GetUserCASVersion() + 1
		p.GetLoginLockInfo().ExpectTableUserDbTimeout = timestamppb.New(ctx.GetSysNow().Add(
			config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetTask().GetCsmsg().GetTimeout().AsDuration()))

		if p.GetRouterSvrId() == 0 {
			// 登出
			oldRouterServerId := p.GetLoginLockInfo().RouterServerId
			oldRouterVersion := p.GetLoginLockInfo().RouterVersion

			p.GetLoginLockInfo().RouterServerId = 0
			// 版本更新
			p.GetLoginLockInfo().RouterVersion = oldRouterVersion + 1

			// 登出时间由上层逻辑设置
			loginLockVersin := p.GetLoginLockCASVersion()
			err = db.DatabaseTableLoginLockUpdateUserId(ctx, p.GetLoginLockInfo(), &loginLockVersin, false)
			if err.IsError() {
				p.GetLoginLockInfo().RouterServerId = oldRouterServerId
				p.GetLoginLockInfo().RouterVersion = oldRouterVersion
				if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
					// 重试
					err.LogError(ctx, "save login to db failed DatabaseTableLoginLockUpdateUserId try next time", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
					continue
				} else {
					err.LogError(ctx, "save login to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
					return err
				}
			} else {
				// 成功
				p.SetLoginLockCASVersion(loginLockVersin)
				p.SetRouterServerId(p.GetLoginLockInfo().GetRouterServerId(), p.GetLoginLockInfo().GetRouterVersion())
			}
		} else {
			// 登录续期 LoginCodeExpired 由上层逻辑设置
			oldRouterServerId := p.GetLoginLockInfo().RouterServerId
			oldRouterVersion := p.GetLoginLockInfo().RouterVersion

			if p.GetRouterSvrId() != oldRouterServerId {
				p.GetLoginLockInfo().RouterServerId = p.GetRouterSvrId()
				p.GetLoginLockInfo().RouterVersion = oldRouterVersion + 1
			}

			loginLockVersin := p.GetLoginLockCASVersion()
			err = db.DatabaseTableLoginLockUpdateUserId(ctx, p.GetLoginLockInfo(), &loginLockVersin, false)
			if err.IsError() {
				p.GetLoginLockInfo().RouterServerId = oldRouterServerId
				p.GetLoginLockInfo().RouterVersion = oldRouterVersion
				if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
					// 重试
					err.LogError(ctx, "save login to db failed DatabaseTableLoginLockUpdateUserId try next time", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
					continue
				} else {
					err.LogError(ctx, "save login to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
					return err
				}
			} else {
				// 成功
				p.SetLoginLockCASVersion(loginLockVersin)
				p.SetRouterServerId(p.GetLoginLockInfo().GetRouterServerId(), p.GetLoginLockInfo().GetRouterVersion())
			}
		}
		break
	}

	if err.IsError() {
		// 锁处理失败
		ctx.LogError("login lock update failed, cannot save user", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return err
	}

	// 锁处理完毕
	dstTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result := p.DumpToDB(ctx, dstTb)
	if result.IsError() {
		// 走到这会丢数据
		result.LogError(ctx, "dump user to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return result
	}

	userCASVersion := p.GetUserCASVersion()
	ctx.LogInfo("save user to db start", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "cas_version", userCASVersion)
	err = db.DatabaseTableUserUpdateZoneIdUserId(ctx, dstTb, &userCASVersion, false)
	p.SetUserCASVersion(userCASVersion)
	if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
		// 重试一次
		err.LogError(ctx, "save user to db failed DatabaseTableUserUpdateZoneIdUserId try next time", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		dstTb = &private_protocol_pbdesc.DatabaseTableUser{}
		result = p.DumpToDB(ctx, dstTb)
		if result.IsError() {
			// 走到这会丢数据
			result.LogError(ctx, "dump user to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
			return result
		}
		ctx.LogInfo("retry save user to db start ", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "cas_version", userCASVersion)
		userCASVersion = p.GetUserCASVersion()
		err = db.DatabaseTableUserUpdateZoneIdUserId(ctx, dstTb, &userCASVersion, false)
		p.SetUserCASVersion(userCASVersion)
	}

	if !err.IsError() {
		p.OnSaved(ctx, userCASVersion)
		result.LogInfo(ctx, "save user to db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
	} else {
		err.LogError(ctx, "save user to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "err", err)
	}
	return err
}
