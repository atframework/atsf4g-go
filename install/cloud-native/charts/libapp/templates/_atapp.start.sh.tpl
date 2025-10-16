{{- define "atapp.start.sh" -}}
{{- $bus_addr := .Values.bus_addr -}}
{{- $proc_name := .Values.proc_name -}}
{{- $type_name := (.Values.type_name | default (include "libapp.name" .)) -}}

#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )";
SCRIPT_DIR="$( readlink -f $SCRIPT_DIR )";
cd "$SCRIPT_DIR";

./{{ $proc_name }} -config ../cfg/{{ $type_name }}_{{ $bus_addr }}.yaml -pid ./{{ $type_name }}_{{ $bus_addr }}.pid start
{{- end }}

{{- define "atapp.start.bat" -}}
{{- $bus_addr := .Values.bus_addr -}}
{{- $proc_name := .Values.proc_name -}}
{{- $type_name := (.Values.type_name | default (include "libapp.name" .)) -}}
@echo off

cd %cd%

.\{{ $proc_name }}.exe -config ..\cfg\{{ $type_name }}_{{ $bus_addr }}.yaml -pid .\{{ $type_name }}_{{ $bus_addr }}.pid start
{{- end }}
