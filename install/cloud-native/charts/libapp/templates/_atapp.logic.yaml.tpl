{{- define "atapp.logic.yaml" -}}
# =========== logic configure ===========
logic:
  world_id: {{ .Values.world_id }} # world_id
  zone_id: {{ .Values.zone_id }} # zone_id
  logic_id: {{ include "libapp.logicID" . }} # svr_zone_id
  server:
    log_path: "{{ .Values.server_log_dir }}"
  excel:
    enable: true
    bindir: "../../resource/excel"
  user:
    enable_session_actor_log: {{ .Values.enable_session_actor_log }}
{{- end }}
