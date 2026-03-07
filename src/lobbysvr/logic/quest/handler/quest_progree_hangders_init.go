package lobbysvr_logic_quest_handler

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// GetInitProgressHandler 获取进度初始化处理函数
// 优先从注册表中查找，如果不存在则返回默认处理函数
func GetInitProgressHandlerByType(progressType int32) InitProgressHandler {
	// 从注册表中查找
	handler := GetInitProgressHandler(progressType)
	if handler != nil {
		return handler
	}

	// 如果未注册，返回默认的空处理函数
	return defaultInitProgressHandler()
}

func defaultInitProgressHandler() InitProgressHandler {
	return func(_ cd.RpcContext,
		_ *public_protocol_config.Readonly_DQuestConditionProgress,
		questData *public_protocol_pbdesc.DUserQuestProgressData,
		_ *data.User) cd.RpcResult {
		if questData != nil {
			questData.Value = 0
		}
		return cd.CreateRpcResultOk()
	}
}
