sources:
  lobbysvr_logs_normal:
    type: file
    include:
      - {{ .Values.volumeMounts.logMountPath }}/*.normal.all.log
    data_dir: {{ .Values.volumeMounts.logMountPath }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_db_inner:
    type: file
    include:
      - {{ .Values.volumeMounts.logMountPath }}/*.db_inner.all.log
    data_dir: {{ .Values.volumeMounts.logMountPath }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_redis:
    type: file
    include:
      - {{ .Values.volumeMounts.logMountPath }}/*.redis.all.log
    data_dir: {{ .Values.volumeMounts.logMountPath }}
    ignore_older_secs: 600
    read_from: beginning
    line_delimiter: "\u001e\n"
    glob_minimum_cooldown_ms: 1000

  lobbysvr_logs_actor:
    type: file
    include:
      - {{ .Values.volumeMounts.logMountPath }}/*/*.new.log
    data_dir: {{ .Values.volumeMounts.logMountPath }}
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
      .@timestamp = format_timestamp!(parsed_ts, format: "%FT%T.%9fZ", timezone: "UTC")
      del(.file)
      del(.file_path)
      del(.host)
      del(.log_ts)
      del(.source_type)
      del(.timestamp)

  normal_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_normal
    source: |
      .file_path = string!(.file)
      .file_name = basename!(.file_path)
      . |= parse_regex!(.file_name, r'^(?P<svrname>[A-Za-z0-9_-]+)_(?P<inst_id>\d+.\d+.\d+.\d+)')
      del(.file)
      del(.file_path)
      del(.file_name)
      del(.host)
      del(.source_type)
      del(.timestamp)
      # Extract timestamp and log level from message prefix like "[2025-12-08 14:44:22.949][ INFO]..."
      . |= parse_regex!(.message, r'^\[(?P<log_ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]\[\s*(?P<log_level>[A-Z]+)\s*\]\((?P<caller>[^():]+:\d+)\)')
      parsed_ts = parse_timestamp!(.log_ts, "%Y-%m-%d %H:%M:%S%.f")
      .@timestamp = format_timestamp!(parsed_ts, format: "%FT%T.%9fZ", timezone: "UTC")
      del(.log_ts)

      # kv_matches = parse_regex_all!(.message, r'\u001F(?P<key>[^=\u001F\s]+)=(?P<value>[^\u001F]+)\u001F')
      # if kv_matches != null && length(kv_matches) > 0 {
      #   for_each(kv_matches) -> |_index, value| {
      #     key, err = "$" + value.key
      #     . = set!(., [key], value.value)
      #   }
      # }

      .log_type = "normal"

  db_inner_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_db_inner
    source: |
      .file_path = string!(.file)
      .file_name = basename!(.file_path)
      . |= parse_regex!(.file_name, r'^(?P<svrname>[A-Za-z0-9_-]+)_(?P<inst_id>\d+.\d+.\d+.\d+)')
      del(.file)
      del(.file_path)
      del(.file_name)
      del(.host)
      del(.source_type)
      del(.timestamp)
      # Extract timestamp and log level from message prefix like "[2025-12-08 14:44:22.949][ INFO]..."
      . |= parse_regex!(.message, r'^\[(?P<log_ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]\[\s*(?P<log_level>[A-Z]+)\s*\]\((?P<caller>[^():]+:\d+)\)')
      parsed_ts = parse_timestamp!(.log_ts, "%Y-%m-%d %H:%M:%S%.f")
      .@timestamp = format_timestamp!(parsed_ts, format: "%FT%T.%9fZ", timezone: "UTC")
      del(.log_ts)

      # kv_matches = parse_regex_all!(.message, r'\u001F(?P<key>[^=\u001F\s]+)=(?P<value>[^\u001F]+)\u001F')
      # if kv_matches != null && length(kv_matches) > 0 {
      #   for_each(kv_matches) -> |_index, value| {
      #     key, err = "$" + value.key
      #     . = set!(., [key], value.value)
      #   }
      # }

      .log_type = "db_inner"

  redis_enrich:
    type: remap
    inputs:
      - lobbysvr_logs_redis
    source: |
      .file_path = string!(.file)
      .file_name = basename!(.file_path)
      . |= parse_regex!(.file_name, r'^(?P<svrname>[A-Za-z0-9_-]+)_(?P<inst_id>\d+.\d+.\d+.\d+)')
      del(.file)
      del(.file_path)
      del(.file_name)
      del(.host)
      del(.source_type)
      del(.timestamp)
      # Extract timestamp and log level from message prefix like "[2025-12-08 14:44:22.949][ INFO]..."
      . |= parse_regex!(.message, r'^\[(?P<log_ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]\[\s*(?P<log_level>[A-Z]+)\s*\]\((?P<caller>[^():]+:\d+)\)')
      parsed_ts = parse_timestamp!(.log_ts, "%Y-%m-%d %H:%M:%S%.f")
      .@timestamp = format_timestamp!(parsed_ts, format: "%FT%T.%9fZ", timezone: "UTC")
      del(.log_ts)

      # kv_matches = parse_regex_all!(.message, r'\u001F(?P<key>[^=\u001F\s]+)=(?P<value>[^\u001F]+)\u001F')
      # if kv_matches != null && length(kv_matches) > 0 {
      #   for_each(kv_matches) -> |_index, value| {
      #     key, err = "$" + value.key
      #     . = set!(., [key], value.value)
      #   }
      # }

      .log_type = "redis"

sinks:
  # out:
  #   type: console
  #   inputs:
  #     - actor_enrich
  #     - normal_enrich
  #     - db_inner_enrich
  #     - redis_enrich
  #   encoding:
  #     codec: json

  # OpenSearch sinks (Elasticsearch-compatible) for indexing
  # Configure endpoint and credentials via environment variables:
  #   VECTOR_OPENSEARCH_ENDPOINT, VECTOR_OPENSEARCH_USERNAME, VECTOR_OPENSEARCH_PASSWORD
  opensearch_log:
    type: elasticsearch
    mode: data_stream
    inputs:
      - normal_enrich
      - db_inner_enrich
      - redis_enrich
    endpoints:
      - {{ .Values.vector.opensearch.endpoint }}
    auth:
      strategy: basic
      user: {{ .Values.vector.opensearch.username }}
      password: {{ .Values.vector.opensearch.password }}
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
      - {{ .Values.vector.opensearch.endpoint }}
    auth:
      strategy: basic
      user: {{ .Values.vector.opensearch.username }}
      password: {{ .Values.vector.opensearch.password }}
    tls:
      verify_certificate: false
      verify_hostname: false
    data_stream:
      type: project-y
      dataset: log
      namespace: actor