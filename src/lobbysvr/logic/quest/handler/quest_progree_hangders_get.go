package lobbysvr_logic_quest_handler

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
)

// GetInitProgressHandler 获取进度初始化处理函数
// 优先从注册表中查找，如果不存在则返回默认处理函数
func GetProgressKeyIndexHandlerByType(progressType int32) ProgressKeyIndexHandler {
	// 从注册表中查找
	handler := GetProgressKeyIndexHandler(progressType)
	if handler != nil {
		return handler
	}

	// 如果未注册，返回默认的空处理函数
	return defaultInProgressKeyIndexHandler()
}

func defaultInProgressKeyIndexHandler() ProgressKeyIndexHandler {
	return func(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams) UserQuestProgressIndexParams {
		return UserQuestProgressIndexParams{
			ParamsOne: 0,
		}
	}
}
