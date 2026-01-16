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
	obj UserImpl
}

func CreateUserRouterCache(ctx cd.RpcContext, key router.RouterObjectKey) *UserRouterCache {
	// 这个时候openid无效，后面需要再init一次
	ret := &UserRouterCache{
		RouterObjectBase: router.CreateRouterObjectBase(ctx, key),
	}
	cache := CreateUserCache(ctx, key.ZoneID, key.ObjectID, "", ret.GetActorExecutor())
	ret.obj = &cache
	return ret
}

type UserRouterPrivateData struct {
	loginLockTb     *private_protocol_pbdesc.DatabaseTableLoginLock
	loginCASVersion uint64
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
		return cd.CreateRpcResultError(fmt.Errorf("loginTb should not be nil, zone_id: %d, user_id: %d", p.obj.GetZoneId(), p.obj.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if lu.IsNil(p) || !p.obj.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.obj.GetZoneId(), p.obj.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	userTb, userCasVersion, err := db.DatabaseTableUserLoadWithZoneIdUserId(ctx, p.obj.GetZoneId(), p.obj.GetUserId())
	if err.IsError() {
		if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			// 新创建记录初始化
			userTb = new(private_protocol_pbdesc.DatabaseTableUser)
			userTb.OpenId = privateData.openId
			userTb.ZoneId = p.obj.GetZoneId()
			userTb.UserId = p.obj.GetUserId()
			userTb.DataVersion = UserDataCurrentVersion
		} else {
			err.LogError(ctx, "load user table from db failed")
			return err
		}
	}

	// 补全OpenId
	if p.obj.GetOpenId() == "" {
		p.obj.InitOpenId(userTb.GetOpenId())
	}

	// 冲突检测
	expectVersion := privateData.loginLockTb.GetExpectTableUserDbVersion()
	if privateData.loginLockTb.GetLoginZoneId() == p.obj.GetZoneId() && userCasVersion < expectVersion {
		// 版本不对
		ctx.LogWarn("user table version conflict", "db_version", userCasVersion, "expect_version", expectVersion)
		if ctx.GetSysNow().UnixNano() < privateData.loginLockTb.GetExpectTableUserDbTimeout().AsTime().UnixNano() {
			ctx.LogWarn("need retry later for user table version conflict", "db_version", userCasVersion, "expect_version", expectVersion)
			return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_EAGAIN)
		}
	}

	ctx.LogInfo("load user table from db success", "cas_version", userCasVersion)

	// TODO 判断USER表Version是否合法
	// TODO DB 版本防止回退校验

	// 设置路由版本
	p.SetRouterServerId(0, privateData.loginLockTb.GetRouterVersion())

	// LoginLock Table
	p.obj.LoadLoginLockInfo(privateData.loginLockTb)
	p.obj.SetLoginLockCASVersion(privateData.loginCASVersion)

	// Init from DB
	// TODO 版本升级
	result := p.obj.InitFromDB(ctx, userTb)
	if result.IsError() {
		result.LogError(ctx, "init user from db failed")
		return result
	}
	p.obj.SetUserCASVersion(userCasVersion)

	// 更新LoginLock表路由信息
	oldRouterServerId := p.obj.GetLoginLockInfo().GetRouterServerId()
	oldRouterServerVersion := p.obj.GetLoginLockInfo().GetRouterVersion()

	// 更新登录锁信息
	p.obj.GetLoginLockInfo().LoginExpired = ctx.GetSysNow().Unix() +
		config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetLoginCodeValidSec().GetSeconds()
	p.obj.GetLoginLockInfo().LoginZoneId = p.obj.GetZoneId()

	p.obj.GetLoginLockInfo().RouterServerId = uint64(config.GetConfigManager().GetLogicId())
	// 手动版本更新
	p.obj.GetLoginLockInfo().RouterVersion = oldRouterServerVersion + 1

	loginCASVersion := p.obj.GetLoginLockCASVersion()
	loginTableResult := db.DatabaseTableLoginLockReplaceUserId(ctx, p.obj.GetLoginLockInfo(), &loginCASVersion, false)
	if loginTableResult.IsError() {
		// 回滚
		p.obj.GetLoginLockInfo().RouterServerId = oldRouterServerId
		p.obj.GetLoginLockInfo().RouterVersion = oldRouterServerVersion
		result.LogError(ctx, "update login table router server id failed")
		return loginTableResult
	}
	p.obj.SetLoginLockCASVersion(loginCASVersion)
	// 更新路由版本
	p.SetRouterServerId(p.obj.GetLoginLockInfo().GetRouterServerId(), p.obj.GetLoginLockInfo().GetRouterVersion())

	result.LogInfo(ctx, "init user from db success")
	return cd.CreateRpcResultOk()
}

