{{ include "atapp.yaml" . }}
{{ include "atapp.logic.yaml" . }}

lobbysvr:
{{- if and .Values.redis .Values.redis.enable }}
  redis:
    addr: {{ .Values.redis.addr }}
    password: {{ .Values.redis.password }}
    pool_size: {{ .Values.redis.pool_size }}
    record_prefix: {{ .Values.redis.record_prefix }}
    random_prefix: {{ .Values.redis.random_prefix }}
{{- end -}} {{- /* end if */}}
  webserver:
    port: {{ .Values.webserver.port }}
  websocket:
    {{- $atapp := (default (dict) .Values.atapp) -}}
    {{- $deploy := (default (dict) $atapp.deployment) -}}
    {{- $env := (default "" $deploy.deployment_environment) -}}
    {{- $ws := (default (dict) .Values.websocket) -}}
    {{- $wspath := (default "" $ws.path) -}}
    {{- if $wspath }}
    path: {{ printf "/%s/%s/ws/v1" $env $wspath | replace "///" "/" | replace "//" "/" | quote }}
    {{- else }}
    path: "/ws/v1"
    {{- end }}
