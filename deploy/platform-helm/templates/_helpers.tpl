{{- define "platform.labels" -}}
app.kubernetes.io/name: {{ required "name is required" .Values.name }}
app.kubernetes.io/managed-by: Helm
{{- with .Values.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "platform.podSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{- define "platform.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop: ["ALL"]
{{- end }}
