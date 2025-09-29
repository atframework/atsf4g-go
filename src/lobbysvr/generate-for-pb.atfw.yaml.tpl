- service:
    name: "{{ .project_namespace }}.LobbyClientService"
    overwrite: false
    output_directory: "{{ .project_current_configure_dir }}"
    custom_variables:
      "service_go_package_prefix": "lobbysvr_"
      "protocol_go_module": "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
      "service_go_module": "github.com/atframework/atsf4g-go/service-lobbysvr"
{{- with .custom_variables }}
    {{- range $key, $value := .custom_variables }}
      "{{ $key }}" : "{{ $value }}"
    {{- end }}
{{- end }}
    service_template:
      - overwrite: true
        input: "{{ .project_template_dir }}/handle_cs_rpc.go.mako"
        output: "app/register_${ service.get_name_lower_rule() }.go"
    rpc_template:
      - overwrite: false
        input: "{{ .project_template_dir }}/task_action_cs_rpc.go.mako"
        output: 'logic/${rpc.get_extension_field("rpc_options", lambda x: x.module_name, "action")}/task_action_${rpc.get_name()}.go'
    # rpc_include: ""
    # rpc_exclude: ""
    # rpc_include_request: [] # include request types for rpc template
    rpc_exclude_request: # exclude request types for rpc template
      - "google.protobuf.Empty"
- service:
    name: "{{ .project_namespace }}.LobbyClientService"
    overwrite: false
    output_directory: "{{ .project_current_configure_dir }}"
    custom_variables:
      "service_go_package_prefix": "lobbysvr_"
      "protocol_go_module": "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/pbdesc"
      "service_go_module": "github.com/atframework/atsf4g-go/service-lobbysvr"
{{- with .custom_variables }}
    {{- range $key, $value := .custom_variables }}
      "{{ $key }}" : "{{ $value }}"
    {{- end }}
{{- end }}
    service_template:
      - overwrite: true
        input: "{{ .project_template_dir }}/session_downstream_api_for_cs.go.mako"
        output: "rpc/${service.get_name_lower_rule()}/session_downstream_api.go"
