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
	"fmt"

	component_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"
	service_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

type TaskAction${ rpc_camel_name } struct {
	component_dispatcher.TaskActionCSBase[*service_protocol.${ rpc.get_request().get_name() }, *service_protocol.${ rpc.get_response().get_name() }]
}

func (t *TaskAction${ rpc_camel_name }) Name() string {
	return "TaskAction${ rpc_camel_name }"
}

func (t *TaskAction${ rpc_camel_name }) Run(_startData *component_dispatcher.DispatcherStartData) error {
	// TODO: implement your logic here, remove this comment after you have done
	user, ok := t.GetUser().(*data.User)
	if !ok || user == nil {
		t.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))
		return fmt.Errorf("user not found")
	}

	// request_body := t.GetRequestBody() // TODO
% if rpc.is_request_stream() or rpc.is_response_stream():
  	// Stream request or stream response, just ignore auto response
	t.DisableResponse()
% else:
	// response_body := t.MutableResponseBody() // TODO
%   if rpc.get_extension_field('rpc_options', lambda x: x.allow_no_wait, False):
	if t.IsStreamRpc() {
		t.DisableResponse()
	}
%   endif
% endif

	return nil
}
