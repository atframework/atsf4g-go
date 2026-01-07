## -*- coding: utf-8 -*-
<%!
import time
import os
import re
%><%
module_name = service.get_extension_field("service_options", lambda x: x.module_name, service.get_name_lower_rule())
%>// Copyright ${time.strftime("%Y", time.localtime()) } atframework
// @brief Created by ${generator} for ${service.get_full_name()}, please don't edit it

package atsf4g_go_robot_user

import (
	"fmt"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
	"google.golang.org/protobuf/proto"
)

% for rpc in rpcs.values():
<%
rpc_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
% if rpc.get_request_descriptor().full_name == "google.protobuf.Empty":
func Get${rpc_name}ResponseRpcName() string {
	return "${rpc.descriptor.full_name}"
}
% else:
func Send${rpc_name}(task *TaskActionUser, reqBody *lobysvr_protocol_pbdesc.${rpc.get_request().get_name()}, needLogin bool) (
	int32, *lobysvr_protocol_pbdesc.${rpc.get_response().get_name()}, error) {
	if lu.IsNil(task.User) || reqBody == nil {
		return 0, nil, fmt.Errorf("user or request is nil")
	}
	if needLogin {
		if !task.User.IsLogin() {
			return 0, nil, fmt.Errorf("user not login")
		}
	}
	csMsg := &public_protocol_extension.CSMsg{
		Head: task.User.MakeMessageHead("${rpc.descriptor.full_name}", "${rpc.get_request_descriptor().full_name}"),
	}
	csMsg.BodyBin, _ = proto.Marshal(reqBody)
	code, bodyRaw, err := task.User.SendReq(task, csMsg, reqBody, true)
	body, ok := bodyRaw.(*lobysvr_protocol_pbdesc.${rpc.get_response().get_name()})
	if !ok {
		return code, nil, fmt.Errorf("type assertion to ${rpc.get_response().get_name()} failed")
	}
	return code, body, err
}
% endif
% endfor