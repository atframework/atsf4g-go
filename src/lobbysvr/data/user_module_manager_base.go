package lobbysvr_data

import (
	"reflect"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type UserModuleManagerImpl interface {
	GetOwner() *User

	// 每次执行任务前刷新
	RefreshLimit(*cd.RpcContext)
	// 每秒刷新
	RefreshLimitSecond(*cd.RpcContext)
	// 每分钟刷新
	RefreshLimitMinute(*cd.RpcContext)

	InitFromDB(*cd.RpcContext, *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult
	DumpToDB(*cd.RpcContext, *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult

	CreateInit(ctx *cd.RpcContext, versionType uint32)
	LoginInit(*cd.RpcContext)

	OnLogin(*cd.RpcContext)
	OnLogout(*cd.RpcContext)
	OnSaved(*cd.RpcContext, uint64)
	OnUpdateSession(ctx *cd.RpcContext, from *uc.Session, to *uc.Session)
}

var userModuleManagerCreators = make(map[reflect.Type]struct {
	typeInst reflect.Type
	fn       func(*User) UserModuleManagerImpl
})

func RegisterUserModuleManagerCreator[ManagerType any](creator func(*User) UserModuleManagerImpl) {
	if creator == nil {
		panic("nil user module manager creator")
	}

	typeInst := reflect.TypeOf((*ManagerType)(nil)).Elem()
	if _, exists := userModuleManagerCreators[typeInst]; exists {
		panic("duplicate user module manager creator for type: " + typeInst.String())
	}

	userModuleManagerCreators[typeInst] = struct {
		typeInst reflect.Type
		fn       func(*User) UserModuleManagerImpl
	}{
		typeInst: typeInst,
		fn:       creator,
	}
}

type UserModuleManagerBase struct {
	_     noCopy
	owner *User
}

func CreateUserModuleManagerBase(owner *User) *UserModuleManagerBase {
	return &UserModuleManagerBase{
		owner: owner,
	}
}

func (m *UserModuleManagerBase) GetOwner() *User {
	return m.owner
}

func (m *UserModuleManagerBase) RefreshLimit(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) RefreshLimitSecond(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) RefreshLimitMinute(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) InitFromDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserModuleManagerBase) DumpToDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.RpcResult{
		Error:        nil,
		ResponseCode: 0,
	}
}

func (m *UserModuleManagerBase) CreateInit(_ctx *cd.RpcContext, _versionType uint32) {
}

func (m *UserModuleManagerBase) LoginInit(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) OnLogin(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) OnLogout(_ctx *cd.RpcContext) {
}

func (m *UserModuleManagerBase) OnSaved(_ctx *cd.RpcContext, _version uint64) {
}

func (m *UserModuleManagerBase) OnUpdateSession(_ctx *cd.RpcContext, _from *uc.Session, _to *uc.Session) {
}

func (m *UserModuleManagerBase) SyncClientDirtyCache() {
}

func (m *UserModuleManagerBase) CleanupClientDirtyCache() {
}
