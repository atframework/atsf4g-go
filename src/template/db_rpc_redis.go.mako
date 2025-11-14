## -*- coding: utf-8 -*-
<%!
import time
import sys
%><%page args="message_name,extension,message,index_type_enum,split_type_enum" />
%	for field in message.fields:
%		if not field.is_db_vaild_type():
// ${message_name} filed: {${field.get_name()}} not db vaild type
<% return %>
%		endif
% 	endfor
%   for index in extension.index:
<%
	db_fmt_key = index.name
	index_key_name = ""
	load_key_fmt_args = ""
	update_key_fmt_args = ""
	for key in index.key_fields:
		index_key_name = index_key_name + message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
		db_fmt_key = db_fmt_key + "." + message.fields_by_name[key].get_go_fmt_type()
		load_key_fmt_args = load_key_fmt_args + message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL) + ", "
		update_key_fmt_args = update_key_fmt_args + "table.Get" + message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL) + "(), "

	prefix_fmt_key = "%s"
	prefix_fmt_value = "dispatcher.GetRecordPrefix()"

	if index.split_type == split_type_enum.values_by_name["EN_ATFRAMEWORK_DB_TABLE_SPLIT_TYPE_WORLD"].descriptor.number:
		prefix_fmt_key = prefix_fmt_key + "-%d-"
		prefix_fmt_value = prefix_fmt_value + ", ctx.GetApp().GetWorldId()"
	if index.split_type == split_type_enum.values_by_name["EN_ATFRAMEWORK_DB_TABLE_SPLIT_TYPE_WORLD_ZONE"].descriptor.number:
		prefix_fmt_key = prefix_fmt_key + "-%d-%d-"
		prefix_fmt_value = prefix_fmt_value + ", ctx.GetApp().GetWorldId(), ctx.GetApp().GetZoneId()"


	load_index_key = "index := fmt.Sprintf(\"" + prefix_fmt_key + db_fmt_key + "\", " + prefix_fmt_value + ",\n	   " + load_key_fmt_args + "\n    )"
	update_index_key = "index := fmt.Sprintf(\"" + prefix_fmt_key + db_fmt_key + "\", " + prefix_fmt_value + ",\n	   " + update_key_fmt_args + "\n    )"
