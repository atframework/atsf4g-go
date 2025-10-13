# -*- coding:utf-8 -*-
from mako import runtime, filters, cache
UNDEFINED = runtime.UNDEFINED
STOP_RENDERING = runtime.STOP_RENDERING
__M_dict_builtin = dict
__M_locals_builtin = locals
_magic_number = 10
_modified_time = 1760155656.9754713
_enable_loop = True
_template_filename = 'D:/workspace/git/github/atsf4g-go/src/template/handle_cs_rpc.go.mako'
_template_uri = 'handle_cs_rpc.go.mako'
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
        service_go_module = context.get('service_go_module', UNDEFINED)
        generator = context.get('generator', UNDEFINED)
        service = context.get('service', UNDEFINED)
        rpcs = context.get('rpcs', UNDEFINED)
        set = context.get('set', UNDEFINED)
        output_render_path = context.get('output_render_path', UNDEFINED)
        __M_writer = context.writer()

        module_name = service.get_extension_field("service_options", lambda x: x.module_name, service.get_name_lower_rule())
        imported_sub_modules = set()
        
        
        __M_locals_builtin_stored = __M_locals_builtin()
        __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['imported_sub_modules','module_name'] if __M_key in __M_locals_builtin_stored]))
        __M_writer('// Copyright ')
        __M_writer(str(time.strftime("%Y", time.localtime()) ))
        __M_writer(' atframework\r\n// @brief Created by ')
        __M_writer(str(generator))
        __M_writer(' for ')
        __M_writer(str(service.get_full_name()))
        __M_writer(", please don't edit it\r\n\r\npackage ")
        __M_writer(str( service_go_package_prefix ))
        __M_writer(str( os.path.dirname(output_render_path).replace("/", "_").replace("\\", "_") ))
        __M_writer('\r\n\r\nimport (\r\n\t"fmt"\r\n\r\n\tcd "github.com/atframework/atsf4g-go/component-dispatcher"\r\n\tuc "github.com/atframework/atsf4g-go/component-user_controller"\r\n\r\n\tsp "')
        __M_writer(str(protocol_go_module))
        __M_writer('"\r\n\r\n')
        for rpc in rpcs.values():

            if rpc.get_request_descriptor().full_name == "google.protobuf.Empty":
                    continue
            sub_module_path = 'logic/' + rpc.get_extension_field("rpc_options", lambda x: x.module_name, "action")
            sub_module_name = sub_module_path.replace("/", "_").replace("\\", "_").replace(".", "_")
            if sub_module_name in imported_sub_modules:
                    continue
            imported_sub_modules.add(sub_module_name)
            
            
            __M_locals_builtin_stored = __M_locals_builtin()
            __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['sub_module_name','sub_module_path'] if __M_key in __M_locals_builtin_stored]))
            __M_writer('\t')
            __M_writer(str(sub_module_name))
            __M_writer(' "')
            __M_writer(str(service_go_module))
            __M_writer('/')
            __M_writer(str(sub_module_path))
            __M_writer('"\r\n')
        __M_writer(')\r\n\r\nfunc RegisterLobbyClientService(\r\n\trd cd.DispatcherImpl, findSessionFn uc.FindCSMessageSession,\r\n) error {\r\n\tsvc := sp.File_')
        __M_writer(str( service.file.get_name().replace("/", "_").replace("\\", "_").replace(".", "_") ))
        __M_writer('.Services().ByName("')
        __M_writer(str(service.get_name()))
        __M_writer('")\r\n\tif svc == nil {\r\n\t\trd.GetApp().GetDefaultLogger().Error("lobbysvr_app.RegisterLobbyClientService no service ')
        __M_writer(str(service.get_full_name()))
        __M_writer('")\r\n\t\treturn fmt.Errorf("no service ')
        __M_writer(str(service.get_full_name()))
        __M_writer('")\r\n\t}\r\n\r\n')
        for rpc in rpcs.values():
            pass
            if not rpc.get_request_descriptor().full_name == "google.protobuf.Empty":

                sub_module_path = 'logic/' + rpc.get_extension_field("rpc_options", lambda x: x.module_name, "action")
                sub_module_name = sub_module_path.replace("/", "_").replace("\\", "_").replace(".", "_")
                
                
                __M_locals_builtin_stored = __M_locals_builtin()
                __M_locals.update(__M_dict_builtin([(__M_key, __M_locals_builtin_stored[__M_key]) for __M_key in ['sub_module_name','sub_module_path'] if __M_key in __M_locals_builtin_stored]))
                __M_writer('\tuc.RegisterCSMessageAction(\r\n\t\trd, findSessionFn, svc, "')
                __M_writer(str( rpc.get_full_name() ))
                __M_writer('",\r\n\t\tfunc(base cd.TaskActionCSBase[*sp.')
                __M_writer(str( rpc.get_request().get_name() ))
                __M_writer(', *sp.')
                __M_writer(str( rpc.get_response().get_name() ))
                __M_writer(']) cd.TaskActionImpl {\r\n\t\t\treturn &')
                __M_writer(str(sub_module_name))
                __M_writer('.TaskAction')
                __M_writer(str( rpc.get_identify_name(rpc.get_name(), PbConvertRule.CONVERT_NAME_CAMEL_CAMEL) ))
                __M_writer('{TaskActionCSBase: base}\r\n\t\t},\r\n\t)\r\n')
        __M_writer('\r\n\treturn nil\r\n}\r\n')
        return ''
    finally:
        context.caller_stack._pop_frame()


"""
__M_BEGIN_METADATA
{"filename": "D:/workspace/git/github/atsf4g-go/src/template/handle_cs_rpc.go.mako", "uri": "handle_cs_rpc.go.mako", "source_encoding": "utf-8", "line_map": {"16": 2, "17": 3, "18": 4, "19": 5, "20": 6, "21": 7, "22": 0, "36": 6, "37": 7, "38": 8, "39": 9, "40": 10, "43": 9, "44": 9, "45": 9, "46": 10, "47": 10, "48": 10, "49": 10, "50": 12, "51": 12, "52": 12, "53": 20, "54": 20, "55": 22, "56": 23, "57": 24, "58": 25, "59": 26, "60": 27, "61": 28, "62": 29, "63": 30, "64": 31, "65": 32, "68": 32, "69": 32, "70": 32, "71": 32, "72": 32, "73": 32, "74": 32, "75": 34, "76": 39, "77": 39, "78": 39, "79": 39, "80": 41, "81": 41, "82": 42, "83": 42, "84": 45, "86": 46, "87": 47, "88": 48, "89": 49, "90": 50, "91": 51, "94": 51, "95": 52, "96": 52, "97": 53, "98": 53, "99": 53, "100": 53, "101": 54, "102": 54, "103": 54, "104": 54, "105": 59, "111": 105}}
__M_END_METADATA
"""
