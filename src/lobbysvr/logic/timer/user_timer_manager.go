package lobbysvr_logic_timer

import (
	cd "github.com/atframework/atsf4g-go/src/component/dispatcher"
	data "github.com/atframework/atsf4g-go/src/lobbysvr/data"
)

type UserTimerManager interface {
	data.UserModuleManagerImpl

	SetTimer(ctx cd.RpcContext, triggerTimestamp int64)
	Tick(ctx cd.RpcContext)
}
