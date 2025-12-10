sources:
  lobbysvr_logs_normal:
    type: file
    include:
      - {{ .Values.vector.log_path }}/*.normal.all.log
    data_dir: {{ .Values.vector.log_path }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_db_inner:
    type: file
    include:
      - {{ .Values.vector.log_path }}/*.db_inner.all.log
    data_dir: {{ .Values.vector.log_path }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_redis:
    type: file
    include:
      - {{ .Values.vector.log_path }}/*.redis.all.log
    data_dir: {{ .Values.vector.log_path }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_actor:
    type: file
    include:
      - {{ .Values.vector.log_path }}/*/*.new.log
    data_dir: {{ .Values.vector.log_path }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

transforms:
  actor_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_actor
    source: |
      .file_path = string!(.file)
      . |= parse_regex!(.file_path, r'(?:^|[\\/])lobbysvr[\\/]log[\\/][^\\/]+[\\/](?P<uid>\d+)\.new\.log$')
      # Extract log timestamp from message content and normalize to RFC3339 with nanoseconds
      . |= parse_regex!(.message, r'(?P<log_ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})')
      parsed_ts = parse_timestamp!(.log_ts, "%Y-%m-%d %H:%M:%S%.f")
      .@timestamp = format_timestamp!(parsed_ts, format: "%FT%T.%9fZ", timezone: "local")
      del(.file)
      del(.file_path)
      del(.host)
      del(.log_ts)
      del(.source_type)
      del(.timestamp)
      .@timestamp = now()

  normal_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_normal
    source: |
{{ include "libapp.vector.server_log_parse" "normal" | indent 6 }}
{{ include "libapp.vector.server_log_index" . | indent 6 }}

  db_inner_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_db_inner
    source: |
{{ include "libapp.vector.server_log_parse" "db_inner" | indent 6 }}

  redis_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_redis
    source: |
{{ include "libapp.vector.server_log_parse" "redis" | indent 6 }}

sinks:
  {{- if .Values.vector.sliks.console.enable }}
  out:
    type: console
    inputs:
      - actor_enrich
      - normal_enrich
      - db_inner_enrich
      - redis_enrich
    encoding:
      codec: json
  {{- end}}

  {{- if .Values.vector.sliks.opensearch.enable }}
  opensearch_log:
    type: elasticsearch
    mode: data_stream
    inputs:
      - normal_enrich
      - db_inner_enrich
      - redis_enrich
    endpoints:
      - {{ .Values.vector.sliks.opensearch.endpoint }}
    auth:
      strategy: basic
      user: {{ .Values.vector.sliks.opensearch.username }}
      password: {{ .Values.vector.sliks.opensearch.password }}
    tls:
      verify_certificate: false
      verify_hostname: false
    data_stream:
      type: project-y
      dataset: log
      namespace: running

  opensearch_actor:
    type: elasticsearch
    mode: data_stream
    inputs:
      - actor_enrich
    endpoints:
      - {{ .Values.vector.sliks.opensearch.endpoint }}
    auth:
      strategy: basic
      user: {{ .Values.vector.sliks.opensearch.username }}
      password: {{ .Values.vector.sliks.opensearch.password }}
    tls:
      verify_certificate: false
      verify_hostname: false
    data_stream:
      type: project-y
      dataset: log
      namespace: actor
  {{- end -}}