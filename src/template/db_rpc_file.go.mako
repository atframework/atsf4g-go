## -*- coding: utf-8 -*-
<%!
import time
%><%page args="message_name,extension,message" />
%   for index in extension.index:
<%
    index_key_name = ""
    for key in index.key_fields:
        index_key_name = index_key_name + message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
    db_fmt_key = index.name
    for key in index.key_fields:
        db_fmt_key = db_fmt_key + "." + message.fields_by_name[key].get_go_fmt_type() %>
func ${message_name}LoadWith${index_key_name}(
	ctx *cd.RpcContext,
%           for key in index.key_fields:
	${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} ${message.fields_by_name[key].get_go_type()},
%           endfor
) (*private_protocol_pbdesc.${message_name}, cd.RpcResult) {
	index := fmt.Sprintf("%s${db_fmt_key}.db", "../data/",
%           for key in index.key_fields:
		${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%           endfor
    )

	if _, serr := os.Stat(index); serr != nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND)
	}

	binData, err := atfw_utils_fs.ReadAllContent(index)
	if err != nil {
		return nil, cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	${message_name}Table := &private_protocol_pbdesc.${message_name}{}
	if err = proto.Unmarshal(binData, ${message_name}Table); err != nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("failed to unmarshal ${message_name} db data: %w", err),
                        public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
	}

	ctx.LogInfo("load ${message_name} login table with key from db success",
%           for key in index.key_fields:
	"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", ${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%           endfor
	)
	return ${message_name}Table, cd.CreateRpcResultOk()
}

func ${message_name}Update${index_key_name}(
	ctx *cd.RpcContext,
	table *private_protocol_pbdesc.${message_name},
) cd.RpcResult {
	index := fmt.Sprintf("%s${db_fmt_key}.db", "../data/",
%           for key in index.key_fields:
		table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
    )

	if _, serr := os.Stat("../data"); serr != nil {
		os.MkdirAll("../data", 0o755)
	}

	var result cd.RpcResult
	if ds, serr := os.Stat("../data"); serr != nil || !ds.IsDir() {
		result = cd.RpcResult{
			Error:        fmt.Errorf("../data is not a directory or can not be created as a directory"),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}

		result.LogError(ctx, "failed to create ../data directory",
			"tableName", "${message_name}",
%           for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
		)
		return result
	}

	binData, err := proto.Marshal(table)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to marshal %s db data: %w", "${message_name}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE),
		}
		result.LogError(ctx, "failed to marshal db data",
			"tableName", "${message_name}",
%           for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
		)
		return result
	}

	f, err := os.Create(index)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create %s db file: %w", "${message_name}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create db file",
			"tableName", "${message_name}",
%           for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
		)
		return result
	}
	defer f.Close()

	_, err = f.Write(binData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write %s db file: %w", "${message_name}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write db file",
			"tableName", "${message_name}",
%           for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%           endfor
		)
		return result
	}

	return cd.CreateRpcResultOk()
}
%   endfor