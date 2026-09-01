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

{{/* Apply the same edge protections to every externally exposed route group. */}}
{{- define "platform.apisixSecurityPlugins" -}}
{{- $gateway := .Values.gateway | default dict -}}
{{- $security := get $gateway "security" | default dict -}}
{{- $rateLimit := get $security "rateLimit" | default dict -}}
{{- $securityHeaders := get $security "headers" | default (dict "X-Content-Type-Options" "nosniff" "X-Frame-Options" "DENY" "Referrer-Policy" "no-referrer" "Strict-Transport-Security" "max-age=31536000; includeSubDomains") -}}
{{- if ne (toString (get $security "enabled")) "false" }}
- name: client-control
  enable: true
  config:
    max_body_size: {{ get $security "maxBodySizeMB" | default 1 }}
- name: limit-req
  enable: true
  config:
    key: remote_addr
    key_type: var
    rate: {{ get $rateLimit "rate" | default 100 }}
    burst: {{ get $rateLimit "burst" | default 50 }}
    rejected_code: 429
- name: response-rewrite
  enable: true
  config:
    headers:
      set:
        {{- $securityHeaders | toYaml | nindent 8 }}
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
    - name: {{ .Values.name }}-api
      match:
        hosts:
          - {{ include "platform.gatewayHostname" . | quote }}
        paths:
          {{- get $gateway "paths" | default (list "/api/*") | toYaml | nindent 10 }}
        methods:
          {{- get $gateway "methods" | default (list "POST" "OPTIONS") | toYaml | nindent 10 }}
      {{- $plugins := include "platform.apisixSecurityPlugins" . }}
      {{- if $plugins }}
      plugins:
        {{- $plugins | nindent 8 }}
      {{- end }}
      backends:
        - serviceName: {{ .Values.name }}
          servicePort: {{ get $gateway "servicePort" | default "http" }}
          resolveGranularity: endpoints
    {{- $standardGetPaths := get $gateway "standardGetPaths" | default (list "/swagger/*" "/swagger-assets/*" "/.well-known/*") }}
    {{- if $standardGetPaths }}
    - name: {{ .Values.name }}-standards
      match:
        hosts:
          - {{ include "platform.gatewayHostname" . | quote }}
        paths:
          {{- $standardGetPaths | toYaml | nindent 10 }}
        methods:
          - GET
          - HEAD
      {{- if $plugins }}
      plugins:
        {{- $plugins | nindent 8 }}
      {{- end }}
      backends:
        - serviceName: {{ .Values.name }}
          servicePort: {{ get $gateway "servicePort" | default "http" }}
          resolveGranularity: endpoints
    {{- end }}
{{- end }}
{{- end }}
