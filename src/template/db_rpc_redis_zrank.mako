## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    args_redis_key_call = index_meta["args_redis_key_call"]
    key_fields = index_meta["key_fields"]
%>
// ${message_name}ZRankWith${index_key_name} 获取成员的排名和分数（ZRANK WITHSCORES）
// rev=true 时使用 ZREVRANK（按分数从高到低排名）
func ${message_name}ZRankWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	member string, rev bool,
) (result SortedSetZRankResult, found bool, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	result, found, retResult = SortedSetZRank(ctx, index, "${message_name}",
		dispatcher, instance, member, rev)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("zrank ${message_name} success",
		"rank", result.Rank,
		"score", result.Score,
		"found", found,
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
