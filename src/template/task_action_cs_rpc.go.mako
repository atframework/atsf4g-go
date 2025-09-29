## -*- coding: utf-8 -*-
<%!
import time
import os
import re
%><%
rpc_camel_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>// Copyright ${time.strftime("%Y", time.localtime()) } atframework

package ${ service_go_package_prefix }${ os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") }


import (
	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"

	service_protocol "${ protocol_go_module }"
)

type TaskAction${ rpc_camel_name } struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.${ rpc.get_request().get_name() }, *service_protocol.${ rpc.get_response().get_name() }]
}

func (t *TaskAction${ rpc_camel_name }) Name() string {
	return "TaskAction${ rpc_camel_name }"
}

func (t *TaskAction${ rpc_camel_name }) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	return nil
}

