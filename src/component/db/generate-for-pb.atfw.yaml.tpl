- global:
    overwrite: true
    input: "{{ .project_template_dir }}/local_db_interface.go.mako"
    output: 'local_db.go'
    output_directory: "{{ .project_current_configure_dir }}"
    custom_variables:
        generate_proto_file: "protocol/pbdesc/svr.local.table.proto"
- global:
    overwrite: true
    input: "{{ .project_template_dir }}/global_db_interface.go.mako"
    output: 'global_db.go'
    output_directory: "{{ .project_current_configure_dir }}"
    custom_variables:
        generate_proto_file: "protocol/pbdesc/svr.global.table.proto"