{{ include "atapp.yaml" . }}
lobbysvr:
{{- if and .Values.redis .Values.redis.enable }}
  redis:
    addr: {{ .Values.redis.addr }}
    password: {{ .Values.redis.password }}
    pool_size: {{ .Values.redis.pool_size }}
{{- end -}} {{- /* end if */}}