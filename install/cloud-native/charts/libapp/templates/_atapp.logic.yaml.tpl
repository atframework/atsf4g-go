{{- define "atapp.logic.yaml" -}}
# =========== logic configure ===========
logic:
  server:
    log_path: "{{ .Values.server_log_dir }}"
  excel:
    enable: true
    bindir: "../../resource/excel"
  user:
    enable_session_actor_log: {{ .Values.enable_session_actor_log }}
{{- end }}
