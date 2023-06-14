{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "stackstate.secret.name" -}}
{{- default "steadybit-extension-stackstate" .Values.stackstate.existingSecret -}}
{{- end -}}
