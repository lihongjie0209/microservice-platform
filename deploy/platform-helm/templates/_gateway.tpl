{{/* Resolve the externally visible host without coupling it to a Git branch name. */}}
{{- define "platform.gatewayHostname" -}}
{{- $gateway := .Values.gateway | default dict -}}
{{- if get $gateway "hostname" -}}
{{- get $gateway "hostname" -}}
{{- else -}}
{{- $baseDomain := required "gateway.baseDomain is required when gateway.enabled=true" (get $gateway "baseDomain") -}}
{{- if get $gateway "production" -}}
{{- printf "%s.%s" .Values.name $baseDomain -}}
{{- else -}}
{{- $environmentLabel := required "gateway.environmentLabel is required outside production" (get $gateway "environmentLabel") -}}
{{- printf "%s.%s.%s" .Values.name $environmentLabel $baseDomain -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create this resource only when a service explicitly opts into external exposure.
The APISIX controller resolves the Kubernetes Service to EndpointSlices and keeps
the upstream current as Pods are added, removed, or marked unready.
*/}}
{{- define "platform.apisixRoute" -}}
{{- $gateway := .Values.gateway | default dict -}}
{{- if get $gateway "enabled" }}
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: {{ .Values.name }}
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "platform.labels" . | nindent 4 }}
spec:
  ingressClassName: {{ get $gateway "ingressClassName" | default "apisix" | quote }}
  http:
    - name: {{ .Values.name }}
      match:
        hosts:
          - {{ include "platform.gatewayHostname" . | quote }}
        paths:
          {{- get $gateway "paths" | default (list "/*") | toYaml | nindent 10 }}
        methods:
          {{- get $gateway "methods" | default (list "POST") | toYaml | nindent 10 }}
      backends:
        - serviceName: {{ .Values.name }}
          servicePort: {{ get $gateway "servicePort" | default "http" }}
          resolveGranularity: endpoints
{{- end }}
{{- end }}
