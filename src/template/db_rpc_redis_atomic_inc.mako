## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta,inc_field" />
<%
    index_key_name = index_meta["index_key_name"]
    load_index_key = index_meta["load_index_key"]
    key_fields = index_meta["key_fields"]
    go_type = inc_field["go_type"]
    is_unsigned = go_type.startswith("uint")
%>
func ${message_name}AtomicInc${index_key_name}${inc_field["ident"]}(
    ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
    incValue ${inc_field["go_type"]},
) (newValue ${inc_field["go_type"]}, retResult cd.RpcResult) {
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

    pushActionFunc := func() cd.RpcResult {
        err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("HIncrBy ${message_name} Send", "Seq", awaitOption.Sequence, "index", index, "field", "${inc_field["raw_name"]}", "incValue", incValue)
			cmdResult, redisError := instance.HIncrBy(ctx.GetContext(), index, "${inc_field["raw_name"]}", int64(incValue)).Result()
            resumeData := &cd.DispatcherResumeData{
                Message: &cd.DispatcherRawMessage{
                    Type: awaitOption.Type,
                },
                Sequence:    awaitOption.Sequence,
                PrivateData: nil,
            }
            if redisError == nil {
				newValue = ${go_type}(cmdResult)
				% if is_unsigned:
				if cmdResult < 0 {
					redisError = fmt.Errorf("negative result for unsigned inc field")
				}
				% endif
            }
            if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("HIncrBy ${message_name} Recv Error", "Seq", awaitOption.Sequence, "redisError", redisError)
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
            ctx.GetApp().GetLogger(2).LogDebug("HIncrBy ${message_name} Recv", "Seq", awaitOption.Sequence, "${inc_field["ident"]}", newValue)
            resumeData.PrivateData = &newValue
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
    resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
    if retResult.IsError() {
        return
    }
    if resumeData.Result.IsError() {
        retResult = resumeData.Result
        return
    }
    value, ok := resumeData.PrivateData.(*${inc_field["go_type"]})
    if !ok {
        ctx.LogError("load ${message_name} failed not ${inc_field["ident"]}")
        retResult = cd.CreateRpcResultError(fmt.Errorf("private data not ${inc_field["ident"]}"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
        return
    }
    newValue = *value

    ctx.LogInfo("atomic inc ${message_name} table with key from db success",
        "${inc_field["ident"]}", newValue,
% for field in key_fields:
        "${field["ident"]}", ${field["ident"]},
% endfor
    )
    retResult = cd.CreateRpcResultOk()
    return
}
