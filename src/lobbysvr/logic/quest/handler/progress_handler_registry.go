package lobbysvr_logic_quest_handler

import (
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_config "github.com/atframework/atsf4g-go/component/protocol/public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
)

// InitProgressHandler 进度初始化处理函数
type InitProgressHandler func(ctx cd.RpcContext,
	progressCfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData,
	user *data.User) cd.RpcResult

// UpdateProgressHandler 进度更新处理函数
type UpdateProgressHandler func(ctx cd.RpcContext,
	params *private_protocol_pbdesc.QuestTriggerParams,
	cfg *public_protocol_config.Readonly_DQuestConditionProgress,
	questData *public_protocol_pbdesc.DUserQuestProgressData) cd.RpcResult

// UserQuestProgressIndexParams 进度索引参数
type UserQuestProgressIndexParams struct {
	ParamsOne int32
}

// ProgressKeyIndexHandler 进度关键字索引处理函数
type ProgressKeyIndexHandler func(_ cd.RpcContext, progressType int32, params *private_protocol_pbdesc.QuestTriggerParams) UserQuestProgressIndexParams

// ProgressHandlers 某个进度类型的处理函数对
type ProgressHandlers struct {
	InitHandler             InitProgressHandler
	UpdateHandler           UpdateProgressHandler
	progressKeyIndexHandler ProgressKeyIndexHandler
}

// progressHandlerRegistry 进度处理器注册表，key为进度类型
var progressHandlerRegistry = make(map[int32]*ProgressHandlers)

// RegisterProgressHandler 注册指定进度类型的初始化和更新处理函数
func RegisterProgressHandler(progressType int32, initHandler InitProgressHandler, updateHandler UpdateProgressHandler, progressKeyIndexHandler ProgressKeyIndexHandler) {
	if progressType <= 0 {
		return
	}

	if updateHandler == nil {
		return
	}

	handlers := &ProgressHandlers{
		InitHandler:             initHandler,
		UpdateHandler:           updateHandler,
		progressKeyIndexHandler: progressKeyIndexHandler,
	}

	progressHandlerRegistry[progressType] = handlers
}

// GetInitProgressHandler 获取指定进度类型的初始化处理函数
func GetInitProgressHandler(progressType int32) InitProgressHandler {
	handlers, ok := progressHandlerRegistry[progressType]
	if !ok || handlers == nil {
		return nil
	}
	return handlers.InitHandler
}

func GetUpdateProgressHandler(progressType int32) UpdateProgressHandler {
	handlers, ok := progressHandlerRegistry[progressType]
	if !ok || handlers == nil {
		return nil
	}
	return handlers.UpdateHandler
}

func GetProgressKeyIndexHandler(progressType int32) ProgressKeyIndexHandler {
	handlers, ok := progressHandlerRegistry[progressType]
	if !ok || handlers == nil {
		return nil
	}
	return handlers.progressKeyIndexHandler
}
