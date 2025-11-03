## -*- coding: utf-8 -*-
<%!
import time
import os
import re
%>// Copyright ${time.strftime("%Y", time.localtime()) } atframework
// @brief Created by ${generator}, please don't edit it

package atframework_component_db

import (
	"fmt"
	"os"
    private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	atfw_utils_fs "github.com/atframework/atframe-utils-go/file_system"
	"google.golang.org/protobuf/proto"
)
<%
file = database.get_file(generate_proto_file) %>
%	for message_name, message_desc in file.descriptor.message_types_by_name.items():
<%
        if file.get_package() == '':
                full_name = message_desc.name
        else:
                full_name = file.get_package() + '.' + message_desc.name
        message = database.get_message(full_name)
        if message == None:
                continue
        extension = message.get_extension("atframework.database_table")
        if extension == None:
                continue %>
%       for index in extension.index:
<%
                index_key_name = ""
                for key in index.key_fields:
                        index_key_name = index_key_name + message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
                db_fmt_key = index.name
                for key in index.key_fields:
                    db_fmt_key = db_fmt_key + "." + message.fields_by_name[key].get_go_fmt_type() %>
func ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}LoadWith${index_key_name}(
	ctx *cd.RpcContext,
%               for key in index.key_fields:
	${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} ${message.fields_by_name[key].get_go_type()},
%               endfor
) (*private_protocol_pbdesc.${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}, cd.RpcResult) {
	index := fmt.Sprintf("%s${db_fmt_key}.db", "../data/",
%               for key in index.key_fields:
		${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%               endfor
    )

	if _, serr := os.Stat(index); serr != nil {
		return nil, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND)
	}

	binData, err := atfw_utils_fs.ReadAllContent(index)
	if err != nil {
		return nil, cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}Table := &private_protocol_pbdesc.${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}{}
	if err = proto.Unmarshal(binData, ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}Table); err != nil {
		return nil, cd.CreateRpcResultError(fmt.Errorf("failed to unmarshal ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} db data: %w", err),
                        public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE)
	}

	ctx.LogInfo("load ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)} login table with key from db success",
%               for key in index.key_fields:
	"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", ${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
%               endfor
	)
	return ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}Table, cd.CreateRpcResultOk()
}

func ${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}Update${index_key_name}(
	ctx *cd.RpcContext,
	table *private_protocol_pbdesc.${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)},
) cd.RpcResult {
	index := fmt.Sprintf("%s${db_fmt_key}.db", "../data/",
%               for key in index.key_fields:
		table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%               endfor
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
			"tableName", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}",
%               for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%               endfor
		)
		return result
	}

	binData, err := proto.Marshal(table)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to marshal %s db data: %w", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_BAD_PACKAGE),
		}
		result.LogError(ctx, "failed to marshal db data",
			"tableName", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}",
%               for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%               endfor
		)
		return result
	}

	f, err := os.Create(index)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to create %s db file: %w", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM_ACCESS_DENY),
		}
		result.LogError(ctx, "failed to create db file",
			"tableName", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}",
%               for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%               endfor
		)
		return result
	}
	defer f.Close()

	_, err = f.Write(binData)
	if err != nil {
		result = cd.RpcResult{
			Error:        fmt.Errorf("failed to write &s db file: %w", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", err),
			ResponseCode: int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM),
		}
		result.LogError(ctx, "failed to write db file",
			"tableName", "${message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}",
%               for key in index.key_fields:
			"${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}", table.Get${message.get_identify_name(key, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)}(),
%               endfor
		)
		return result
	}

	return cd.CreateRpcResultOk()
}
%       endfor
%   endfor