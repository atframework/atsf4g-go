package atframework_component_db

import (
	"fmt"
	"strconv"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	pu "github.com/atframework/atframe-utils-go/proto_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type RedisSetIndexMessage struct {
	Result     cd.RpcResult
	Message    proto.Message
	CASVersion uint64
}

func HashTableBatchLoad(ctx cd.AwaitableContext, index []string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message,
) (setMessage []*RedisSetIndexMessage, retResult cd.RpcResult) {
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

	pendingActionBatchCount := 20
	runnnigTask := make([]cd.TaskActionImpl, 0, pendingActionBatchCount)
	setMessage = make([]*RedisSetIndexMessage, len(index))
	for i, idx := range index {
		if currentAction.IsExiting() {
			retResult = cd.CreateRpcResultError(fmt.Errorf("task exiting"), public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
			return
		}

		indexName := idx
		subTaskAction := cd.AsyncInvoke(ctx, "HashTableBatchLoadInner",
			currentAction.GetActorExecutor(), func(childCtx cd.AwaitableContext) cd.RpcResult {
				message, casVersion, result := HashTableLoad(childCtx, indexName, tableName, dispatcher, instance, messageCreate)
				setMessage[i] = &RedisSetIndexMessage{
					Result:     result,
					Message:    message,
					CASVersion: casVersion,
				}
				return cd.CreateRpcResultOk()
			})
		if lu.IsNil(subTaskAction) {
			continue
		}
		runnnigTask = append(runnnigTask, subTaskAction)
		if len(runnnigTask) >= pendingActionBatchCount {
			retResult = cd.AwaitTasks(ctx, runnnigTask)
			if retResult.IsError() {
				ctx.LogError("HashTableBatchLoad AwaitTasks failed", "err", retResult.GetErrorString())
				return
			}
			runnnigTask = runnnigTask[:0]
		}
	}
	retResult = cd.AwaitTasks(ctx, runnnigTask)
	if retResult.IsError() {
		ctx.LogError("HashTableBatchLoad AwaitTasks failed", "err", retResult.GetErrorString())
	}
	return
}

func HashTableLoad(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message,
) (table proto.Message, CASVersion uint64, retResult cd.RpcResult) {
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
		Table      proto.Message
		CASVersion uint64
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoad HGetAll Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HGetAll(ctx.GetContext(), index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoad HGetAll Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).LogInfo("HashTableLoad HGetAll Record Not Found", "TableName", tableName, "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
				}
				return resumeError
			}
			pbResult := messageCreate()
			casVersion, err := pu.RedisKVMapToPB(result, pbResult)
			if err != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoad HGetAll Parese Failed", "TableName", tableName, "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoad HGetAll Parse Success", "Seq", awaitOption.Sequence, "TableName", tableName, "Proto", pbResult)
			resumeData.PrivateData = &innerPrivateData{
				Table:      pbResult,
				CASVersion: casVersion,
			}
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
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("load PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	return privateData.Table, privateData.CASVersion, cd.CreateRpcResultOk()
}

func HashTableLoadListAll(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message,
) (indexMessage []pu.RedisListIndexMessage, retResult cd.RpcResult) {
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
		IndexMessage []pu.RedisListIndexMessage
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoadListAll HGetAll Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HGetAll(ctx.GetContext(), index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoadListAll HGetAll Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).LogInfo("HashTableLoadListAll HGetAll Record Not Found", "TableName", tableName, "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
				}
				return resumeError
			}
			indexMessages, err := pu.RedisKLMapToPB(result, messageCreate)
			if err != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoadListAll HGetAll Parese Failed", "TableName", tableName, "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoadListAll HGetAll Parse Success", "Seq", awaitOption.Sequence, "TableName", tableName, "Len", len(indexMessages))
			resumeData.PrivateData = &innerPrivateData{
				IndexMessage: indexMessages,
			}
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
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("load PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	return privateData.IndexMessage, cd.CreateRpcResultOk()
}

func HashTableLoadListIndex(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message, listIndex []uint64, enableCAS bool,
) (indexMessage []pu.RedisListIndexMessage, retResult cd.RpcResult) {
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
		IndexMessage []pu.RedisListIndexMessage
	}

	var indexGetField []string
	var sliceKey []pu.RedisSliceKey
	if enableCAS {
		indexGetField = make([]string, 0, len(listIndex)*2)
		sliceKey = make([]pu.RedisSliceKey, 0, len(listIndex)*2)
	} else {
		indexGetField = make([]string, 0, len(listIndex))
		sliceKey = make([]pu.RedisSliceKey, 0, len(listIndex))
	}
	for _, listIndexId := range listIndex {
		if enableCAS {
			indexGetField = append(indexGetField, fmt.Sprintf("%s%d", pu.RedisListVersionField, listIndexId))
			sliceKey = append(sliceKey, pu.RedisSliceKey{
				Version: true,
				Index:   listIndexId,
			})
		}
		indexGetField = append(indexGetField, fmt.Sprintf("%s%d", pu.RedisListValueField, listIndexId))
		sliceKey = append(sliceKey, pu.RedisSliceKey{
			Version: false,
			Index:   listIndexId,
		})
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoadListIndex HMGet Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "indexGetField", indexGetField)
			result, redisError := instance.HMGet(ctx.GetContext(), index, indexGetField...).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoadListIndex HMGet Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).LogInfo("HashTableLoadListIndex HMGet Record Not Found", "TableName", tableName, "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
				}
				return resumeError
			}
			indexMessages, err := pu.RedisSliceKLMapToPB(sliceKey, result, messageCreate)
			if err != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableLoadListIndex HMGet Parese Failed", "TableName", tableName, "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
			ctx.GetApp().GetLogger(2).LogDebug("HashTableLoadListIndex HMGet Parse Success", "Seq", awaitOption.Sequence, "TableName", tableName, "Len", len(indexMessages))
			resumeData.PrivateData = &innerPrivateData{
				IndexMessage: indexMessages,
			}
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
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("load PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	return privateData.IndexMessage, cd.CreateRpcResultOk()
}

