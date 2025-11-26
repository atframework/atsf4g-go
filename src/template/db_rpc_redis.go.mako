## -*- coding: utf-8 -*-
<%!
import time
import sys
%><%page args="message_name,extension,message,index_type_enum,split_type_enum" />
%	for field in message.fields:
%		if not field.is_db_vaild_type():
// ${message_name} filed: {${field.get_name()}} not db vaild type
<% return %>
%		endif
%	endfor
%	for index in extension.index:
<%
    db_fmt_key = index.name
    index_key_name = ""
    load_key_fmt_args = ""
    update_key_fmt_args = ""
    key_fields = []

    for key in index.key_fields:
        field = message.fields_by_name[key]
        ident = message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
        index_key_name += ident
        db_fmt_key += "." + field.get_go_fmt_type()
        load_key_fmt_args += ident + ", "
        update_key_fmt_args += "table.Get" + ident + "(), "
        key_fields.append({
            "raw_name": key,
            "ident": ident,
            "go_type": field.get_go_type(),
        })

    prefix_fmt_key = "%s"
    prefix_fmt_value = "dispatcher.GetRecordPrefix()"

    if index.split_type == split_type_enum.values_by_name["EN_ATFRAMEWORK_DB_TABLE_SPLIT_TYPE_WORLD"].descriptor.number:
        prefix_fmt_key += "-%d-"
        prefix_fmt_value += ", ctx.GetApp().GetWorldId()"
    if index.split_type == split_type_enum.values_by_name["EN_ATFRAMEWORK_DB_TABLE_SPLIT_TYPE_WORLD_ZONE"].descriptor.number:
        prefix_fmt_key += "-%d-%d-"
        prefix_fmt_value += ", ctx.GetApp().GetWorldId(), ctx.GetApp().GetZoneId()"

    load_index_key = (
        "index := fmt.Sprintf(\"" + prefix_fmt_key + db_fmt_key + "\", "
        + prefix_fmt_value
        + ",\n\t   "
        + load_key_fmt_args
        + "\n    )"
    )
    update_index_key = (
        "index := fmt.Sprintf(\"" + prefix_fmt_key + db_fmt_key + "\", "
        + prefix_fmt_value
        + ",\n\t   "
        + update_key_fmt_args
        + "\n    )"
    )

    index_meta = {
        "index_key_name": index_key_name,
        "load_index_key": load_index_key,
        "update_index_key": update_index_key,
        "key_fields": key_fields,
        "cas_enabled": index.enable_cas,
    }
%>
<%include file="db_rpc_redis_load.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />

<%include file="db_rpc_redis_update.mako" args="message_name=message_name,message=message,index=index,index_meta=index_meta" />

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

%	endfor