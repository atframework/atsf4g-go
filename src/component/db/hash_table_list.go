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
