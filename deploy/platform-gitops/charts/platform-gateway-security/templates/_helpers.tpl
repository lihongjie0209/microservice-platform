{{- define "platform-gateway-security.environmentDomain" -}}
{{- $baseDomain := required "baseDomain is required when enabled=true" .Values.baseDomain -}}
{{- if .Values.production -}}
{{- $baseDomain -}}
{{- else -}}
{{- $environmentLabel := required "environmentLabel is required outside production" .Values.environmentLabel -}}
{{- printf "%s.%s" $environmentLabel $baseDomain -}}
{{- end -}}
{{- end -}}
