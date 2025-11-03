- global:
    overwrite: true
    input: "{{ .project_template_dir }}/db_interface.go.mako"
    output: 'local_db.go'
    output_directory: "{{ .project_current_configure_dir }}"
    custom_variables:
        generate_proto_file: "protocol/pbdesc/svr.local.table.proto"