func HashTableBatchPartlyGet(ctx cd.AwaitableContext, index []string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message,
	partlyGetField []string,
) (setMessage []*RedisSetIndexMessage, retResult cd.RpcResult) {
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

	pendingActionBatchCount := 20
	runnnigTask := make([]cd.TaskActionImpl, 0, pendingActionBatchCount)
	setMessage = make([]*RedisSetIndexMessage, len(index))
	for i, idx := range index {
		if currentAction.IsExiting() {
			retResult = cd.CreateRpcResultError(fmt.Errorf("task exiting"), public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
			return
		}

		indexName := idx
		subTaskAction := cd.AsyncInvoke(ctx, "HashTableBatchLoadInner",
			currentAction.GetActorExecutor(), func(childCtx cd.AwaitableContext) cd.RpcResult {
				message, casVersion, result := HashTablePartlyGet(childCtx, indexName, tableName, dispatcher, instance, messageCreate, partlyGetField)
				setMessage[i] = &RedisSetIndexMessage{
					Result:     result,
					Message:    message,
					CASVersion: casVersion,
				}
				return cd.CreateRpcResultOk()
			})
		if lu.IsNil(subTaskAction) {
			continue
		}
		runnnigTask = append(runnnigTask, subTaskAction)
		if len(runnnigTask) >= pendingActionBatchCount {
			retResult = cd.AwaitTasks(ctx, runnnigTask)
			if retResult.IsError() {
				ctx.LogError("HashTableBatchLoad AwaitTasks failed", "err", retResult.GetErrorString())
				return
			}
			runnnigTask = runnnigTask[:0]
		}
	}
	retResult = cd.AwaitTasks(ctx, runnnigTask)
	if retResult.IsError() {
		ctx.LogError("HashTableBatchLoad AwaitTasks failed", "err", retResult.GetErrorString())
	}
	return
}

func HashTablePartlyGet(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, messageCreate func() proto.Message,
	partlyGetField []string,
) (table proto.Message, CASVersion uint64, retResult cd.RpcResult) {
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
		Table      proto.Message
		CASVersion uint64
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTablePartlyGet HMGet Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HMGet(ctx.GetContext(), index, partlyGetField...).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTablePartlyGet HMGet Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).LogInfo("HashTablePartlyGet HMGet Record Not Found", "TableName", tableName, "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
				}
				return resumeError
			}
			pbResult := messageCreate()
			casVersion, recordExist, err := pu.RedisSliceKVMapToPB(partlyGetField, result, pbResult)
			if err != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTablePartlyGet HMGet Parese Failed", "TableName", tableName, "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
					return resumeError
				}
				return err
			} else if !recordExist {
				ctx.GetApp().GetLogger(2).LogInfo("HashTablePartlyGet HMGet Record Not Found", "TableName", tableName, "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load failed resume error", "TableName", tableName,
						"err", resumeError,
					)
				}
				return resumeError
			}
			ctx.GetApp().GetLogger(2).LogDebug("HashTablePartlyGet HMGet Parse Success", "Seq", awaitOption.Sequence, "TableName", tableName, "Proto", pbResult)
			resumeData.PrivateData = &innerPrivateData{
				Table:      pbResult,
				CASVersion: casVersion,
			}
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
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("load PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	return privateData.Table, privateData.CASVersion, cd.CreateRpcResultOk()
}

