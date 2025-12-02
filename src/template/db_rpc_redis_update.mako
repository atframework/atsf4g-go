## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    update_index_key = index_meta["update_index_key"]
    key_fields = index_meta["key_fields"]
    cas_enabled = index_meta["cas_enabled"]
%>
func ${message_name}Update${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
% if cas_enabled:
	currentCASVersion *uint64,
	forceUpdate bool,
% endif
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${update_index_key}
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

% if cas_enabled:
	if currentCASVersion == nil {
		ctx.LogError("currentCASVersion nil")
		retResult = cd.CreateRpcResultError(fmt.Errorf("currentCASVersion nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	if forceUpdate {
		*currentCASVersion = 0
	}
	OldCASVersion := *currentCASVersion
    redisData := pu.PBMapToRedis(table, &OldCASVersion, forceUpdate)
% else:
    redisData := pu.PBMapToRedis(table, nil, false)
% endif

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
% if cas_enabled:
			ctx.GetApp().GetLogger(2).Debug("EvalSha HSet ${message_name} Send: \n", "Seq", awaitOption.Sequence, "index", index, "currentCASVersion", OldCASVersion, "data", table)
			cmdResult, redisError := instance.EvalSha(ctx.GetContext(), dispatcher.GetCASLuaSHA(), []string{index}, redisData).Result()
% else:
			ctx.GetApp().GetLogger(2).Debug("HSet ${message_name} Send: \n", "Seq", awaitOption.Sequence, "index", index, "data", table)
			redisError := instance.HSet(ctx.GetContext(), index, redisData).Err()
% endif
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
% if cas_enabled:
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
% endif
			if redisError != nil {
% if cas_enabled:
				ctx.GetApp().GetLogger(2).Error("EvalSha HSet ${message_name} Recv Error: \n", "Seq", awaitOption.Sequence, "currentCASVersion", *currentCASVersion, "redisError", redisError)
% else:
				ctx.GetApp().GetLogger(2).Error("HSet ${message_name} Recv Error: \n", "Seq", awaitOption.Sequence, "redisError", redisError)
% endif
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return redisError
            }
% if cas_enabled:
			ctx.GetApp().GetLogger(2).Debug("EvalSha HSet ${message_name} Recv: \n", "Seq", awaitOption.Sequence, "realVersion", realVersion)
			resumeData.PrivateData = &realVersion
			resumeData.Result = cd.CreateRpcResultOk()
% else:
			ctx.GetApp().GetLogger(2).Debug("HSet ${message_name} Recv: \n", "Seq", awaitOption.Sequence)
			resumeData.Result = cd.CreateRpcResultOk()
% endif
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("load ${message_name} failed resume error",
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
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
% if cas_enabled:
    realVersion, ok := resumeData.PrivateData.(*uint64)
	if !ok {
		ctx.LogError("load ${message_name} CASVersion failed not *uint64")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not CASVersion"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	*currentCASVersion = *realVersion
	if OldCASVersion != 0 && OldCASVersion+1 != *currentCASVersion {
		ctx.GetApp().GetLogger(2).Info("EvalSha HSet ${message_name} CAS Check Failed: \n", "Seq",
			awaitOption.Sequence, "currentCASVersion+1", OldCASVersion+1, "RealCASVersion", *currentCASVersion)
		retResult = cd.CreateRpcResultError(fmt.Errorf("cas check failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED)
		return
	}
% endif
	ctx.LogInfo("update ${message_name} table with key from db success",
% if cas_enabled:
		"RealCASVersion", *currentCASVersion,
% endif
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
