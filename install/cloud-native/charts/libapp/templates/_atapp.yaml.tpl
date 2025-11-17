{{- define "atapp.yaml" -}}
{{- $bus_addr := include "libapp.busAddr" . -}}
{{- $uniq_id := .Values.uniq_id -}}
atapp:
  # =========== bus configure ===========
  id: {{ $bus_addr | quote }}
  app_id: {{ $uniq_id }}
  world_id: {{ .Values.world_id }}
  zone_id: {{ .Values.zone_id }}
  type_id: {{ required ".Values.type_id who entry required!" .Values.type_id }} # server type id
  type_name: {{ .Values.type_name | default (include "libapp.name" .) }}         # server type name
  area:
    zone_id: {{ include "libapp.logicID" . }} # svr_zone_id
{{ include "atapp.default.metadata.yaml" . | indent 4 }}
  remove_pidfile_after_exit: false     # keep pid file after exited
  {{- with .Values.inner_ip }}
  hostname: "{{ . }}"   # hostname, any host should has a unique name. if empty, we wil try to use the mac address
  {{- end }}
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
              number: {{ .Values.log_rotate_num }}
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
              number: {{ .Values.log_rotate_num }}
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
              number: {{ .Values.log_rotate_num }}
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
              number: {{ .Values.log_rotate_num }}
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
    initialize_timeout: 30s          # 20s for initialization
{{- end }}