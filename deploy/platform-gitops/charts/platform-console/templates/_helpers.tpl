{{- define "platform-console.labels" -}}
app.kubernetes.io/name: {{ .Values.name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
platform.environment: {{ .Values.environment }}
{{- end -}}

{{- define "platform-console.environmentDomain" -}}
{{- $base := required "gateway.baseDomain is required" .Values.gateway.baseDomain -}}
{{- if .Values.gateway.production -}}
{{- $base -}}
{{- else -}}
{{- printf "%s.%s" (required "gateway.environmentLabel is required outside production" .Values.gateway.environmentLabel) $base -}}
{{- end -}}
{{- end -}}

{{- define "platform-console.serviceURL" -}}
{{- printf "https://%s.%s" .service (include "platform-console.environmentDomain" .root) -}}
{{- end -}}
