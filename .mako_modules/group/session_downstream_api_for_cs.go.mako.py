# -*- coding:utf-8 -*-
from mako import runtime, filters, cache
UNDEFINED = runtime.UNDEFINED
STOP_RENDERING = runtime.STOP_RENDERING
__M_dict_builtin = dict
__M_locals_builtin = locals
_magic_number = 10
_modified_time = 1760155656.993851
_enable_loop = True
_template_filename = 'D:/workspace/git/github/atsf4g-go/src/template/session_downstream_api_for_cs.go.mako'
_template_uri = 'session_downstream_api_for_cs.go.mako'
_source_encoding = 'utf-8'
_exports = []



import time
import os
import re


def render_body(context,**pageargs):
    __M_caller = context.caller_stack._push_frame()
    try:
        __M_locals = __M_dict_builtin(pageargs=pageargs)
        PbConvertRule = context.get('PbConvertRule', UNDEFINED)
        service_go_package_prefix = context.get('service_go_package_prefix', UNDEFINED)
        protocol_go_module = context.get('protocol_go_module', UNDEFINED)
        generator = context.get('generator', UNDEFINED)
        service = context.get('service', UNDEFINED)
        rpcs = context.get('rpcs', UNDEFINED)
        output_render_path = context.get('output_render_path', UNDEFINED)
        __M_writer = context.writer()

        module_name = service.get_extension_field("service_options", lambda x: x.module_name, service.get_name_lower_rule())
        
        
        __M_locals_builtin_stored = __M_locals_builtin()
        __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['module_name'] if __M_key in __M_locals_builtin_stored]))
        __M_writer('// Copyright ')
        __M_writer(str(time.strftime("%Y", time.localtime()) ))
        __M_writer(' atframework\r\n// @brief Created by ')
        __M_writer(str(generator))
        __M_writer(' for ')
        __M_writer(str(service.get_full_name()))
        __M_writer(", please don't edit it\r\n\r\npackage ")
        __M_writer(str( service_go_package_prefix ))
        __M_writer(str( os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") ))
        __M_writer('\r\n\r\n\r\nimport (\r\n\t"fmt"\r\n\t"time"\r\n\r\n\t"google.golang.org/protobuf/proto"\r\n\t"google.golang.org/protobuf/types/known/timestamppb"\r\n\r\n\tcd "github.com/atframework/atsf4g-go/component-dispatcher"\r\n\tuc "github.com/atframework/atsf4g-go/component-user_controller"\r\n\r\n\tppe "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"\r\n\r\n\tsp "')
        __M_writer(str(protocol_go_module))
        __M_writer('"\r\n)\r\n\r\nfunc sendMessage(responseCode int32, session *uc.Session,\r\n\trd cd.DispatcherImpl, now time.Time,\r\n\trpcType interface{}, body proto.Message,\r\n) error {\r\n\tmsg, err := cd.CreateCSMessage(responseCode, now, 0,\r\n\t\trd, session,\r\n\t\trpcType, body)\r\n\tif err != nil {\r\n\t\treturn err\r\n\t}\r\n\r\n\treturn session.SendMessage(msg)\r\n}\r\n\r\n')
        for rpc in rpcs.values():

            if rpc.get_request_descriptor().full_name != "google.protobuf.Empty":
              continue
            
            rpc_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
            
            
            __M_locals_builtin_stored = __M_locals_builtin()
            __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['rpc_name'] if __M_key in __M_locals_builtin_stored]))
            __M_writer('\r\nfunc Send')
            __M_writer(str(rpc_name))
            __M_writer('(session *uc.Session, body *sp.')
            __M_writer(str(rpc.get_response().get_name()))
            __M_writer(', responseCode int32) error {\r\n\tif session == nil || body == nil {\r\n\t\treturn fmt.Errorf("session or message body is nil")\r\n\t}\r\n\r\n\trd := session.GetNetworkHandle().GetDispatcher()\r\n\tif rd == nil {\r\n\t\treturn fmt.Errorf("session dispatcher is nil")\r\n\t}\r\n\r\n\tnow := rd.GetNow()\r\n\r\n\treturn sendMessage(responseCode, session, rd, now, &ppe.RpcStreamMeta{\r\n\t\tVersion:         "0.1.0",  // TODO: make it configurable\r\n\t\tRpcName:         "')
            __M_writer(str(rpc.get_full_name()))
            __M_writer('",\r\n\t\tTypeUrl:         "')
            __M_writer(str(rpc.get_response().get_full_name()))
            __M_writer('",\r\n\t\tCaller:          rd.GetApp().GetTypeName(),\r\n\t\tCallerTimestamp: timestamppb.New(now),\r\n\t}, body)\r\n}\r\n')
        return ''
    finally:
        context.caller_stack._pop_frame()


"""
__M_BEGIN_METADATA
{"filename": "D:/workspace/git/github/atsf4g-go/src/template/session_downstream_api_for_cs.go.mako", "uri": "session_downstream_api_for_cs.go.mako", "source_encoding": "utf-8", "line_map": {"16": 2, "17": 3, "18": 4, "19": 5, "20": 6, "21": 7, "22": 0, "34": 6, "35": 7, "36": 8, "37": 9, "40": 8, "41": 8, "42": 8, "43": 9, "44": 9, "45": 9, "46": 9, "47": 11, "48": 11, "49": 11, "50": 26, "51": 26, "52": 43, "53": 44, "54": 45, "55": 46, "56": 47, "57": 48, "58": 49, "59": 50, "62": 49, "63": 50, "64": 50, "65": 50, "66": 50, "67": 64, "68": 64, "69": 65, "70": 65, "76": 70}}
__M_END_METADATA
"""
