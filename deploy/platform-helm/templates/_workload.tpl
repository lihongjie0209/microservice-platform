{{- define "platform.deployment" -}}
{{- $migration := .Values.migration | default dict -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.name }}
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "platform.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicas | default 2 }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ .Values.name }}
  strategy:
    type: RollingUpdate
    rollingUpdate: {maxUnavailable: 0, maxSurge: 1}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ .Values.name }}
    spec:
      serviceAccountName: {{ .Values.name }}
      automountServiceAccountToken: false
      terminationGracePeriodSeconds: 30
      securityContext:
        {{- include "platform.podSecurityContext" . | nindent 8 }}
      {{- if and .Values.database.enabled (eq (get $migration "mode" | default "initContainer") "initContainer") }}
      initContainers:
        - name: migrate
          image: {{ required "image.repository is required" .Values.image.repository }}:{{ required "image.tag is required" .Values.image.tag }}
          imagePullPolicy: IfNotPresent
          command: ["/app/migrate"]
          args: ["-env", {{ .Values.environment | quote }}, "-direction", "up"]
          envFrom:
            - configMapRef: {name: {{ .Values.name }}}
            - secretRef: {name: {{ .Values.name }}}
          resources:
            requests: {cpu: 50m, memory: 64Mi}
            limits: {cpu: 500m, memory: 256Mi}
          securityContext:
            {{- include "platform.containerSecurityContext" . | nindent 12 }}
          volumeMounts:
            - name: logs
              mountPath: /app/logs
      {{- end }}
      containers:
        - name: api
          image: {{ required "image.repository is required" .Values.image.repository }}:{{ required "image.tag is required" .Values.image.tag }}
          imagePullPolicy: IfNotPresent
          envFrom:
            - configMapRef: {name: {{ .Values.name }}}
            - secretRef: {name: {{ .Values.name }}}
          ports:
            - {name: http, containerPort: 8080}
            - {name: grpc, containerPort: 9090}
          startupProbe: {httpGet: {path: /live, port: http}, periodSeconds: 2, failureThreshold: 30}
          livenessProbe: {httpGet: {path: /live, port: http}, periodSeconds: 10}
          readinessProbe: {httpGet: {path: /ready, port: http}, periodSeconds: 5, timeoutSeconds: 3}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          securityContext:
            {{- include "platform.containerSecurityContext" . | nindent 12 }}
          volumeMounts:
            - name: logs
              mountPath: /app/logs
      volumes:
        - name: logs
          emptyDir: {}
{{- end }}

{{- define "platform.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ .Values.name }}
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "platform.labels" . | nindent 4 }}
    {{- if .Values.swaggerDiscovery.enabled }}
    platform.swagger/enabled: "true"
    {{- end }}
  {{- if .Values.swaggerDiscovery.enabled }}
  annotations:
    platform.swagger/enabled: "true"
    platform.swagger/port: http
    platform.swagger/path: {{ .Values.swaggerDiscovery.path | quote }}
    platform.swagger/title: {{ .Values.swaggerDiscovery.title | default .Values.name | quote }}
  {{- end }}
spec:
  selector: {app.kubernetes.io/name: {{ .Values.name }}}
  ports:
    - {name: http, port: 8080, targetPort: http}
    - {name: grpc, port: 9090, targetPort: grpc}
{{- end }}

{{- define "platform.migrationJob" -}}
{{- if .Values.database.enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .Values.name }}-migrate-{{ .Release.Revision }}
  namespace: {{ .Values.namespace }}
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      automountServiceAccountToken: false
      securityContext:
        {{- include "platform.podSecurityContext" . | nindent 8 }}
      containers:
        - name: migrate
          image: {{ .Values.image.repository }}:{{ .Values.image.tag }}
          command: ["/app/migrate"]
          args: ["-env", {{ .Values.environment | quote }}, "-direction", "up"]
          envFrom:
            - configMapRef: {name: {{ .Values.name }}}
            - secretRef: {name: {{ .Values.name }}}
          securityContext:
            {{- include "platform.containerSecurityContext" . | nindent 12 }}
          volumeMounts:
            - name: logs
              mountPath: /app/logs
      volumes:
        - name: logs
          emptyDir: {}
{{- end }}
{{- end }}
