package atframework_component_user_controller

import (
	"fmt"
	"log/slog"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	router "github.com/atframework/atsf4g-go/component-router"
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
	loginTb         *private_protocol_pbdesc.DatabaseTableLogin
	loginCASVersion uint64
	loginTbVersion  uint64
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
	if privateData.loginTb == nil {
		return cd.CreateRpcResultError(fmt.Errorf("loginTb should not be nil, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if lu.IsNil(p) || !p.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	userTb, casVersion, err := db.DatabaseTableUserLoadWithZoneIdUserId(ctx, p.GetZoneId(), p.GetUserId())
	if err.IsError() {
		if err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND) {
			// 新创建得记录初始化
			userTb = new(private_protocol_pbdesc.DatabaseTableUser)
			userTb.AccountData = &private_protocol_pbdesc.AccountInformation{
				AccountType: privateData.loginTb.GetAccount().GetAccountType(),
				Access:      privateData.loginTb.GetAccount().GetAccess(),
				Profile: &public_protocol_pbdesc.DUserProfile{
					OpenId: privateData.loginTb.GetOpenId(),
					UserId: p.GetUserId(),
				},
				ChannelId:   privateData.loginTb.GetAccount().GetChannelId(),
				VersionType: privateData.loginTb.GetAccount().GetVersionType(),
			}
			userTb.UserData = &private_protocol_pbdesc.UserData{
				UserLevel:       1,
				SessionSequence: 1,
			}
			userTb.DataVersion = UserDataCurrentVersion
		} else {
			err.LogError(ctx, "load user table from db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
			return err
		}
	}
	// TODO 修复数据

	userTb.OpenId = privateData.loginTb.GetOpenId()
	userTb.ZoneId = p.GetZoneId()
	userTb.UserId = p.GetUserId()

	ctx.LogInfo("load user table from db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId(), "cas_version", casVersion)
	if p.GetOpenId() == "" {
		p.InitOpenId(userTb.GetOpenId())
	}

	p.SetUserCASVersion(casVersion)

	// TODO 冲突检测
	// TODO 判断USER表Version是否合法
	// TODO DB 版本防止回退校验

	// 拉取玩家数据
	// 设置路由ID
	p.SetRouterServerId(privateData.loginTb.GetRouterServerId(), privateData.loginTb.GetRouterVersion())

	// Login Table
	p.LoadLoginInfo(privateData.loginTb, privateData.loginTbVersion)
	p.SetLoginCASVersion(privateData.loginCASVersion)

	// Init from DB
	result := p.InitFromDB(ctx, userTb)
	if result.IsError() {
		result.LogError(ctx, "init user from db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return result
	}

	// 如果router server id是0则设置为本地的登入地址
	if 0 == p.GetRouterSvrId() {
		oldRouterServerId := p.GetLoginInfo().GetRouterServerId()
		oldRouterServerVersion := p.GetLoginInfo().GetRouterVersion()
		p.GetLoginInfo().RouterServerId = uint64(ctx.GetApp().GetLogicId())
		p.GetLoginInfo().RouterVersion = oldRouterServerVersion + 1

		// 新登入则设置登入时间
		p.UpdateLoginData(ctx)

		loginCASVersion := p.GetLoginCASVersion()
		loginTableResult := db.DatabaseTableLoginUpdateZoneIdUserId(ctx, p.GetLoginInfo(), &loginCASVersion)
		if loginTableResult.IsError() {
			// 回滚
			p.GetLoginInfo().RouterServerId = oldRouterServerId
			p.GetLoginInfo().RouterVersion = oldRouterServerVersion
			result.LogError(ctx, "update login table router server id failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
			return loginTableResult
		}
		p.SetLoginCASVersion(loginCASVersion)
		p.SetRouterServerId(p.GetLoginInfo().GetRouterServerId(), p.GetLoginInfo().GetRouterVersion())
	}
	result.LogInfo(ctx, "init user from db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())

	return cd.CreateRpcResultOk()
}

func (p *UserRouterCache) SaveObject(ctx cd.AwaitableContext, _ router.RouterPrivateData) cd.RpcResult {
	if lu.IsNil(p) || !p.CanBeWriteable() {
		return cd.CreateRpcResultError(fmt.Errorf("user router cache is not writeable, zone_id: %d, user_id: %d", p.GetZoneId(), p.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_ROUTER_OBJECT_NOT_WRITEABLE)
	}

	// 保存Login
	loginVersion := p.GetLoginCASVersion()
	err := db.DatabaseTableLoginUpdateZoneIdUserId(ctx, p.GetLoginInfo(), &loginVersion)
	if err.IsError() && err.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
		err.LogError(ctx, "save login to db failed EN_ERR_DB_CAS_CHECK_FAILED try next time", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		db.DatabaseTableLoginUpdateZoneIdUserId(ctx, p.GetLoginInfo(), &loginVersion)
	}
	p.SetLoginCASVersion(loginVersion)

	// TODO 登出流程 or 登录续期

	dstTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result := p.DumpToDB(ctx, dstTb)

	if result.IsError() {
		result.LogError(ctx, "dump user to db failed", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
		return result
	}

	userCASVersion := p.GetUserCASVersion()
	err = db.DatabaseTableUserUpdateZoneIdUserId(ctx, dstTb, &userCASVersion)
	p.SetUserCASVersion(userCASVersion)

	p.OnSaved(ctx, userCASVersion)

	result.LogInfo(ctx, "save user to db success", "zone_id", p.GetZoneId(), "user_id", p.GetUserId())
	return cd.CreateRpcResultOk()
}
