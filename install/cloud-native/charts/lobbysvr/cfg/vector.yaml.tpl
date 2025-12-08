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
      . |= parse_regex!(.file_path, r'(?:^|[\\/])lobbysvr[\\/]log[\\/](?P<index_date>\d{4}-\d{2}-\d{2})[\\/](?P<uid>\d+)\.new\.log$')
      # Normalize index_date to YYYY.MM.DD for consistent indexing
      .index_date = replace(.index_date, "-", ".")
      del(.file)
      del(.file_path)
      del(.host)
      del(.timestamp)
      del(.source_type)

      .@timestamp = now()
      .log_type = "actor"

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
      del(.timestamp)
      del(.source_type)

      .@timestamp = now()
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
      del(.timestamp)
      del(.source_type)

      .@timestamp = now()
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
      del(.timestamp)
      del(.source_type)

      .@timestamp = now()
      .log_type = "redis"

sinks:
  # OpenSearch sinks (Elasticsearch-compatible) for indexing
  # Configure endpoint and credentials via environment variables:
  #   VECTOR_OPENSEARCH_ENDPOINT, VECTOR_OPENSEARCH_USERNAME, VECTOR_OPENSEARCH_PASSWORD
  opensearch_normal:
    type: elasticsearch
    mode: data_stream
    inputs:
      - normal_enrich
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
      dataset: "lobbysvr"
      namespace: "normal"

  opensearch_db_inner:
    type: elasticsearch
    mode: data_stream
    inputs:
      - db_inner_enrich
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
      dataset: "lobbysvr"
      namespace: "db-inner"

  opensearch_redis:
    type: elasticsearch
    mode: data_stream
    inputs:
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
      dataset: "lobbysvr"
      namespace: "redis"

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
      dataset: "lobbysvr"
      namespace: "actor"