{{/*
Expand the name of the chart.
*/}}
{{- define "libapp.name" -}}
  {{- default (default .Chart.Name .Values.nameOverride) .Values.type_name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "libapp.fullname" -}}
  {{- if .Values.fullnameOverride }}
    {{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
  {{- else }}
    {{- $name := default .Chart.Name .Values.nameOverride }}
    {{- if contains $name .Release.Name }}
      {{- .Release.Name | trunc 63 | trimSuffix "-" }}
    {{- else }}
      {{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
    {{- end }}
  {{- end }}
{{- end }}

{{- define "libapp.discovery_name" -}}
  {{- default (include "libapp.name" .) .Values.discovery_name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "libapp.hpa_target_name" -}}
  {{- if not .Values.tcm_mode }}
    {{- include "libapp.fullname" . -}}
  {{- else }}
    {{- if not (empty .Values.atapp.deployment.deployment_environment) -}}
      {{ .Values.atapp.deployment.deployment_environment }}-{{- include "libapp.name" . -}}
    {{- else }}
      {{- include "libapp.name" . -}}
    {{- end }}
  {{- end }}
{{- end }}

{{- define "libapp.metrics.normalize_prometheus_name" -}}
  {{- if and .metrics_name .metrics_unit -}}
    {{- if hasSuffix .metrics_unit .metrics_name -}}
      {{ regexReplaceAll "[^a-zA-Z0-9_:]" .metrics_name "_" }}
    {{- else -}}
      {{ regexReplaceAll "[^a-zA-Z0-9_:]" .metrics_name "_" }}_{{ regexReplaceAll "[^a-zA-Z0-9_:]" .metrics_unit "_" }}
    {{- end -}}
  {{- else -}}
    {{ regexReplaceAll "[^a-zA-Z0-9_:]" .metrics_name "_" }}
  {{- end -}}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "libapp.chart" -}}
  {{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "libapp.labels" -}}
helm.sh/chart: {{ include "libapp.chart" . }}
{{ include "libapp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "libapp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "libapp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.matrix.io/service-type: gs
app.matrix.io/id: {{ int .Values.logic_id | quote }}
app.matrix.io/name: {{ include "libapp.name" . }}
app.matrix.io/environment: {{ include "libapp.environment" . }}
app.matrix.io/partition: {{ .Values.partition | quote }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "libapp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "libapp.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Libapp deploy environment
*/}}
{{- define "libapp.environment" -}}
{{- default "production" .Values.envOverride }}
{{- end }}

{{/*
Libapp deploy cluster
*/}}
{{- define "libapp.cluster" -}}
{{- default "local" .Values.cluster }}
{{- end }}