package lobbysvr_data

type UserModuleManagerImpl interface {
	GetOwner() *User

	RefreshLimit()
	RefreshLimitSecond()
	RefreshLimitMinute()

	InitFromDB()
	DumpToDB()

	SyncDirtyCache()
	CleanupDirtyCache()
}

type UserModuleManagerBase struct {
	owner *User
}
