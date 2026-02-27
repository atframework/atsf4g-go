## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    args_redis_key_call = index_meta["args_redis_key_call"]
    key_fields = index_meta["key_fields"]
%>
// ${message_name}ExpireWith${index_key_name} 对指定表的 Redis Key 设置过期时间（TTL）
// expiration: 过期时间（Duration）
// condition: 条件选项
//   - ExpireConditionNone: 不指定条件，始终设置过期时间
//   - ExpireConditionNX: 仅当键当前没有设置过期时间时才设置
//   - ExpireConditionXX: 仅当键当前已有过期时间时才设置
//   - ExpireConditionGT: 仅当新过期时间大于当前过期时间时才设置
//   - ExpireConditionLT: 仅当新过期时间小于当前过期时间时才设置
func ${message_name}ExpireWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	expiration time.Duration,
	condition ExpireConditionOption,
) (success bool, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	success, retResult = TableExpire(ctx, index, "${message_name}",
		dispatcher, instance, expiration, condition)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("expire ${message_name} success",
		"success", success,
		"expiration", expiration,
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}

// ${message_name}ExpireAtWith${index_key_name} 对指定表的 Redis Key 设置过期时间点（绝对时间戳）
// expireAt: 过期时间点（time.Time）
func ${message_name}ExpireAtWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
	expireAt time.Time,
) (success bool, retResult cd.RpcResult) {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	instance := dispatcher.GetRedisInstance()
	if instance == nil {
        ctx.LogError("get redis instance failed")
		retResult = cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	${args_redis_key_call}
	success, retResult = TableExpireAt(ctx, index, "${message_name}",
		dispatcher, instance, expireAt)
	if retResult.IsError() {
		return
	}
	ctx.LogDebug("expireat ${message_name} success",
		"success", success,
		"expireAt", expireAt,
% for field in key_fields:
		"${field["ident"]}", ${field["ident"]},
% endfor
	)
	retResult = cd.CreateRpcResultOk()
	return
}