func HashTableDel(ctx cd.AwaitableContext, index string, tableName string,
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableDel Del Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			_, redisError := instance.Del(ctx.GetContext(), index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableDel Del Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableDel Del Parse Success", "TableName", tableName, "Seq", awaitOption.Sequence)
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

func HashTableDelListIndex(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, listIndex []uint64, enabelCAS bool,
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

	if len(listIndex) == 0 {
		ctx.LogInfo("listIndex is empty")
		return
	}

	var delField []string
	if enabelCAS {
		delField = make([]string, 0, len(listIndex)*2)
	} else {
		delField = make([]string, 0, len(listIndex))
	}

	for _, listIndex := range listIndex {
		if enabelCAS {
			delField = append(delField, fmt.Sprintf("%s%d", pu.RedisListVersionField, listIndex))
		}
		delField = append(delField, fmt.Sprintf("%s%d", pu.RedisListValueField, listIndex))
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableDelListIndex HDel Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index)
			_, redisError := instance.HDel(ctx.GetContext(), index, delField...).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableDelListIndex HDel Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableDelListIndex HDel Parse Success", "TableName", tableName, "Seq", awaitOption.Sequence)
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

func HashTableUpdate(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	table proto.Message) (retResult cd.RpcResult) {
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

	redisData := pu.PBMapToRedisKV(table, nil, false)

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdate HSet Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "data", table)
			redisError := instance.HSet(ctx.GetContext(), index, redisData).Err()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableUpdate HSet Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdate HSet Recv", "TableName", tableName, "Seq", awaitOption.Sequence)
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

func HashTableUpdateCAS(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	table proto.Message, currentCASVersion *uint64, forceUpdate bool,
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

	if forceUpdate {
		*currentCASVersion = 0
	}
	OldCASVersion := *currentCASVersion
	redisData := pu.PBMapToRedisKV(table, &OldCASVersion, forceUpdate)

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateCAS EvalSha Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "currentCASVersion", OldCASVersion, "data", table)
			cmdResult, redisError := instance.EvalSha(ctx.GetContext(), dispatcher.GetCASLuaSHA(), []string{index}, redisData).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			var realVersion uint64
			if redisError == nil {
				switch val := cmdResult.(type) {
				case string:
					realVersion, redisError = strconv.ParseUint(val, 10, 64)
				case []byte:
					realVersion, redisError = strconv.ParseUint(string(val), 10, 64)
				default:
					redisError = fmt.Errorf("unsupport cmd result")
				}
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableUpdateCAS EvalSha Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "currentCASVersion", *currentCASVersion, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateCAS EvalSha Recv", "TableName", tableName, "Seq", awaitOption.Sequence, "realVersion", realVersion)
			resumeData.PrivateData = &realVersion
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("load  failed resume error", "TableName", tableName,
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
		return
	}
	realVersion, ok := resumeData.PrivateData.(*uint64)
	if !ok {
		ctx.LogError("load CASVersion failed not *uint64", "TableName", tableName)
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not CASVersion"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	*currentCASVersion = *realVersion
	if !forceUpdate && OldCASVersion+1 != *currentCASVersion {
		ctx.GetApp().GetLogger(2).LogInfo("HashTableUpdateCAS EvalSha CAS Check Failed", "TableName", tableName, "Seq",
			awaitOption.Sequence, "currentCASVersion+1", OldCASVersion+1, "RealCASVersion", *currentCASVersion)
		retResult = cd.CreateRpcResultError(fmt.Errorf("cas check failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED)
	}
	return
}

func HashTableUpdateList(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	table proto.Message, listIndex uint64) (retResult cd.RpcResult) {
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

	redisData := pu.PBMapToRedisKL(table, nil, false, listIndex)

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateList HSet Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "listIndex", listIndex, "data", table)
			redisError := instance.HSet(ctx.GetContext(), index, redisData).Err()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableUpdateList HSet Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateList HSet Recv", "TableName", tableName, "Seq", awaitOption.Sequence)
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

func HashTableUpdateListCAS(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	table proto.Message, listIndex uint64, currentCASVersion *uint64, forceUpdate bool,
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

	if forceUpdate {
		*currentCASVersion = 0
	}
	OldCASVersion := *currentCASVersion
	redisData := pu.PBMapToRedisKL(table, &OldCASVersion, forceUpdate, listIndex)

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateListCAS EvalSha Send", "TableName", tableName, "Seq", awaitOption.Sequence,
				"index", index, "listIndex", listIndex, "currentCASVersion", OldCASVersion, "data", table)
			cmdResult, redisError := instance.EvalSha(ctx.GetContext(), dispatcher.GetCASLuaSHA(), []string{index}, redisData).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			var realVersion uint64
			if redisError == nil {
				switch val := cmdResult.(type) {
				case string:
					realVersion, redisError = strconv.ParseUint(val, 10, 64)
				case []byte:
					realVersion, redisError = strconv.ParseUint(string(val), 10, 64)
				default:
					redisError = fmt.Errorf("unsupport cmd result")
				}
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableUpdateListCAS EvalSha Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "currentCASVersion", *currentCASVersion, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableUpdateListCAS EvalSha Recv", "TableName", tableName, "Seq", awaitOption.Sequence, "realVersion", realVersion)
			resumeData.PrivateData = &realVersion
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("load  failed resume error", "TableName", tableName,
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
		return
	}
	realVersion, ok := resumeData.PrivateData.(*uint64)
	if !ok {
		ctx.LogError("load CASVersion failed not *uint64", "TableName", tableName)
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not CASVersion"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	*currentCASVersion = *realVersion
	if !forceUpdate && OldCASVersion+1 != *currentCASVersion {
		ctx.GetApp().GetLogger(2).LogInfo("EvalSha HSet CAS Check Failed", "TableName", tableName, "Seq",
			awaitOption.Sequence, "listIndex", listIndex, "currentCASVersion+1", OldCASVersion+1, "RealCASVersion", *currentCASVersion)
		retResult = cd.CreateRpcResultError(fmt.Errorf("cas check failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED)
	}
	return
}

func HashTableAtomicInc(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient, incField string, incValue uint64) (newValue uint64, retResult cd.RpcResult) {
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableAtomicInc HIncrBy Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "field", incField, "incValue", incValue)
			cmdResult, redisError := instance.HIncrBy(ctx.GetContext(), index, incField, int64(incValue)).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError == nil {
				newValue = uint64(cmdResult)
				if cmdResult < 0 {
					redisError = fmt.Errorf("negative result for unsigned inc field")
				}
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HashTableAtomicInc HIncrBy Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
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
			ctx.GetApp().GetLogger(2).LogDebug("HashTableAtomicInc HIncrBy Recv", "TableName", tableName, "Seq", awaitOption.Sequence, "AutoIncId", newValue)
			resumeData.PrivateData = &newValue
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
		return
	}
	value, ok := resumeData.PrivateData.(*uint64)
	if !ok {
		ctx.LogError("load failed not AutoIncId", "TableName", tableName)
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not AutoIncId"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	newValue = *value
	return
}
