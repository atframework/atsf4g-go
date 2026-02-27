package atframework_component_db

import (
	"fmt"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Expire 选项枚举
// ============================================================================

// ExpireConditionOption EXPIRE/EXPIREAT 命令的条件选项
type ExpireConditionOption int32

const (
	// ExpireConditionNone 不指定条件，始终设置过期时间
	ExpireConditionNone ExpireConditionOption = 0
	// ExpireConditionNX 仅当键当前没有设置过期时间时才设置
	ExpireConditionNX ExpireConditionOption = 1
	// ExpireConditionXX 仅当键当前已有过期时间时才设置
	ExpireConditionXX ExpireConditionOption = 2
	// ExpireConditionGT 仅当新过期时间大于当前过期时间时才设置
	ExpireConditionGT ExpireConditionOption = 3
	// ExpireConditionLT 仅当新过期时间小于当前过期时间时才设置
	ExpireConditionLT ExpireConditionOption = 4
)

func TableDel(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
) (retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("TableDel Del Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			_, redisError := instance.Del(ctx.GetContext(), index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("TableDel Del Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
					return resumeError
				}
				return redisError
			}
			ctx.GetApp().GetLogger(2).LogDebug("TableDel Del Parse Success", "TableName", tableName, "Seq", awaitOption.Sequence)
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("load failed resume error", "TableName", tableName,
					"err", resumeError,
				)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
	}
	return
}

// TableExpire 对指定 key 设置过期时间（秒级 TTL）
// condition: 条件选项 NX/XX/GT/LT
func TableExpire(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	expiration time.Duration,
	condition ExpireConditionOption,
) (success bool, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		Success bool
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("TableExpire Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "expiration", expiration, "condition", condition)

			var cmdResult bool
			var redisError error
			switch condition {
			case ExpireConditionNX:
				cmdResult, redisError = instance.ExpireNX(ctx.GetContext(), index, expiration).Result()
			case ExpireConditionXX:
				cmdResult, redisError = instance.ExpireXX(ctx.GetContext(), index, expiration).Result()
			case ExpireConditionGT:
				cmdResult, redisError = instance.ExpireGT(ctx.GetContext(), index, expiration).Result()
			case ExpireConditionLT:
				cmdResult, redisError = instance.ExpireLT(ctx.GetContext(), index, expiration).Result()
			default:
				cmdResult, redisError = instance.Expire(ctx.GetContext(), index, expiration).Result()
			}

			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("TableExpire Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("expire failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			ctx.GetApp().GetLogger(2).LogDebug("TableExpire Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "success", cmdResult)
			resumeData.PrivateData = &innerPrivateData{Success: cmdResult}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("expire failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("expire PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	success = privateData.Success
	return
}

// TableExpireAt 对指定 key 设置过期时间点（绝对时间戳）
func TableExpireAt(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	expireAt time.Time,
) (success bool, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		Success bool
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("TableExpireAt Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "expireAt", expireAt)

			cmdResult, redisError := instance.ExpireAt(ctx.GetContext(), index, expireAt).Result()

			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("TableExpireAt Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("expireat failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			ctx.GetApp().GetLogger(2).LogDebug("TableExpireAt Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "success", cmdResult)
			resumeData.PrivateData = &innerPrivateData{Success: cmdResult}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("expireat failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("expireat PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	success = privateData.Success
	return
}
