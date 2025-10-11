# -*- coding:utf-8 -*-
from mako import runtime, filters, cache
UNDEFINED = runtime.UNDEFINED
STOP_RENDERING = runtime.STOP_RENDERING
__M_dict_builtin = dict
__M_locals_builtin = locals
_magic_number = 10
_modified_time = 1760170784.7649565
_enable_loop = True
_template_filename = 'D:/GIT/server/src/template/task_action_cs_rpc.go.mako'
_template_uri = 'task_action_cs_rpc.go.mako'
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
        rpc = context.get('rpc', UNDEFINED)
        output_render_path = context.get('output_render_path', UNDEFINED)
        __M_writer = context.writer()

        rpc_camel_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
        
        
        __M_locals_builtin_stored = __M_locals_builtin()
        __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['rpc_camel_name'] if __M_key in __M_locals_builtin_stored]))
        __M_writer('// Copyright ')
        __M_writer(str(time.strftime("%Y", time.localtime()) ))
        __M_writer(' atframework\r\n\r\npackage ')
        __M_writer(str( service_go_package_prefix ))
        __M_writer(str( os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") ))
        __M_writer('\r\n\r\nimport (\r\n\t"fmt"\r\n\r\n\tcomponent_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"\r\n\tpublic_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"\r\n\tdata "github.com/atframework/atsf4g-go/service-lobbysvr/data"\r\n\tservice_protocol "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"\r\n)\r\n\r\ntype TaskAction')
        __M_writer(str( rpc_camel_name ))
        __M_writer(' struct {\r\n\tcomponent_dispatcher.TaskActionCSBase[*service_protocol.')
        __M_writer(str( rpc.get_request().get_name() ))
        __M_writer(', *service_protocol.')
        __M_writer(str( rpc.get_response().get_name() ))
        __M_writer(']\r\n}\r\n\r\nfunc (t *TaskAction')
        __M_writer(str( rpc_camel_name ))
        __M_writer(') Name() string {\r\n\treturn "TaskAction')
        __M_writer(str( rpc_camel_name ))
        __M_writer('"\r\n}\r\n\r\nfunc (t *TaskAction')
        __M_writer(str( rpc_camel_name ))
        __M_writer(') Run(_startData *component_dispatcher.DispatcherStartData) error {\r\n\t// TODO: implement your logic here, remove this comment after you have done\r\n\tuser, ok := t.GetUser().(*data.User)\r\n\tif !ok || user == nil {\r\n\t\tt.SetResponseCode(int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_USER_NOT_FOUND))\r\n\t\treturn fmt.Errorf("user not found")\r\n\t}\r\n\r\n\t// reqBody := t.GetRequestBody() // TODO\r\n')
        if rpc.is_request_stream() or rpc.is_response_stream():
            __M_writer('  \t// Stream request or stream response, just ignore auto response\r\n\tt.DisableResponse()\r\n')
        else:
            __M_writer('\t// rspBody := t.MutableResponseBody() // TODO\r\n')
            if rpc.get_extension_field('rpc_options', lambda x: x.allow_no_wait, False):
                __M_writer('\tif t.IsStreamRpc() {\r\n\t\tt.DisableResponse()\r\n\t}\r\n')
        __M_writer('\r\n\treturn nil\r\n}\r\n\r\n')
        return ''
    finally:
        context.caller_stack._pop_frame()


"""
__M_BEGIN_METADATA
{"filename": "D:/GIT/server/src/template/task_action_cs_rpc.go.mako", "uri": "task_action_cs_rpc.go.mako", "source_encoding": "utf-8", "line_map": {"16": 2, "17": 3, "18": 4, "19": 5, "20": 6, "21": 7, "22": 0, "31": 6, "32": 7, "33": 8, "34": 9, "37": 8, "38": 8, "39": 8, "40": 10, "41": 10, "42": 10, "43": 21, "44": 21, "45": 22, "46": 22, "47": 22, "48": 22, "49": 25, "50": 25, "51": 26, "52": 26, "53": 29, "54": 29, "55": 38, "56": 39, "57": 41, "58": 42, "59": 43, "60": 44, "61": 49, "67": 61}}
__M_END_METADATA
"""
