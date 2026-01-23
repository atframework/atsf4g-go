## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_type_kv = index_meta["index_type_kv"]
    index_key_name = index_meta["index_key_name"]
    args_index_key = index_meta["args_index_key"]
    key_fields = index_meta["key_fields"]
    cas_enabled = index_meta["cas_enabled"]
%>
func ${message_name}DelWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_index_key}

	retResult = HashTableDel(ctx, index, "${message_name}", dispatcher, instance)
	if retResult.IsError() {
		return
	}

	ctx.LogDebug("delete ${message_name} table with key from db success",
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

% if not index_type_kv:
func ${message_name}DelIndexWith${index_key_name}(
	ctx cd.AwaitableContext,
	listIndex []uint64,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_index_key}

	retResult = HashTableDelListIndex(ctx, index, "${message_name}", dispatcher, instance, listIndex)
	if retResult.IsError() {
		ctx.LogError("delete ${message_name} table with key from db failed",
			"listIndex", listIndex,
	% for field in key_fields:
			"${field["ident"]}", ${field["ident"]},
	% endfor
			"error", retResult,
		)
		return
	}

	ctx.LogDebug("delete index ${message_name} table with key from db success",
		"listIndex", listIndex,
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
		"list_index", listIndex,
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% endif