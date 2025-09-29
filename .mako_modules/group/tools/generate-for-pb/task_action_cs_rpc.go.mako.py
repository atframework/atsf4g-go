# -*- coding:utf-8 -*-
from mako import runtime, filters, cache
UNDEFINED = runtime.UNDEFINED
STOP_RENDERING = runtime.STOP_RENDERING
__M_dict_builtin = dict
__M_locals_builtin = locals
_magic_number = 10
_modified_time = 1759149917.592502
_enable_loop = True
_template_filename = 'D:/workspace/git/github/atsf4g-go/src/template/task_action_cs_rpc.go.mako'
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
        protocol_go_module = context.get('protocol_go_module', UNDEFINED)
        output_render_path = context.get('output_render_path', UNDEFINED)
        PbConvertRule = context.get('PbConvertRule', UNDEFINED)
        rpc = context.get('rpc', UNDEFINED)
        service_go_package_prefix = context.get('service_go_package_prefix', UNDEFINED)
        __M_writer = context.writer()

        rpc_camel_name = rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL)
        
        
        __M_locals_builtin_stored = __M_locals_builtin()
        __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['rpc_camel_name'] if __M_key in __M_locals_builtin_stored]))
        __M_writer('// Copyright ')
        __M_writer(str(time.strftime("%Y", time.localtime()) ))
        __M_writer(' atframework\r\n\r\npackage ')
        __M_writer(str( service_go_package_prefix ))
        __M_writer(str( os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") ))
        __M_writer('\r\n\r\n\r\nimport (\r\n\tcomponent_dispatcher "github.com/atframework/atsf4g-go/component-dispatcher"\r\n\r\n\tservice_protocol "')
        __M_writer(str( protocol_go_module ))
        __M_writer('"\r\n)\r\n\r\ntype TaskAction')
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
        __M_writer(') Run(_startData *component_dispatcher.DispatcherStartData) error {\r\n\t// TODO: implement your logic here, remove this comment after you have done\r\n\treturn nil\r\n}\r\n\r\n')
        return ''
    finally:
        context.caller_stack._pop_frame()


"""
__M_BEGIN_METADATA
{"filename": "D:/workspace/git/github/atsf4g-go/src/template/task_action_cs_rpc.go.mako", "uri": "task_action_cs_rpc.go.mako", "source_encoding": "utf-8", "line_map": {"16": 2, "17": 3, "18": 4, "19": 5, "20": 6, "21": 7, "22": 0, "32": 6, "33": 7, "34": 8, "35": 9, "38": 8, "39": 8, "40": 8, "41": 10, "42": 10, "43": 10, "44": 16, "45": 16, "46": 19, "47": 19, "48": 20, "49": 20, "50": 20, "51": 20, "52": 23, "53": 23, "54": 24, "55": 24, "56": 27, "57": 27, "63": 57}}
__M_END_METADATA
"""
