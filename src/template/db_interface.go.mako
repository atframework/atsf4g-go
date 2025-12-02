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
	"strconv"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	pu "github.com/atframework/atframe-utils-go/proto_utility"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
    config "github.com/atframework/atsf4g-go/component-config"

	"github.com/atframework/libatapp-go"
)
<%
file = database.get_file(generate_proto_file)
index_type_enum = database.get_enum("atframework.database_index_type")
split_type_enum = database.get_enum("atframework.database_table_split_type")
if index_type_enum is None or split_type_enum is None:
    return
%>
%	for message_name, message_desc in file.descriptor.message_types_by_name.items():
<%
    package_name = file.get_package()
    full_name = message_desc.name if package_name == '' else f"{package_name}.{message_desc.name}"
    message = database.get_message(full_name)
    if message is None:
        continue
    extension = message.get_extension("atframework.database_table")
    if extension is None:
        continue
    message_name = message.get_identify_name(message_name, PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
<%include file="db_rpc_redis.go.mako" args="message_name=message_name,extension=extension,message=message,index_type_enum=index_type_enum,split_type_enum=split_type_enum" />
%   endfor