## -*- coding: utf-8 -*-
<%!
import time
import os
import re
%><%
module_name = service.get_extension_field("service_options", lambda x: x.module_name, service.get_name_lower_rule())
imported_sub_modules = set()
%>// Copyright ${time.strftime("%Y", time.localtime()) } atframework
// @brief Created by ${generator} for ${service.get_full_name()}, please don't edit it

package ${ service_go_package_prefix }${ os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") }

import (
	"fmt"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	uc_d "github.com/atframework/atsf4g-go/component-user_controller/dispatcher"

	sp "${protocol_go_module}"

% for rpc in rpcs.values():
<%
	if rpc.get_request_descriptor().full_name == "google.protobuf.Empty":
		continue
	sub_module_path = 'logic/' + rpc.get_extension_field("rpc_options", lambda x: x.module_name, "")
	if sub_module_path.endswith("/"):
		sub_module_path = sub_module_path[:-1]
	sub_module_path = sub_module_path + "/action"
	sub_module_name = sub_module_path.replace("/", "_").replace("\\", "_").replace(".", "_")
	if sub_module_name in imported_sub_modules:
		continue
	imported_sub_modules.add(sub_module_name)
%>\
	${sub_module_name} "${service_go_module}/${sub_module_path}"
% endfor
)

func RegisterLobbyClientService(
	rd cd.DispatcherImpl, findSessionFn uc_d.FindCSMessageSession,
) error {
	svc := sp.File_${ service.file.get_name().replace("/", "_").replace("\\", "_").replace(".", "_") }.Services().ByName("${service.get_name()}")
	if svc == nil {
		rd.GetApp().GetDefaultLogger().Error("lobbysvr_app.RegisterLobbyClientService no service ${service.get_full_name()}")
		return fmt.Errorf("no service ${service.get_full_name()}")
	}

% for rpc in rpcs.values():
%   if not rpc.get_request_descriptor().full_name == "google.protobuf.Empty":
<%
			sub_module_path = 'logic/' + rpc.get_extension_field("rpc_options", lambda x: x.module_name, "action")
			if sub_module_path.endswith("/"):
				sub_module_path = sub_module_path[:-1]
			sub_module_path = sub_module_path + "/action"
			sub_module_name = sub_module_path.replace("/", "_").replace("\\", "_").replace(".", "_")
%>\
	uc_d.RegisterCSMessageAction(
		rd, findSessionFn, svc, "${ rpc.get_full_name() }",
		func(base cd.TaskActionCSBase[*sp.${ rpc.get_request().get_name() }, *sp.${ rpc.get_response().get_name() }]) cd.TaskActionImpl {
			ret :=  &${sub_module_name}.TaskAction${ rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL) }{TaskActionCSBase: base}
			ret.Impl = ret
			return ret
		},
	)
%   endif
% endfor

	return nil
}
