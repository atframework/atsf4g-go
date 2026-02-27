## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    args_redis_key_call = index_meta["args_redis_key_call"]
    key_fields = index_meta["key_fields"]
%>
// ${message_name}ZRangeByRankWith${index_key_name} 使用 BYRANK 模式查询有序集合，固定带 WITHSCORES
// start/stop 为 rank 范围（0-based，支持负数）
// rev=true 时逆序（等价于 ZREVRANGE 语义）
func ${message_name}ZRangeByRankWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	start int64, stop int64, rev bool,
) (members []SortedSetRangeMember, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	members, retResult = SortedSetZRangeByRank(ctx, index, "${message_name}",
		dispatcher, instance, start, stop, rev)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("zrange by rank ${message_name} success",
		"count", len(members),
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

// ${message_name}ZRangeByScoreWith${index_key_name} 使用 BYSCORE 模式查询有序集合，固定带 WITHSCORES
// min/max 为分数范围边界
// offset 跳过数量
// count 返回最大数量
// rev=true 时逆序
func ${message_name}ZRangeByScoreWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	min SortedSetScoreBound, max SortedSetScoreBound, offset int64, count int64, rev bool,
) (members []SortedSetRangeMember, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	members, retResult = SortedSetZRangeByScore(ctx, index, "${message_name}",
		dispatcher, instance, min, max, offset, count, rev)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("zrange by score ${message_name} success",
		"count", len(members),
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
