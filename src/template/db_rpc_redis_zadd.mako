## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    args_redis_key_call = index_meta["args_redis_key_call"]
    key_fields = index_meta["key_fields"]
%>
// ${message_name}ZAddWith${index_key_name} 向有序集合添加成员
// existence: ZAdd 存在性选项 (NX/XX)
// comparison: ZAdd 比较选项 (GT/LT)
func ${message_name}ZAddWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	members []SortedSetMember,
	existence ZAddExistenceOption,
	comparison ZAddComparisonOption,
) (retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	retResult = SortedSetZAdd(ctx, index, "${message_name}",
		dispatcher, instance, members, existence, comparison)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("zadd ${message_name} success",
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

// ${message_name}ZAddIncrWith${index_key_name} 向有序集合执行 ZADD INCR 操作（仅支持单个成员）
func ${message_name}ZAddIncrWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	member SortedSetMember,
) (result ZAddIncrResult, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	result, retResult = SortedSetZAddIncr(ctx, index, "${message_name}",
		dispatcher, instance, member)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("zadd incr ${message_name} success",
		"newScore", result.NewScore,
		"exists", result.Exists,
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
