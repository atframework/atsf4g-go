{{ include "atapp.yaml" . }}
lobbysvr:
{{- if and .Values.redis .Values.redis.enable }}
  redis:
    addr: {{ .Values.redis.addr }}
    password: {{ .Values.redis.password }}
    pool_size: {{ .Values.redis.pool_size }}
    record_prefix: {{ .Values.redis.record_prefix }}
    random_prefix: {{ .Values.redis.random_prefix }}
{{- end -}} {{- /* end if */}}