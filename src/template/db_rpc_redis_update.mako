## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_type_kv = index_meta["index_type_kv"]
    update_index_key = index_meta["update_index_key"]
    key_fields = index_meta["key_fields"]
    cas_enabled = index_meta["cas_enabled"]
%>
% if cas_enabled:
func ${message_name}Add${index_meta["index_key_name"]}(
	ctx cd.AwaitableContext,
    table *private_protocol_pbdesc.${message_name},
% if not index_type_kv:
	listIndex uint64,
% endif
) (retResult cd.RpcResult, CASVersion uint64) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${update_index_key}
	CASVersion = 0
% if index_type_kv:
	retResult = HashTableUpdateCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, &CASVersion, false)
% else:
	retResult = HashTableUpdateListCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, listIndex, &CASVersion, false)
% endif
	if retResult.IsError() {
		if retResult.GetResponseCode() == int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_CAS_CHECK_FAILED) {
			retResult = cd.CreateRpcResultError(fmt.Errorf("record exist"), public_protocol_pbdesc.EnErrorCode_EN_ERR_DB_RECORD_EXIST)
		}
		return
	}

	ctx.LogInfo("update ${message_name} table with key from db success",
		"RealCASVersion", CASVersion,
% if not index_type_kv:
		"listIndex", listIndex,
% endif
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
% if not index_type_kv:
	listIndex uint64,
% endif
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
	${update_index_key}
	if currentCASVersion == nil {
		ctx.LogError("currentCASVersion nil")
		retResult = cd.CreateRpcResultError(fmt.Errorf("currentCASVersion nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
% if index_type_kv:
	retResult = HashTableUpdateCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, currentCASVersion, forceUpdate)
% else:
	retResult = HashTableUpdateListCAS(ctx, index, "${message_name}",
		dispatcher, instance, table, listIndex, currentCASVersion, forceUpdate)
% endif
	if retResult.IsError() {
		return
	}

	ctx.LogInfo("update ${message_name} table with key from db success",
		"RealCASVersion", *currentCASVersion,
% if not index_type_kv:
		"listIndex", listIndex,
% endif
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
% if not index_type_kv:
	listIndex uint64,
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
% if index_type_kv:
	retResult = HashTableUpdate(ctx, index, "${message_name}",
		dispatcher, instance, table)
% else:
	retResult = HashTableUpdateList(ctx, index, "${message_name}",
		dispatcher, instance, table, listIndex)
% endif
	if retResult.IsError() {
		return
	}

	ctx.LogInfo("update ${message_name} table with key from db success",
% if not index_type_kv:
		"listIndex", listIndex,
% endif
% for field in key_fields:
		"${field["ident"]}", table.Get${field["ident"]}(),
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% endif