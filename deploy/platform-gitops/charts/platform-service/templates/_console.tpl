{{- define "platform-service.consoleOrigin" -}}
{{- $baseDomain := required "gateway.baseDomain is required when gateway.enabled=true" .Values.gateway.baseDomain -}}
{{- if .Values.gateway.production -}}
{{- printf "https://console.%s" $baseDomain -}}
{{- else -}}
{{- $environmentLabel := required "gateway.environmentLabel is required outside production" .Values.gateway.environmentLabel -}}
{{- printf "https://console.%s.%s" $environmentLabel $baseDomain -}}
{{- end -}}
{{- end -}}