%>
func ${message_name}LoadWith${index_key_name}(
	ctx *cd.RpcContext,
%           for key in index.key_fields:
	${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} ${message.fields_by_name[key].get_go_type()},
%           endfor
) (*private_protocol_pbdesc.${message_name}, cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	${load_index_key}
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		return nil, cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().Context) {
		ctx.LogError("not found context")
		return nil, cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).Debug("HGetAll ${message_name} Send: \n", "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HGetAll(ctx.Context, index).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).Error("HGetAll ${message_name} Recv Error: \n", "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return redisError
            }
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).Info("HGetAll ${message_name} Record Not Found: \n", "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
				}
				return resumeError
			}
			pbResult := new(private_protocol_pbdesc.${message_name})
			err := pu.RedisMapToPB(result, pbResult)
			if err != nil {
				ctx.GetApp().GetLogger(2).Error("HGetAll ${message_name} Parese Failed: \n", "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
			ctx.GetApp().GetLogger(2).Debug("HGetAll ${message_name} Parse Success: \n", "Seq", awaitOption.Sequence, "Proto", pbResult)
			resumeData.PrivateData = pbResult
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
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
	resumeData, result := cd.YieldTaskAction(ctx.GetApp(), currentAction, awaitOption, pushActionFunc)
	if result.IsError() {
		return nil, *result
	}
	if resumeData.Result.IsError() {
		return nil, resumeData.Result
	}
    ${message_name}Table, ok := resumeData.PrivateData.(*private_protocol_pbdesc.${message_name})
	if !ok {
		ctx.LogError("load ${message_name} failed not ${message_name}")
		return nil, cd.CreateRpcResultError(fmt.Errorf("private data not ${message_name}"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	ctx.LogInfo("load ${message_name} table with key from db success",
%           for key in index.key_fields:
		"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", ${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%           endfor
	)
	return ${message_name}Table, cd.CreateRpcResultOk()
}

func ${message_name}Update${index_key_name}(
	ctx *cd.RpcContext,
    table *private_protocol_pbdesc.${message_name},
) cd.RpcResult {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	${update_index_key}
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		return cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().Context) {
		ctx.LogError("not found context")
		return cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

    redisData := pu.PBMapToRedis(table)

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).Debug("HSet ${message_name} Send: \n", "Seq", awaitOption.Sequence, "index", index, "data", table)
			redisError := instance.HSet(ctx.Context, index, redisData).Err()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).Error("HSet ${message_name} Recv Error: \n", "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return redisError
            }
			ctx.GetApp().GetLogger(2).Debug("HSet ${message_name} Recv: \n", "Seq", awaitOption.Sequence)
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
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
	resumeData, result := cd.YieldTaskAction(ctx.GetApp(), currentAction, awaitOption, pushActionFunc)
	if result.IsError() {
		return *result
	}
	if resumeData.Result.IsError() {
		return resumeData.Result
	}

	ctx.LogInfo("update ${message_name} table with key from db success",
%           for key in index.key_fields:
		"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
	)
	return cd.CreateRpcResultOk()
}
%   	for partly_get in index.partly_get:
<%
    	partly_field_name = ""
    	for field in partly_get.fields:
			check = message.fields_by_name[field]
			partly_field_name = partly_field_name + message.get_identify_name(field, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
func ${message_name}LoadWith${index_key_name}PartlyField${partly_field_name}(
	ctx *cd.RpcContext,
%           for key in index.key_fields:
	${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} ${message.fields_by_name[key].get_go_type()},
%           endfor
) (*private_protocol_pbdesc.${message_name}, cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	${update_index_key}
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		return nil, cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().Context) {
		ctx.LogError("not found context")
		return nil, cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			redisField := []string{
%           for key in index.key_fields:
				"${key}",
%           endfor
			}
			ctx.GetApp().GetLogger(2).Debug("HMGet ${message_name} Send: \n", "Seq", awaitOption.Sequence, "index", index)
			result, redisError := instance.HMGet(ctx.Context, index, redisField...).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).Error("HMGet ${message_name} Recv Raw: \n", "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return redisError
            }
			if len(result) == 0 {
				ctx.GetApp().GetLogger(2).Info("HMGet ${message_name} Record Not Found: \n", "Seq", awaitOption.Sequence)
				resumeData.Result = cd.CreateRpcResultError(fmt.Errorf("record not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_NOT_FOUND)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
				}
				return resumeError
			}
			pbResult := new(private_protocol_pbdesc.${message_name})
			err := pu.RedisSliceMapToPB(redisField, result, pbResult)
			if err != nil {
				ctx.GetApp().GetLogger(2).Error("HMGet ${message_name} Parese Failed: \n", "Seq", awaitOption.Sequence, "Raw", result)
				resumeData.Result = cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
				resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("load ${message_name} failed resume error",
						"err", resumeError,
					)
					return resumeError
				}
				return err
			}
			ctx.GetApp().GetLogger(2).Debug("HMGet ${message_name} Parse Success: \n", "Seq", awaitOption.Sequence, "Proto", pbResult)
			resumeData.PrivateData = pbResult
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(app_action.App, currentAction, resumeData)
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
	resumeData, result := cd.YieldTaskAction(ctx.GetApp(), currentAction, awaitOption, pushActionFunc)
	if result.IsError() {
		return nil, *result
	}
	if resumeData.Result.IsError() {
		return nil, resumeData.Result
	}
    ${message_name}Table, ok := resumeData.PrivateData.(*private_protocol_pbdesc.${message_name})
	if !ok {
		ctx.LogError("load ${message_name} failed not ${message_name}")
		return nil, cd.CreateRpcResultError(fmt.Errorf("private data not ${message_name}"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	ctx.LogInfo("load ${message_name} table with key from db success",
%           for key in index.key_fields:
		"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", ${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%           endfor
	)
	return ${message_name}Table, cd.CreateRpcResultOk()
}
%   	endfor
%   endfor