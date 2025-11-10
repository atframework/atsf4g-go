{{- define "atapp.yaml" -}}
{{- $bus_addr := .Values.bus_addr -}}
atapp:
  # =========== bus configure ===========
  id: {{ $bus_addr | quote }}
  world_id: {{ .Values.world_id }}
  zone_id: {{ .Values.zone_id }}
  type_id: {{ required ".Values.type_id who entry required!" .Values.type_id }} # server type id
  type_name: {{ .Values.type_name | default (include "libapp.name" .) }}         # server type name
  area:
    zone_id: {{ int .Values.logic_id }} # svr_zone_id
  metadata:
{{- if hasKey .Values.service_discovery.selector (include "libapp.discovery_name" .) }}
    {{- $discovery_selector := (get .Values.service_discovery.selector (include "libapp.discovery_name" .)) }}
    {{- $metadata_discovery_yaml_content := (include "atapp.discovery.metadata.yaml" $discovery_selector) }}
    {{- $metadata_default_yaml_content := (include "atapp.default.metadata.yaml" .) }}
{{ include "libapp.util.merge_yaml" (list $metadata_default_yaml_content $metadata_discovery_yaml_content) | indent 4 }}
{{- else }}
{{ include "atapp.default.metadata.yaml" . | indent 4 }}
{{- end }}
  remove_pidfile_after_exit: false     # keep pid file after exited
  {{- with .Values.inner_ip }}
  hostname: "{{ . }}"   # hostname, any host should has a unique name. if empty, we wil try to use the mac address
  {{- end }}
  resource_path: ../../resource          # resource directory
  log_path: "{{ .Values.server_log_dir }}"
  bus:
    listen: unix:///run/atapp/{{ .Values.atapp.deployment.project_name }}/{{ include "libapp.name" . }}_{{ $bus_addr }}.sock
    # bus.subnets: 0/0
    # proxy:                           # atgateway must has parent node
    loop_times: {{ .Values.atapp.bus_loop_times_per_tick | default 2048 }}                    # max message number in one loop
    ttl: {{ .Values.atapp.bus_ttl | default 16 }}                            # max ttl when transfer messages
    backlog: {{ .Values.atapp.backlog | default 256 }}                       # tcp backlog
    overwrite_listen_path: false       # overwrite the existing unix socket
    first_idle_timeout: 30s            # first idle timeout when have new connection(second)
    ping_interval: 8s                  # ping interval(second)
    retry_interval: 3s                 # retry interval when error happen(second)
    fault_tolerant: 2                  # how many errors at most to ignore, or it will kill the connection
    msg_size: 256KB                    # max message size(256KB)
    recv_buffer_size: {{ default "2MB" .Values.atapp.bus_recv_buff_size }} # recv channel size(2MB), will be used to initialize (shared) memory channel size
    send_buffer_size: {{ default "1MB" .Values.atapp.bus_send_buff_size }} # send buffer size, will be used to initialize io_stream channel write queue
    send_buffer_number: 0              # send message number limit, will be used to initialize io_stream channel write queue, 0 for dynamic buffer
    gateways:
      address: tbuspp://{{ include "libapp.name" . }}_{{ $bus_addr }}
  worker_pool:
    {{- toYaml .Values.atapp.worker_pool | trim | nindent 4  }}
  # =========== upper configures can not be reload ===========
  # =========== log configure ===========
  log:
    level: {{ .Values.log_level }}            # log active level(disable/disabled, fatal, error, warn/warning, info, notice, debug)
    category:
      - name: "default"
        index: 0
        prefix: "[Log %L][%F %T.%f][%s:%n(%C)]: "
{{- if or (eq .Values.log_stacktrace_level "disable") (eq .Values.log_stacktrace_level "disabled") }}
        stacktrace:
          min: disable
          max: disable
{{- else }}
        stacktrace:
          min: {{ .Values.log_stacktrace_level }}
          max: fatal
{{- end }}
        sink:
          # default error log for file
          - type: file
            level:
              min: fatal
              max: warning
            rotate:
              number: 10
              size: 20MB
            path: "{{ .Values.server_log_dir }}"
            file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.error.log"
            auto_flush: error
            flush_interval: 1s    # flush log interval
            hard_link: true
          - type: file
            level:
              min: fatal
              max: debug
            rotate:
              number: 10
              size: 20MB
            path: "{{ .Values.server_log_dir }}"
            file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.all.log"
            auto_flush: error
            flush_interval: 1s    # flush log interval
            hard_link: true
      - name: redis
        index: 1
        prefix: "[%F %T.%f]: "
        stacktrace:
          min: disable
          max: disable
        sink:
          - type: file
            level:
              min: fatal
              max: debug
            rotate:
              number: 10
              size: 10MB
            path: "{{ .Values.server_log_dir }}"
            file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.redis.all.log"
            auto_flush: error
            flush_interval: 1s        # flush log interval
          - type: file
            level:
              min: fatal
              max: warning
            rotate:
              number: 10
              size: 10MB
            path: "{{ .Values.server_log_dir }}"
            file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.redis.error.log"
            auto_flush: error
            flush_interval: 1s        # flush log interval
      - name: db_inner
        index: 2
        prefix: "[%F %T.%f]: "
        stacktrace:
          min: disable
          max: disable
        sink:
          - type: file
            level:
              min: fatal
              max: debug
            rotate:
              number: 10
              size: 10MB
            path: "{{ .Values.server_log_dir }}"
            file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.db_inner.info.log"
            auto_flush: error
            flush_interval: 1s    # flush log interval
  # =========== timer ===========
  timer:
    tick_interval: 8ms               # 8ms for tick active
    tick_round_timeout: 128ms
    stop_timeout: 30s                # 20s for stop operation
    stop_interval: 256ms
    message_timeout: 8s
    initialize_timeout: 30s          # 20s for initialization
    reserve_interval_min: 200us
    reserve_interval_max: 1s
    reserve_permille: 10
  # =========== etcd service for discovery ===========
  etcd:
    {{- if .Values.etcd }}
    enable: {{ .Values.etcd.enabled }}
    {{- else }}
    enable: false
    {{- end }}
{{- /* etcd module enabled */}}
{{- if and .Values.etcd .Values.etcd.enabled }}
{{- if .Values.etcd.log.enable }}
    log:
      startup_level: {{ .Values.etcd.log.startup_level }}
      level: {{ .Values.etcd.log.level }}
      category:
        - name: etcd_default
          prefix: "[Log %L][%F %T.%f][%s:%n(%C)]: " # log categorize 0's name = etcd_default
          stacktrace:
            min: disable
            max: disable
          sink:
            - type: file
              level:
                min: fatal
                max: trace
              rotate:
                number: 10
                size: 10485760 # 10MB
              path: "{{ .Values.server_log_dir }}"
              file: "{{ include "libapp.name" . }}_{{ $bus_addr }}.etcd.log"
              auto_flush: info
              flush_interval: 1s # 1s (unit: s,m,h,d)
{{- end -}} {{- /* end if */}}
{{- if .Values.etcd.client_urls}}
    hosts:
{{- range $_, $host := .Values.etcd.client_urls }}
      - {{ $host }}
{{- end }} {{- /* end range client_urls */ -}}
{{- else }}
{{- $node_url_list := list -}}
{{- range $idx, $node := .Values.etcd_deploy_nodes }}
{{- $port := add $.etcd.etcd_listen_client_base_port $node.Index }}
{{- $url := printf "http://%s:%d" $node.InnerIP $port }}
      - {{ $url }}
{{- end }} {{- /* end range etcd_deploy_nodes */ -}}
{{- end -}} {{- /* end if */}}
{{- if .Values.partition }}
    path: {{ .Values.etcd.path }}/{{ include "libapp.environment" . }}/{{ .Values.partition }}
{{- else }}
    path: {{ .Values.etcd.path }}
{{- end }}
{{- if empty .Values.etcd.authorization }}
    # authorization:  # username:password
{{- else }}
    authorization: "{{ .Values.etcd.authorization }}"
{{- end }}
    # http:
    #   debug: false
    #   user_agent: ""
    #   proxy: ""
    #   no_proxy: ""
    #   proxy_user_name: ""
    #   proxy_password: ""
    ssl:
      enable_alpn: {{ .Values.etcd.ssl.enable_alpn }}
      verify_peer: {{ .Values.etcd.ssl.verify_peer }}
      ssl_min_version: {{ .Values.etcd.ssl.ssl_min_version }}
      ssl_client_cert: "../../resource/ssl/{{ .Values.etcd.ssl.ssl_client_cert_file }}"
      ssl_client_key: "../../resource/ssl/{{ .Values.etcd.ssl.ssl_client_key_file }}"
{{- if empty .Values.etcd.ssl.ssl_client_key_passwd }}
      # ssl_client_key_passwd:
{{- else }}
      ssl_client_key_passwd: {{ .Values.etcd.ssl.ssl_client_key_passwd }}
{{- end }}
      ssl_ca_cert: ../../resource/ssl/{{ .Values.etcd.ssl.ssl_ca_cert_file }}
{{- if empty .Values.etcd.ssl.ssl_cipher_list }}
      # ssl_cipher_list:
{{- else }}
      ssl_cipher_list: "{{ .Values.etcd.ssl.ssl_cipher_list }}"
{{- end }}
{{- if empty .Values.etcd.ssl.ssl_cipher_list_tls13 }}
      # ssl_cipher_list_tls13:
{{- else }}
      ssl_cipher_list_tls13: "{{ .Values.etcd.ssl.ssl_cipher_list_tls13 }}"
{{- end }}
    cluster:
      auto_update: {{ .Values.etcd.cluster.auto_update }}       # set false when etcd service is behind a safe cluster(Kubernetes etc.)
      update_interval: {{ .Values.etcd.cluster.update_interval }}       # update etcd cluster members interval
      retry_interval: {{ .Values.etcd.cluster.retry_interval }}       # update etcd cluster retry interval
    keepalive:
      timeout: {{ .Values.etcd.keepalive.timeout }}            # expired timeout
      ttl: {{ .Values.etcd.keepalive.ttl }}                # renew ttl interval
      retry_interval: {{ .Values.etcd.keepalive.retry_interval }} # keepalive retry interval
    request:
      timeout: {{ .Values.etcd.request.timeout }}             # timeout for etcd request
      initialization_timeout: {{ .Values.etcd.request.initialization_timeout }} # timeout for etcd request when initializing
      connect_timeout: {{ .Values.etcd.request.connect_timeout }} # timeout for etcd request connect
      dns_cache_timeout: {{ .Values.etcd.request.dns_cache_timeout }} # timeout for dns cache of etcd request
      dns_servers: "{{ .Values.etcd.request.dns_servers }}" # dns servers: 8.8.8.8:53,1.1.1.1
    init:
      timeout: {{ .Values.etcd.init.timeout }}                  # initialize timeout
      tick_interval: 256ms
    watcher:
      retry_interval: {{ .Values.etcd.watcher.retry_interval }}       # retry interval watch when previous request failed
      request_timeout: {{ .Values.etcd.watcher.request_timeout }}       # request timeout for watching
      get_request_timeout: {{ .Values.etcd.watcher.get_request_timeout }}            # range request timeout for watcher
      startup_random_delay_min: {{ .Values.etcd.watcher.startup_random_delay_min }}  # delay start watching - min
      startup_random_delay_max: {{ .Values.etcd.watcher.startup_random_delay_max }}  # delay start watching - max
      by_id: {{ .Values.etcd.watcher.by_id }}       # watch service discovery by id
      by_name: {{ .Values.etcd.watcher.by_name }}       # watch service discovery by name
      # by_type_id: []
      # by_type_name: []
      # by_tag: []
    report_alive:
      by_id: {{ .Values.etcd.report_alive.by_id }}
      by_type: {{ .Values.etcd.report_alive.by_type }}
      by_name: {{ .Values.etcd.report_alive.by_name }}
{{- if not (empty .Values.etcd.report_alive.by_tag) }}
      by_tag:
{{- range $_, $tag := .Values.etcd.report_alive.by_tag }}
        - {{ $tag }}
{{- end }} {{- /* end range .Values.etcd.report_alive.by_tag */}}
{{- end }} {{- /* end if not (empty .Values.etcd.report_alive.by_tag) */}}
{{- end }} {{- /* end if .Values.etcd */}}
{{- end }}