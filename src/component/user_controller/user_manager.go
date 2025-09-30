package atframework_component_user_controller

type CreateUserCallback func(zoneId uint32, userId uint64, openId string) UserImpl

type UserManager struct {
	_ noCopy

	users map[uint32]*map[uint64]UserImpl

	createUserCallback CreateUserCallback
}

func createUserManager() *UserManager {
	ret := &UserManager{
		users: make(map[uint32]*map[uint64]UserImpl),
	}

	ret.createUserCallback = func(zoneId uint32, userId uint64, openId string) UserImpl {
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
