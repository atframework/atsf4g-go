## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta,partly_get,partly_field_name" />
<%
    index_type = index_meta["index_type"]
    batch_redis_key_call = index_meta["batch_redis_key_call"]
    key_fields = index_meta["key_fields"]
    args_redis_key_call = index_meta["args_redis_key_call"]
    cas_enabled = index_meta["cas_enabled"]
%>
% if index_type == "kv":
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
	${args_redis_key_call}
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
	ctx.LogDebug("partly load ${message_name} table with key from db success",
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

func ${message_name}BatchLoadWith${index_meta["index_key_name"]}PartlyGet${partly_field_name}(
	ctx cd.AwaitableContext,
	keys []${message_name}TableKey,
) (dbResult []${message_name}BatchGetResult, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	loadIndexs := make([]string, len(keys))
	for i := range keys {
		${batch_redis_key_call}
	}
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
	var messages []*RedisSetIndexMessage
	messages, retResult = HashTableBatchPartlyGet(
		ctx, loadIndexs, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		}, redisField)
	dbResult = make([]${message_name}BatchGetResult, len(messages))
	for index, message := range messages {
		if message == nil {
			dbResult[index].Result = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			ctx.LogDebug("partly load ${message_name} table with key from db error nil",
% for field in key_fields:
				"${field["ident"]}", keys[index].${field["ident"]},
% endfor
			)
			continue
		}
		dbResult[index].Result = message.Result
		if message.Result.IsError() {
			ctx.LogDebug("partly load ${message_name} table with key from db error",
				"error", message.Result,
% for field in key_fields:
				"${field["ident"]}", keys[index].${field["ident"]},
% endfor
			)
			continue
		}
		dbResult[index].Table = message.Message.(*private_protocol_pbdesc.${message_name})
% for field in key_fields:
		dbResult[index].Table.${field["ident"]} = keys[index].${field["ident"]}
% endfor
% if cas_enabled:
		dbResult[index].CASVersion = message.CASVersion
% endif
		ctx.LogDebug("partly load ${message_name} table with key from db success",
% if cas_enabled:
			"CASVersion", dbResult[index].CASVersion,
% endif
% for field in key_fields:
			"${field["ident"]}", keys[index].${field["ident"]},
% endfor
		)
	}
	return
}
% endif