func (p *UserRouterCache) SaveObject(ctx cd.AwaitableContext, _ router.RouterPrivateData) cd.RpcResult {
	if lu.IsNil(p) || !p.obj.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.obj.GetZoneId(), p.obj.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	tryTime := 2
	var err cd.RpcResult
	for tryTime > 0 {
		tryTime--
		if err.IsError() && err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
			// 再拉一次数据库
			loginLockTable, loginLockCASVersion, dbResult := db.DatabaseTableLoginLockLoadWithUserId(ctx, p.obj.GetUserId())
			if dbResult.IsError() {
				dbResult.LogError(ctx, "reload login lock table from db failed")
				return dbResult
			}
			p.obj.LoadLoginLockInfo(loginLockTable)
			p.obj.SetLoginLockCASVersion(loginLockCASVersion)
			err.LogInfo(ctx, "reload login lock table from db success")
		}

		if p.obj.GetLoginLockInfo().GetRouterServerId() != uint64(config.GetConfigManager().GetLogicId()) {
			// 别的地方登录成功 尝试下线
			ctx.LogError("login lock occupied by other router server, cannot save user, need kick off", "router_server_id", p.obj.GetLoginLockInfo().GetRouterServerId())
			s := p.obj.GetUserSession()
			// 在其他设备登入的要把这里的Session踢下线
			if s != nil {
				s.UnbindUser(ctx, p.obj)
				libatapp.AtappGetModule[*SessionManager](ctx.GetApp()).
					RemoveSession(ctx, s.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_TFRAMEHEAD_REASON_SELF_CLOSE), "other device login")
			}
			p.Downgrade(ctx)
			return cd.CreateRpcResultError(fmt.Errorf("login lock occupied by other router server, cannot save user, need kick off, zone_id: %d, user_id: %d", p.obj.GetZoneId(), p.obj.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_OTHER_DEVICE)
		}

		// 冲突检测的版本号设置
		p.obj.GetLoginLockInfo().ExpectTableUserDbVersion = p.obj.GetUserCASVersion() + 1
		p.obj.GetLoginLockInfo().ExpectTableUserDbTimeout = timestamppb.New(ctx.GetSysNow().Add(
			config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetTask().GetCsmsg().GetTimeout().AsDuration()))

		if p.GetRouterSvrId() == 0 {
			// 登出
			oldRouterServerId := p.obj.GetLoginLockInfo().RouterServerId
			oldRouterVersion := p.obj.GetLoginLockInfo().RouterVersion

			p.obj.GetLoginLockInfo().RouterServerId = 0
			// 版本更新
			p.obj.GetLoginLockInfo().RouterVersion = oldRouterVersion + 1

			// 登录锁失效
			p.obj.GetLoginLockInfo().LoginExpired = 0
			loginLockVersin := p.obj.GetLoginLockCASVersion()
			err = db.DatabaseTableLoginLockReplaceUserId(ctx, p.obj.GetLoginLockInfo(), &loginLockVersin, false)
			if err.IsError() {
				p.obj.GetLoginLockInfo().RouterServerId = oldRouterServerId
				p.obj.GetLoginLockInfo().RouterVersion = oldRouterVersion
				if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
					// 重试
					err.LogError(ctx, "save login to db failed DatabaseTableLoginLockUpdateUserId try next time")
					continue
				}
				err.LogError(ctx, "save login to db failed")
				return err
			}
			// 成功
			p.obj.SetLoginLockCASVersion(loginLockVersin)
			p.SetRouterServerId(p.obj.GetLoginLockInfo().GetRouterServerId(), p.obj.GetLoginLockInfo().GetRouterVersion())
			// Logout
			p.obj.OnLogout(ctx)
		} else {
			// 登录锁续期
			p.obj.GetLoginLockInfo().LoginExpired = ctx.GetSysNow().Unix() +
				config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetSession().GetLoginCodeValidSec().GetSeconds()
			oldRouterServerId := p.obj.GetLoginLockInfo().RouterServerId
			oldRouterVersion := p.obj.GetLoginLockInfo().RouterVersion

			if p.GetRouterSvrId() != oldRouterServerId {
				p.obj.GetLoginLockInfo().RouterServerId = p.GetRouterSvrId()
				p.obj.GetLoginLockInfo().RouterVersion = oldRouterVersion + 1
			}

			loginLockVersin := p.obj.GetLoginLockCASVersion()
			err = db.DatabaseTableLoginLockReplaceUserId(ctx, p.obj.GetLoginLockInfo(), &loginLockVersin, false)
			if err.IsError() {
				p.obj.GetLoginLockInfo().RouterServerId = oldRouterServerId
				p.obj.GetLoginLockInfo().RouterVersion = oldRouterVersion
				if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
					// 重试
					err.LogError(ctx, "save login to db failed DatabaseTableLoginLockUpdateUserId try next time")
					continue
				} else {
					err.LogError(ctx, "save login to db failed")
					return err
				}
			} else {
				// 成功
				p.obj.SetLoginLockCASVersion(loginLockVersin)
				p.SetRouterServerId(p.obj.GetLoginLockInfo().GetRouterServerId(), p.obj.GetLoginLockInfo().GetRouterVersion())
			}
		}
		break
	}

	if err.IsError() {
		// 锁处理失败
		ctx.LogError("login lock update failed, cannot save user")
		return err
	}

	// 锁处理完毕
	dstTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result := p.obj.DumpToDB(ctx, dstTb)
	if result.IsError() {
		// 走到这会丢数据
		result.LogError(ctx, "dump user to db failed")
		return result
	}

	userCASVersion := p.obj.GetUserCASVersion()
	ctx.LogInfo("save user to db start", "cas_version", userCASVersion)
	err = db.DatabaseTableUserReplaceZoneIdUserId(ctx, dstTb, &userCASVersion, false)
	p.obj.SetUserCASVersion(userCASVersion)
	if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
		// 重试一次
		err.LogError(ctx, "save user to db failed DatabaseTableUserReplaceZoneIdUserId try next time")
		dstTb = &private_protocol_pbdesc.DatabaseTableUser{}
		result = p.obj.DumpToDB(ctx, dstTb)
		if result.IsError() {
			// 走到这会丢数据
			result.LogError(ctx, "dump user to db failed")
			return result
		}
		ctx.LogInfo("retry save user to db start ", "cas_version", userCASVersion)
		userCASVersion = p.obj.GetUserCASVersion()
		err = db.DatabaseTableUserReplaceZoneIdUserId(ctx, dstTb, &userCASVersion, false)
		p.obj.SetUserCASVersion(userCASVersion)
	}

	if !err.IsError() {
		p.obj.OnSaved(ctx, userCASVersion)
		result.LogInfo(ctx, "save user to db success")
	} else {
		err.LogError(ctx, "save user to db failed", "err", err)
	}
	return err
}
