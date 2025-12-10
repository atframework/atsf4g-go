## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    load_index_key = index_meta["load_index_key"]
    key_fields = index_meta["key_fields"]
    cas_enabled = index_meta["cas_enabled"]
%>
func ${message_name}LoadWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
% if cas_enabled:
) (table *private_protocol_pbdesc.${message_name}, CASVersion uint64, retResult cd.RpcResult) {
% else:
) (table *private_protocol_pbdesc.${message_name}, retResult cd.RpcResult) {
% endif
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${load_index_key}
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
		Table      *private_protocol_pbdesc.${message_name}
% if cas_enabled:
		CASVersion uint64
% endif
    }

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HGetAll ${message_name} Send", "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HGetAll(ctx.GetContext(), index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HGetAll ${message_name} Recv Error", "Seq", awaitOption.Sequence, "redisError", redisError)
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
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).LogInfo("HGetAll ${message_name} Record Not Found", "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
				}
				return resumeError
			}
			pbResult := new(private_protocol_pbdesc.${message_name})
% if cas_enabled:
		    casVersion, err := pu.RedisMapToPB(result, pbResult)
% else:
		    _, err := pu.RedisMapToPB(result, pbResult)
% endif
    		if err != nil {
				ctx.GetApp().GetLogger(2).LogError("HGetAll ${message_name} Parese Failed", "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
% if cas_enabled:
    		ctx.GetApp().GetLogger(2).LogDebug("HGetAll ${message_name} Parse Success", "Seq", awaitOption.Sequence, "CASVersion", casVersion, "Proto", pbResult)
% else:
    		ctx.GetApp().GetLogger(2).LogDebug("HGetAll ${message_name} Parse Success", "Seq", awaitOption.Sequence, "Proto", pbResult)
% endif
    		resumeData.PrivateData = &innerPrivateData{
				Table: pbResult,
% if cas_enabled:
			    CASVersion: casVersion,
% endif
            }
			resumeData.Result = cd.CreateRpcResultOk()
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
    privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("load ${message_name} failed not ${message_name}")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not ${message_name}"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	table = privateData.Table
% if cas_enabled:
	CASVersion = privateData.CASVersion
% endif

	ctx.LogInfo("load ${message_name} table with key from db success",
% if cas_enabled:
		"CASVersion", CASVersion,
% endif
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
