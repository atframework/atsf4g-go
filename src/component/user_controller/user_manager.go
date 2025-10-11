package atframework_component_user_controller

import (
	"fmt"
	"os"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"google.golang.org/protobuf/proto"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
)

type CreateUserCallback func(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl

type UserManager struct {
	_ noCopy

	users map[uint32]*map[uint64]UserImpl

	createUserCallback CreateUserCallback
}

func createUserManager() *UserManager {
	ret := &UserManager{
		users: make(map[uint32]*map[uint64]UserImpl),
	}

	ret.createUserCallback = func(ctx *cd.RpcContext, zoneId uint32, userId uint64, openId string) UserImpl {
		ret := CreateUserCache(zoneId, userId, openId)
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

func (um *UserManager) Find(zoneID uint32, userID uint64) UserImpl {
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

// TODO: 临时的数据读取
func UserLoadFromFile(u UserImpl) cd.RpcResult {
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

	// TODO: 临时的保存数据
	if _, serr := os.Stat("../data"); serr != nil {
		os.Mkdir("../data", 0o755)
	}

	var result cd.RpcResult
	if ds, serr := os.Stat("../data"); serr != nil || !ds.IsDir() {
		result = cd.RpcResult{
			Error:        fmt.Errorf("../data is not a directory or can not be created as a directory"),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}

		result.LogError(ctx, "failed to create ../data directory", "zone_id", zoneID, "user_id", userID)
		return result
	}

	dstTb := &private_protocol_pbdesc.DatabaseTableUser{}
	result = userImpl.DumpToDB(userImpl, ctx, dstTb)

	if result.IsError() {
		result.LogError(ctx, "dump user to db failed", "zone_id", zoneID, "user_id", userID)
		return result
	}

	userData, err := proto.Marshal(dstTb)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to marshal user db data: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE),
		}
		result.LogError(ctx, "failed to marshal user db data", "zone_id", zoneID, "user_id", userID)
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
		result.LogError(ctx, "failed to marshal login db data", "zone_id", zoneID, "user_id", userID)
		return result
	}

	uf, err := os.Create(fmt.Sprintf("../data/%d-%d.user.db", zoneID, userID))
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create user db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create user db file", "zone_id", zoneID, "user_id", userID)
		return result
	}
	defer uf.Close()

	lf, err := os.Create(fmt.Sprintf("../data/%d-%d.login.db", zoneID, userID))
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create login db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create login db file", "zone_id", zoneID, "user_id", userID)
		return result
	}
	defer lf.Close()

	_, err = uf.Write(userData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write user db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write user db file", "zone_id", zoneID, "user_id", userID)
		return result
	}

	_, err = lf.Write(loginData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write login db file: %w", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write login db file", "zone_id", zoneID, "user_id", userID)
		return result
	}

	if routerVersion > userImpl.GetLoginInfo().RouterVersion {
		userImpl.GetLoginInfo().RouterVersion = routerVersion
	}
	userImpl.OnSaved(userImpl, ctx, int64(routerVersion))

	return cd.CreateRpcResultOk()
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
				GlobalSessionManager.RemoveSession(session.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_SESSION_KICKOFF_BY_SERVER), "scheduled save and kick off")
			}
		}
	}

	return cd.CreateRpcResultOk()
}

func (um *UserManager) ScheduleQuickSave(ctx *cd.RpcContext, zoneID uint32, userID uint64, kickOff bool) cd.RpcResult {
	// TODO: Implement scheduling logic
	return um.ScheduleImmediateSave(ctx, zoneID, userID, kickOff)
}
