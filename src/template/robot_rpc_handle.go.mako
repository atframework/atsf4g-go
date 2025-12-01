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
if rpc.get_request_descriptor().full_name == "google.protobuf.Empty":
  continue

rpc_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
func Send${rpc_name}(user User, reqBody *lobysvr_protocol_pbdesc.${rpc.get_request().get_name()}) error {
	if lu.IsNil(user) || reqBody == nil {
		return fmt.Errorf("user or request is nil")
	}
	csMsg := &public_protocol_extension.CSMsg{
		Head: user.MakeMessageHead("${rpc.descriptor.full_name}", "${rpc.get_request_descriptor().full_name}"),
	}
	csMsg.BodyBin, _ = proto.Marshal(reqBody)
	return user.SendReq(csMsg, reqBody)
}
% endfor

% for rpc in rpcs.values():
<%
if rpc.get_response_descriptor().full_name == "google.protobuf.Empty":
  continue

rpc_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
func Get${rpc_name}TypeName() string {
	return "${rpc.descriptor.full_name}"
}
% endfor