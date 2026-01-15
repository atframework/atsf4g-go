- service:
    name: "{{ .project_namespace }}.LobbyClientService"
    overwrite: true
    output_directory: "{{ .project_current_configure_dir }}"
    service_template:
      - overwrite: true
        input: "{{ .project_current_configure_dir }}/template/robot_rpc_handle.go.mako"
        output: "protocol/rpc_handle.go"