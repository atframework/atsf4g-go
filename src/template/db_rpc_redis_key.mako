## -*- coding: utf-8 -*-
<%page args="message_name,message,index,index_meta" />
<%
    index_key_name = index_meta["index_key_name"]
    key_fields = index_meta["key_fields"]

    # 在本模板内计算 args_index_key（fmt.Sprintf 形式的 Redis Key 生成）
    db_fmt_key = index.name
    args_key_fmt_args = ""
    for key in index.key_fields:
        field = message.fields_by_name[key]
        db_fmt_key += "." + field.get_go_fmt_type()
        ident = message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
        args_key_fmt_args += ident + ", "

    prefix_fmt_key = "%s-"
    prefix_fmt_value = "dispatcher.GetRecordPrefix()"
    args_index_key = (
        "index := fmt.Sprintf(\"" + prefix_fmt_key + db_fmt_key + "\", "
        + prefix_fmt_value
        + ",\n\t   "
        + args_key_fmt_args
        + "\n    )"
    )
%>

// ${message_name}RedisKeyWith${index_key_name} 根据主键生成 Redis Key
func ${message_name}RedisKeyWith${index_key_name}(
	ctx cd.AwaitableContext,
% for field in key_fields:
    ${field["ident"]} ${field["go_type"]},
% endfor
) string {
	dispatcher := libatapp.AtappGetModule[*cd.RedisMessageDispatcher](ctx.GetApp())
	${args_index_key}
	return index
}
