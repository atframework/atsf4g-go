master-addr: {{ .Values.master_addr }}
redis-addr: {{ .Values.report_redis_addr }}
redis-pwd: {{ .Values.report_redis_password }}
cluster-mode: false

url: ws://localhost:{{ default 7001 .Values.server_port | int }}/ws/v1
connect-type: websocket
resource: ../../resource/excel

set:
  openIDPrefix: 1250000

dbtool-redis-addr:
{{- range $_, $addr := .Values.redis.addrs }}
  - {{ $addr }}
{{- end }}
dbtool-redis-password: {{ .Values.redis.password }}
dbtool-redis-cluster: {{ .Values.redis.cluster_mode }}
dbtool-random-prefix: {{ .Values.redis.record_prefix }}
dbtool-pb-file: {{ .Values.db_pb_path }}
dbtool-redis-version: {{ .Values.redis_version }}