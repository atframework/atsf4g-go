## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
	index_type_kv = index_meta["index_type_kv"]
	index_key_name = index_meta["index_key_name"]
	args_index_key = index_meta["args_index_key"]
	batch_args_index_key = index_meta["batch_args_index_key"]
	key_fields = index_meta["key_fields"]
	cas_enabled = index_meta["cas_enabled"]
%>
% if index_type_kv:
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
	${args_index_key}
	var message proto.Message
% if cas_enabled:
	message, CASVersion, retResult = HashTableLoad(
% else:
	message, _, retResult = HashTableLoad(
% endif
		ctx, index, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		})
	if retResult.IsError() {
		return
	}
	table = message.(*private_protocol_pbdesc.${message_name})
% for field in key_fields:
	table.${field["ident"]} = ${field["ident"]}
% endfor
	ctx.LogDebug("load ${message_name} table with key from db success",
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

type ${message_name}TableKey struct {
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]}
% endfor
};

type ${message_name}BatchGetResult struct {
	Table *private_protocol_pbdesc.${message_name}
% if cas_enabled:
	CASVersion uint64
% endif
	Result cd.RpcResult
}

func ${message_name}BatchLoadWith${index_key_name}(
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
		${batch_args_index_key}
	}
	var messages []*RedisSetIndexMessage
	messages, retResult = HashTableBatchLoad(
		ctx, loadIndexs, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		})
	if retResult.IsError() {
		return
	}
	dbResult = make([]${message_name}BatchGetResult, len(messages))
	for index, message := range messages {
		if message == nil {
			dbResult[index].Result = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
			ctx.LogDebug("load ${message_name} table with key from db error nil",
% for field in key_fields:
				"${field["ident"]}", keys[index].${field["ident"]},
% endfor
			)
			continue
		}
		dbResult[index].Result = message.Result
		if message.Result.IsError() {
			ctx.LogDebug("load ${message_name} table with key from db error",
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
		ctx.LogDebug("load ${message_name} table with key from db success",
% if cas_enabled:
			"CASVersion", dbResult[index].CASVersion,
% endif
% for field in key_fields:
			"${field["ident"]}", keys[index].${field["ident"]},
% endfor
		)
	}
	retResult = cd.CreateRpcResultOk()
	return
}
% else:
func ${message_name}LoadAllWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
) (indexMessage []pu.RedisListIndexMessage, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_index_key}
	indexMessage, retResult = HashTableLoadListAll(
		ctx, index, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		})
	if retResult.IsError() {
		return
	}
	for _, item := range indexMessage {
		table, ok := item.Table.(*private_protocol_pbdesc.${message_name})
		if ok {
% for field in key_fields:
			table.${field["ident"]} = ${field["ident"]}
% endfor
		}
	}
	ctx.LogDebug("load ${message_name} table with key from db success",
		"index_message_len", len(indexMessage),
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

func ${message_name}LoadIndexWith${index_key_name}(
	ctx cd.AwaitableContext,
	listIndex []uint64,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
) (indexMessage []pu.RedisListIndexMessage, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_index_key}
	indexMessage, retResult = HashTableLoadListIndex(
		ctx, index, "${message_name}", dispatcher, instance, func() proto.Message {
			return new(private_protocol_pbdesc.${message_name})
		}, listIndex)
	if retResult.IsError() {
		return
	}
	for _, item := range indexMessage {
		table, ok := item.Table.(*private_protocol_pbdesc.${message_name})
		if ok {
% for field in key_fields:
			table.${field["ident"]} = ${field["ident"]}
% endfor
		}
	}
	ctx.LogDebug("load ${message_name} table with key from db success",
		"index_message_len", len(indexMessage),
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
% endif