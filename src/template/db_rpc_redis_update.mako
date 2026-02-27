## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_type = index_meta["index_type"]
    struct_redis_key_call = index_meta["struct_redis_key_call"]
    key_fields = index_meta["key_fields"]
    cas_enabled = index_meta["cas_enabled"]
    max_list_length = index_meta["max_list_length"]
%>
% if index_type == "kv":
% if cas_enabled:
func ${message_name}Add${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
) (retResult cd.RpcResult, CASVersion uint64) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${struct_redis_key_call}
	CASVersion = 0
	retResult = HashTableUpdateCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, &CASVersion, false)
	if retResult.IsError() {
		if retResult.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
			retResult = cd.CreateRpcResultError(fmt.Errorf("record exist"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_EXIST)
		}
		return
	}

	ctx.LogDebug("update ${message_name} table with key from db success",
		"RealCASVersion", CASVersion,
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

func ${message_name}Replace${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
	currentCASVersion *uint64,
	forceUpdate bool,
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${struct_redis_key_call}
	if currentCASVersion == nil {
		ctx.LogError("currentCASVersion nil")
		retResult = cd.CreateRpcResultError(fmt.Errorf("currentCASVersion nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	retResult = HashTableUpdateCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, currentCASVersion, forceUpdate)
	if retResult.IsError() {
		return
	}

	ctx.LogDebug("update ${message_name} table with key from db success",
		"RealCASVersion", *currentCASVersion,
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% else:
func ${message_name}Update${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${struct_redis_key_call}
	retResult = HashTableUpdate(ctx, index, "${message_name}",
		dispatcher, instance, table)
	if retResult.IsError() {
		return
	}

	ctx.LogDebug("update ${message_name} table with key from db success",
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% endif
% else:
func ${message_name}Add${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
) (retResult cd.RpcResult, newListIndex uint64) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${struct_redis_key_call}
	retResult, newListIndex = HashTableAddList(ctx, index, "${message_name}",
		dispatcher, instance, table, ${max_list_length})
	if retResult.IsError() {
		if retResult.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
			retResult = cd.CreateRpcResultError(fmt.Errorf("record exist"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_EXIST)
		}
		return
	}

	ctx.LogDebug("add ${message_name} table with key from db success",
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

func ${message_name}Update${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
	listIndex uint64,
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${struct_redis_key_call}
	retResult = HashTableUpdateList(ctx, index, "${message_name}",
		dispatcher, instance, table, listIndex)
	if retResult.IsError() {
		return
	}

	ctx.LogDebug("update ${message_name} table with key from db success",
		"listIndex", listIndex,
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% endif