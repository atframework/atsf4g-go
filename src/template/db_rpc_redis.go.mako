## -*- coding: utf-8 -*-
<%!
import time
import sys
%><%page args="message_name,extension,message,index_type_enum" />
%	for field in message.fields:
%		if not field.is_db_vaild_type():
// ${message_name} filed: {${field.get_name()}} not db vaild type
<% return %>
%		endif
%	endfor
%	for index in extension.index:
<%
    index_key_name = ""
    batch_args_key_fmt_args = ""
    args_key_fmt_args = ""
    struct_key_fmt_args = ""
    key_fields = []
    # index_type: "kv" / "kl" / "sorted_set"
    index_type = "kv"
    if index.type == index_type_enum.values_by_name["EN_ATFRAMEWORK_DB_INDEX_TYPE_KL"].descriptor.number:
        index_type = "kl"
    if "EN_ATFRAMEWORK_DB_INDEX_TYPE_SORTED_SET" in index_type_enum.values_by_name and \
       index.type == index_type_enum.values_by_name["EN_ATFRAMEWORK_DB_INDEX_TYPE_SORTED_SET"].descriptor.number:
        index_type = "sorted_set"

    for key in index.key_fields:
        field = message.fields_by_name[key]
        ident = message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
        index_key_name += ident
        batch_args_key_fmt_args += "keys[i]." + ident + ", "
        args_key_fmt_args += ident + ", "
        struct_key_fmt_args += "table.Get" + ident + "(), "
        key_fields.append({
            "raw_name": key,
            "ident": ident,
            "go_type": field.get_go_type(),
        })

    redis_key_func_name = message_name + "RedisKeyWith" + index_key_name
    args_redis_key_call = (
        "index := " + redis_key_func_name + "(ctx,\n\t   "
        + args_key_fmt_args
        + "\n    )"
    )
    struct_redis_key_call = (
        "index := " + redis_key_func_name + "(ctx,\n\t   "
        + struct_key_fmt_args
        + "\n    )"
    )
    batch_redis_key_call = (
        "loadIndexs[i] = " + redis_key_func_name + "(ctx,\n\t   "
        + batch_args_key_fmt_args
        + "\n    )"
    )

    atomic_inc_fields = []
    for inc_field in index.atomic_inc_fields:
        if inc_field not in message.fields_by_name:
            continue
        field = message.fields_by_name[inc_field]
        go_type = field.get_go_type()
        if go_type not in ("int32", "int64", "uint32", "uint64"):
            continue
        atomic_inc_fields.append({
            "raw_name": inc_field,
            "ident": message.get_identify_name(inc_field, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL),
            "go_type": go_type,
        })

    index_meta = {
        "index_type": index_type,
        "index_key_name": index_key_name,
        "args_redis_key_call": args_redis_key_call,
        "struct_redis_key_call": struct_redis_key_call,
        "batch_redis_key_call": batch_redis_key_call,
        "key_fields": key_fields,
        "cas_enabled": index.enable_cas,
        "max_list_length": index.max_list_length,
        "atomic_inc_fields": atomic_inc_fields,
    }
%>
## ========== 所有类型共用：RedisKey 生成 + Expire + ExpireAt ==========
<%include file="db_rpc_redis_key.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_expire.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />

% if index_type == "sorted_set":
## ========== Sorted Set 类型：ZAdd / ZRange / ZRank ==========
<%include file="db_rpc_redis_zadd.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_zrange.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_zrank.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_del.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
% else:
## ========== KV / KL 类型：Load / Del / Update ==========
<%include file="db_rpc_redis_load.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_del.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />
<%include file="db_rpc_redis_update.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />

%   if len(atomic_inc_fields) > 0:
%       for inc_field in atomic_inc_fields:
<%include file="db_rpc_redis_atomic_inc.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta,inc_field=inc_field" />
%       endfor
%   endif

%	for partly_get in index.partly_get:
<%
    partly_field_name = ""
    if partly_get.name != "":
        partly_field_name += message.get_identify_name(partly_get.name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
    else:
        for field in partly_get.fields:
            _ = message.fields_by_name[field]
            partly_field_name += message.get_identify_name(field, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
<%include file="db_rpc_redis_partly_get.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta,partly_get=partly_get,partly_field_name=partly_field_name" />
%	endfor
% endif

%	endfor