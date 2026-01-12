## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta,inc_field" />
<%
    index_type_kv = index_meta["index_type_kv"]
    index_key_name = index_meta["index_key_name"]
    load_index_key = index_meta["load_index_key"]
    key_fields = index_meta["key_fields"]
    go_type = inc_field["go_type"]
    is_unsigned = go_type.startswith("uint")
%>
% if index_type_kv:
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

    newValue, retResult = HashTableAtomicInc(ctx, index, "${message_name}",
        dispatcher, instance, "${inc_field["raw_name"]}", uint64(incValue))
    if retResult.IsError() {
        return
    }

    ctx.LogDebug("atomic inc ${message_name} table with key from db success",
        "incValue", incValue,
        "${inc_field["ident"]}", newValue,
% for field in key_fields:
        "${field["ident"]}", ${field["ident"]},
% endfor
    )
    retResult = cd.CreateRpcResultOk()
    return
}
% endif