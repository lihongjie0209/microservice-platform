{{- define "platform.serviceAccount" -}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.name }}
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "platform.labels" . | nindent 4 }}
automountServiceAccountToken: false
{{- end }}

{{- define "platform.podDisruptionBudget" -}}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata: {name: {{ .Values.name }}, namespace: {{ .Values.namespace }}}
spec:
  minAvailable: {{ .Values.pdb.minAvailable | default 1 }}
  selector: {matchLabels: {app.kubernetes.io/name: {{ .Values.name }}}}
{{- end }}

{{- define "platform.horizontalPodAutoscaler" -}}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata: {name: {{ .Values.name }}, namespace: {{ .Values.namespace }}}
spec:
  scaleTargetRef: {apiVersion: apps/v1, kind: Deployment, name: {{ .Values.name }}}
  minReplicas: {{ .Values.autoscaling.minReplicas | default 2 }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas | default 10 }}
  behavior: {scaleDown: {stabilizationWindowSeconds: 300}}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target: {type: Utilization, averageUtilization: {{ .Values.autoscaling.targetCPUUtilization | default 70 }}}
{{- end }}

{{- define "platform.networkPolicy" -}}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: {{ .Values.name }}, namespace: {{ .Values.namespace }}}
spec:
  podSelector: {matchLabels: {app.kubernetes.io/name: {{ .Values.name }}}}
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: {{ .Values.namespace }}}}
        {{- with .Values.networkPolicy.gatewayNamespace }}
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: {{ . }}}}
        {{- end }}
      ports: [{port: 8080}, {port: 9090}]
  egress:
    {{- toYaml .Values.networkPolicy.egress | nindent 4 }}
{{- end }}

{{- define "platform.serviceMonitor" -}}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata: {name: {{ .Values.name }}, namespace: {{ .Values.namespace }}}
spec:
  selector: {matchLabels: {app.kubernetes.io/name: {{ .Values.name }}}}
  endpoints: [{port: http, path: /metrics, interval: 30s}]
{{- end }}

{{- define "platform.externalSecret" -}}
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata: {name: {{ .Values.name }}, namespace: {{ .Values.namespace }}}
spec:
  refreshInterval: {{ .Values.externalSecret.refreshInterval | default "1h" }}
  secretStoreRef: {name: {{ required "externalSecret.store is required" .Values.externalSecret.store }}, kind: ClusterSecretStore}
  target: {name: {{ .Values.name }}, creationPolicy: Owner}
  dataFrom:
    - extract: {key: {{ required "externalSecret.key is required" .Values.externalSecret.key }}}
{{- end }}
