package lobbysvr_logic_timer

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

type UserTimerManager interface {
	data.UserModuleManagerImpl

	SetTimer(ctx cd.RpcContext, triggerTimestamp int64)
	Tick(ctx cd.RpcContext)
}
