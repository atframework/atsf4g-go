package atframework_component_user_controller

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	atfw_utils_fs "github.com/atframework/atframe-utils-go/file_system"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"google.golang.org/protobuf/proto"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type CreateUserCallback func(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl

type UserManager struct {
	_ noCopy

	userLock sync.RWMutex
	users    map[uint32]*map[uint64]UserImpl

	createUserCallback CreateUserCallback
}

func createUserManager() *UserManager {
	ret := &UserManager{
		users: make(map[uint32]*map[uint64]UserImpl),
	}

	ret.createUserCallback = func(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl {
		ret := CreateUserCache(ctx, zoneId, userId, openId)
		return &ret
	}

	return ret
}

var GlobalUserManager = createUserManager()

func (um *UserManager) SetCreateUserCallback(callback CreateUserCallback) {
	if callback != nil {
		um.createUserCallback = callback
	}
}

func (um *UserManager) replace(ctx *cd.RpcContext, u UserImpl) {
	if u == nil {
		return
	}

	um.userLock.RLock()
	defer um.userLock.RUnlock()

	ctx.LogInfo("user removed", "zone_id", u.GetZoneId(), "user_id", u.GetUserId())

	uidMap, ok := um.users[u.GetZoneId()]
	if !ok {
		m := &map[uint64]UserImpl{}
		(*m)[u.GetUserId()] = u
		um.users[u.GetZoneId()] = m
		return
	}

	(*uidMap)[u.GetUserId()] = u
}

func (um *UserManager) Find(zoneID uint32, userID uint64) UserImpl {
	um.userLock.RLock()
	defer um.userLock.RUnlock()

	uidMap, ok := um.users[zoneID]
	if !ok {
		return nil
	}
	user, ok := (*uidMap)[userID]
	if !ok {
		return nil
	}
	return user
}

func (um *UserManager) Remove(ctx *cd.RpcContext, zoneID uint32, userID uint64, checked UserImpl, _forceKickoff bool) UserImpl {
	um.userLock.Lock()
	defer um.userLock.Unlock()

	uidMap, ok := um.users[zoneID]
	if !ok {
		return nil
	}
	user, ok := (*uidMap)[userID]
	if !ok {
		return nil
	}

	if checked != nil && user != checked {
		return nil
	}

	ctx.LogInfo("user removed", "zone_id", zoneID, "user_id", userID)
	delete(*uidMap, userID)
	if len(*uidMap) == 0 {
		delete(um.users, zoneID)
	}

	if user.IsWriteable() {
		um.internalSave(ctx, user)
	}

	return user
}

func UserManagerFindUserAs[T UserImpl](um *UserManager, zoneID uint32, userID uint64) T {
	userImpl := um.Find(zoneID, userID)
	if userImpl == nil {
		var zero T
		return zero
	}
	casted, ok := userImpl.(T)
	if !ok {
		var zero T
		return zero
	}

	return casted
}

func UserManagerCreateUserAs[T UserImpl](ctx *cd.RpcContext,
	um *UserManager, zoneID uint32, userID uint64, openID string,
	loginTb *private_protocol_pbdesc.DatabaseTableLogin,
	loginTbVersion uint64,
	tryLockUserResource func(user T) cd.RpcResult,
	unlockUserResource func(user T),
) (T, cd.RpcResult) {
	var zero T
	if um == nil || zoneID <= 0 || userID <= 0 || loginTb == nil {
		return zero, cd.CreateRpcResultError(fmt.Errorf("invalid param"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// TODO: 托管给路由系统，执行数据库读取
	u := um.Find(zoneID, userID)

	defer func() {
		if u != nil && unlockUserResource != nil {
			unlockUserResource(u.(T))
		}
	}()

	var ret T
	if u == nil {
		u = um.createUserCallback(ctx, zoneID, userID, openID)
		if u == nil {
			return zero, cd.CreateRpcResultError(fmt.Errorf("invalcan not create userid param"), public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_CREATE_PLAYER_FAILED)
		}
		convertRet, ok := u.(T)
		if !ok {
			return zero, cd.CreateRpcResultError(fmt.Errorf("user type mismatch, zone_id: %d, user_id: %d, type: %T", zoneID, userID, u), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		ret = convertRet

		if tryLockUserResource != nil {
			result := tryLockUserResource(u.(T))
			if result.IsError() {
				unlockUserResource = nil
				return zero, result
			}
		}

		um.replace(ctx, u)
	} else {
		convertRet, ok := u.(T)
		if !ok {
			return zero, cd.CreateRpcResultError(fmt.Errorf("user type mismatch, zone_id: %d, user_id: %d, type: %T", zoneID, userID, u), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		ret = convertRet

		if tryLockUserResource != nil {
			result := tryLockUserResource(u.(T))
			if result.IsError() {
				unlockUserResource = nil
				return zero, result
			}
		}
	}

	result := UserLoadUserTableFromFile(ctx, u, loginTb, loginTbVersion)
	if result.IsError() {
		return zero, result
	}

	// 路由系统外逻辑

	// 创建初始化
	if u.GetLoginVersion() <= 0 {
		// 新用户初始化逻辑
		u.CreateInit(u, ctx, uint32(public_protocol_common.EnVersionType_EN_VERSION_DEFAULT))

		// 设置版本号
		u.GetLoginInfo().RouterVersion = 0
		// 更新Login Table版本号
		u.LoadLoginInfo(u, u.GetLoginInfo(), u.GetLoginInfo().RouterVersion)

		result = um.internalSave(ctx, u)
		if result.IsError() {
			return zero, result
		}
	}

	return ret, cd.CreateRpcResultOk()
}

// TODO: 临时的鉴权数据读取
func UserGetAuthDataFromFile(ctx *cd.RpcContext, zoneID uint32, userID uint64) (string, string) {
	accessFilePath := fmt.Sprintf("../data/%d-%d.access.db", zoneID, userID)
	if _, serr := os.Stat(accessFilePath); serr != nil {
		return "", ""
	}

	af, err := os.Open(accessFilePath)
	if err != nil {
		return "", ""
	}
	defer af.Close()

	var accessSecret string
	var loginCode string
	scanner := bufio.NewScanner(af)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			continue
		}

		if loginCode == "" {
			loginCode = text
			continue
		}

		accessSecret = text
		break
	}

	return accessSecret, loginCode
}

// TODO: 临时的鉴权数据更新
func UserUpdateAuthDataToFile(ctx *cd.RpcContext, zoneID uint32, userID uint64, accessSecret string, loginCode string) error {
	dataDir := "../data"
	accessFilePath := fmt.Sprintf("../data/%d-%d.access.db", zoneID, userID)

	if _, serr := os.Stat(dataDir); serr != nil {
		os.MkdirAll(dataDir, 0o755)
	}

	af, err := os.OpenFile(accessFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer af.Close()

	writer := bufio.NewWriter(af)

	_, err = writer.WriteString(loginCode + "\n")
	if err != nil {
		return err
	}

	_, err = writer.WriteString(accessSecret + "\n")
	if err != nil {
		return err
	}

	writer.Flush()

	return nil
}

// TODO: 临时的数据读取
func UserLoadLoginTableFromFile(ctx *cd.RpcContext, zoneID uint32, userID uint64) (*private_protocol_pbdesc.DatabaseTableLogin, cd.RpcResult) {
	loginTbFilePath := fmt.Sprintf("../data/%d-%d.login.db", zoneID, userID)

	if _, serr := os.Stat(loginTbFilePath); serr != nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND)
	}

	ldata, err := atfw_utils_fs.ReadAllContent(loginTbFilePath)
	if err != nil {
		return nil, cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	loginTb := &private_protocol_pbdesc.DatabaseTableLogin{}
	if err = proto.Unmarshal(ldata, loginTb); err != nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("failed to unmarshal login db data: %w", err), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
	}

	ctx.LogInfo("load login table from db success", "zone_id", zoneID, "user_id", userID)
	return loginTb, cd.CreateRpcResultOk()
}

// TODO: 临时的数据读取
func UserLoadUserTableFromFile(ctx *cd.RpcContext, u UserImpl, loginTb *private_protocol_pbdesc.DatabaseTableLogin, loginTbVersion uint64) cd.RpcResult {
	if u == nil {
		return cd.CreateRpcResultError(fmt.Errorf("user should not be nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if loginTb == nil {
		return cd.CreateRpcResultError(fmt.Errorf("loginTb should not be nil, zone_id: %d, user_id: %d", u.GetZoneId(), u.GetUserId()), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	userTb := &private_protocol_pbdesc.DatabaseTableUser{}

	userTbFilePath := fmt.Sprintf("../data/%d-%d.user.db", u.GetZoneId(), u.GetUserId())
	if _, serr := os.Stat(userTbFilePath); serr != nil {
		// 新创建得记录初始化
		userTb.AccountData = &private_protocol_pbdesc.AccountInformation{
			AccountType: loginTb.GetAccount().GetAccountType(),
			Access:      loginTb.GetAccount().GetAccess(),
			Profile: &public_protocol_pbdesc.DUserProfile{
				OpenId: loginTb.GetOpenId(),
				UserId: u.GetUserId(),
			},
			ChannelId:   loginTb.GetAccount().GetChannelId(),
			VersionType: loginTb.GetAccount().GetVersionType(),
		}
		userTb.UserData = &private_protocol_pbdesc.UserData{
			UserLevel:       1,
			SessionSequence: 1,
		}
		userTb.DataVersion = UserDataCurrentVersion
	} else {
		udata, err := atfw_utils_fs.ReadAllContent(userTbFilePath)
		if err != nil {
			ctx.LogInfo("load user table from db failed", "zone_id", u.GetZoneId(), "user_id", u.GetUserId(), "file_path", userTbFilePath)
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}

		if err = proto.Unmarshal(udata, userTb); err != nil {
			ctx.LogInfo("unmarshal user table from db failed", "zone_id", u.GetZoneId(), "user_id", u.GetUserId(), "file_path", userTbFilePath, "error", err)
			return cd.CreateRpcResultError(fmt.Errorf("failed to unmarshal user db data: %w", err), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
		}
	}
	userTb.OpenId = loginTb.GetOpenId()
	userTb.ZoneId = u.GetZoneId()
	userTb.UserId = u.GetUserId()

	ctx.LogInfo("load user table from db success", "zone_id", u.GetZoneId(), "user_id", u.GetUserId())

	// Login Table
	u.LoadLoginInfo(u, loginTb, loginTbVersion)

	// Init from DB
	result := u.InitFromDB(u, ctx, userTb)
	if result.IsError() {
		result.LogError(ctx, "init user from db failed", "zone_id", u.GetZoneId(), "user_id", u.GetUserId())
		return result
	}
	result.LogInfo(ctx, "init user from db success", "zone_id", u.GetZoneId(), "user_id", u.GetUserId())

	return cd.CreateRpcResultOk()
}

func (um *UserManager) internalSave(ctx *cd.RpcContext, userImpl UserImpl) cd.RpcResult {
	// TODO: 托管给路由系统
	if userImpl == nil {
		return cd.RpcResult{
			Error:        nil,
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND),
		}
	}

	// TODO: 临时的保存数据
	if _, serr := os.Stat("../data"); serr != nil {
		os.MkdirAll("../data", 0o755)
	}

	var result cd.RpcResult
	if ds, serr := os.Stat("../data"); serr != nil || !ds.IsDir() {
		result = cd.RpcResult{
			Error:        fmt.Errorf("../data is not a directory or can not be created as a directory"),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}

		result.LogError(ctx, "failed to create ../data directory", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	dstTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result = userImpl.DumpToDB(userImpl, ctx, dstTb)

	if result.IsError() {
		result.LogError(ctx, "dump user to db failed", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	userData, err := proto.Marshal(dstTb)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to marshal user db data: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE),
		}
		result.LogError(ctx, "failed to marshal user db data", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	// 路由版本号+1
	routerVersion := userImpl.GetLoginInfo().RouterVersion + 1
	userImpl.GetLoginInfo().RouterVersion = routerVersion
	loginData, err := proto.Marshal(userImpl.GetLoginInfo())
	userImpl.GetLoginInfo().RouterVersion = routerVersion - 1
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to marshal login db data: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE),
		}
		result.LogError(ctx, "failed to marshal login db data", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	uf, err := os.Create(fmt.Sprintf("../data/%d-%d.user.db", userImpl.GetZoneId(), userImpl.GetUserId()))
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create user db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create user db file", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}
	defer uf.Close()

	lf, err := os.Create(fmt.Sprintf("../data/%d-%d.login.db", userImpl.GetZoneId(), userImpl.GetUserId()))
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create login db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create login db file", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}
	defer lf.Close()

	_, err = uf.Write(userData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write user db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write user db file", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	_, err = lf.Write(loginData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write login db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write login db file", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
		return result
	}

	if routerVersion > userImpl.GetLoginInfo().RouterVersion {
		userImpl.GetLoginInfo().RouterVersion = routerVersion

		// 更新Login Table版本号
		userImpl.LoadLoginInfo(userImpl, userImpl.GetLoginInfo(), routerVersion)
	}
	userImpl.OnSaved(userImpl, ctx, routerVersion)

	result.LogInfo(ctx, "save user to db success", "zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId())
	return cd.CreateRpcResultOk()
}

func (um *UserManager) Save(ctx *cd.RpcContext, zoneID uint32, userID uint64, checkUser UserImpl) cd.RpcResult {
	// TODO: 托管给路由系统

	// TODO: 托管给路由对象检查
	userImpl := um.Find(zoneID, userID)
	if userImpl == nil {
		return cd.RpcResult{
			Error:        nil,
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND),
		}
	}

	if checkUser != nil && userImpl != checkUser {
		return cd.RpcResult{
			Error:        nil,
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND),
		}
	}

	return um.internalSave(ctx, userImpl)
}

func (um *UserManager) ScheduleImmediateSave(ctx *cd.RpcContext, zoneID uint32, userID uint64, kickOff bool) cd.RpcResult {
	// TODO: Implement scheduling logic
	userImpl := um.Find(zoneID, userID)
	if userImpl == nil {
		return cd.RpcResult{
			Error:        nil,
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND),
		}
	}

	result := um.Save(ctx, zoneID, userID, userImpl)
	if result.IsError() {
		result.LogError(ctx, "scheduled save user failed", "zone_id", zoneID, "user_id", userID)
		return result
	}

	if kickOff {
		cs_session := userImpl.GetSession()
		if cs_session != nil {
			session, ok := cs_session.(*Session)
			if ok && session != nil {
				GlobalSessionManager.RemoveSession(ctx, session.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_SESSION_KICKOFF_BY_SERVER), "scheduled save and kick off")
			}
		}
	}

	return cd.CreateRpcResultOk()
}

func (um *UserManager) ScheduleQuickSave(ctx *cd.RpcContext, zoneID uint32, userID uint64, kickOff bool) cd.RpcResult {
	// TODO: Implement scheduling logic
	return um.ScheduleImmediateSave(ctx, zoneID, userID, kickOff)
}
