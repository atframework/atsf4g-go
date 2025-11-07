## -*- coding: utf-8 -*-
<%!
import time
import os
import re
%><%
module_name = service.get_extension_field("service_options", lambda x: x.module_name, service.get_name_lower_rule())
%>// Copyright ${time.strftime("%Y", time.localtime()) } atframework
// @brief Created by ${generator} for ${service.get_full_name()}, please don't edit it

package ${ service_go_package_prefix }${ os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") }


import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	pu "github.com/atframework/atframe-utils-go/proto_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	ppe "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"

	sp "${protocol_go_module}"
)

func sendMessage(responseCode int32, session cd.TaskActionCSSession,
	rd cd.DispatcherImpl, now time.Time,
	rpcType interface{}, body proto.Message,
) error {
	if lu.IsNil(session) {
		return fmt.Errorf("session is nil")
	}
	msg, err := cd.CreateCSMessage(responseCode, now, 0,
		rd, session,
		rpcType, body)
	if err != nil {
		return err
	}

	var rpcUrl string
	switch v := rpcType.(type) {
	case *ppe.RpcResponseMeta:
		rpcUrl = v.TypeUrl
	case *ppe.RpcRequestMeta:
		rpcUrl = v.TypeUrl
	case *ppe.RpcStreamMeta:
		rpcUrl = v.TypeUrl
	}

	logWriter := session.GetActorLogWriter()
	if !lu.IsNil(logWriter) {
		fmt.Fprintf(logWriter, "%s >>>>>>>>>>>>>>>>>>>> Sending: %s\n", time.Now().Format(time.DateTime), rpcUrl)
		fmt.Fprintf(logWriter, "Head:{\n%s}\n", pu.MessageReadableText(msg.Head))
		fmt.Fprintf(logWriter, "Body:{\n%s}\n\n", pu.MessageReadableText(body))
	}

	return session.SendMessage(msg)
}

% for rpc in rpcs.values():
<%
if rpc.get_request_descriptor().full_name != "google.protobuf.Empty":
  continue

rpc_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
%>
func Send${rpc_name}(session cd.TaskActionCSSession, body *sp.${rpc.get_response().get_name()}, responseCode int32) error {
	if lu.IsNil(session) || body == nil {
		return fmt.Errorf("session or message body is nil")
	}

	rd := session.GetDispatcher()
	if rd == nil {
		return fmt.Errorf("session dispatcher is nil")
	}

	now := rd.GetNow()

	return sendMessage(responseCode, session, rd, now, &ppe.RpcStreamMeta{
		Version:         "0.1.0",  // TODO: make it configurable
		RpcName:         "${rpc.get_full_name()}",
		TypeUrl:         "${rpc.get_response().get_full_name()}",
		Caller:          rd.GetApp().GetTypeName(),
		CallerTimestamp: timestamppb.New(now),
	}, body)
}
% endfor
