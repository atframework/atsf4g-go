## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta,partly_get,partly_field_name" />
<%
    index_type_kv = index_meta["index_type_kv"]
    key_fields = index_meta["key_fields"]
    load_index_key = index_meta["load_index_key"]
    cas_enabled = index_meta["cas_enabled"]
%>
% if index_type_kv:
func ${message_name}LoadWith${index_meta["index_key_name"]}PartlyGet${partly_field_name}(
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
	redisField := []string{
% if cas_enabled:
		pu.CASKeyField,
% endif
% for field in key_fields:
		"${field["raw_name"]}",
% endfor
% for field_name in partly_get.fields:
		"${field_name}",
% endfor
	}
	var message proto.Message
% if cas_enabled:
	message, CASVersion, retResult = HashTablePartlyGet(
% else:
	message, _, retResult = HashTablePartlyGet(
% endif
		ctx, index, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		}, redisField)
	if retResult.IsError() {
		return
	}
	table = message.(*private_protocol_pbdesc.${message_name})
% for field in key_fields:
	table.${field["ident"]} = ${field["ident"]}
% endfor
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
% endif