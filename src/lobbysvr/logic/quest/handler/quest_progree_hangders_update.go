package lobbysvr_logic_quest_handler

import (
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

// GetUpdateProgressHandler 获取进度更新处理函数
// 优先从注册表中查找，如果不存在则返回默认处理函数
func GetUpdateProgressHandlerByType(progressType int32) UpdateProgressHandler {
	// 从注册表中查找处理器
	handlers := progressHandlerRegistry[progressType]
	if handlers != nil && handlers.UpdateHandler != nil {
		// 将 UpdateProgressHandler（带 countType）包装为 UpdateProgressConditionFunc
		return handlers.UpdateHandler
	}

	// 如果未注册，返回默认的空处理函数
	return defaultUpdateProgressHandler()
}

func defaultUpdateProgressHandler() UpdateProgressHandler {
	return func(_ cd.RpcContext,
		_ *private_protocol_pbdesc.QuestTriggerParams,
		_ *public_protocol_config.Readonly_DQuestConditionProgress,
		_ *public_protocol_pbdesc.DUserQuestProgressData) cd.RpcResult {
		return cd.CreateRpcResultOk()
	}
}
