{{/*
=======================================================================
Clario360 Platform — Helm Template Helpers
=======================================================================
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "clario360.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this
(by the DNS naming spec). If release name contains chart name it will be used
as a full name.
*/}}
{{- define "clario360.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "clario360.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels for the chart.
*/}}
{{- define "clario360.labels" -}}
helm.sh/chart: {{ include "clario360.chart" . }}
{{ include "clario360.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: clario360
{{- end }}

{{/*
Selector labels for the chart.
*/}}
{{- define "clario360.selectorLabels" -}}
app.kubernetes.io/name: {{ include "clario360.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
=======================================================================
Component helpers — accept a dict with .root (top-level context) and
.component (string name like "api-gateway", "iam-service", etc.)
=======================================================================
*/}}

{{/*
Component fully-qualified name: {{ fullname }}-{{ component }}
Usage: {{ include "clario360.componentName" (dict "root" . "component" "api-gateway") }}
*/}}
{{- define "clario360.componentName" -}}
{{- printf "%s-%s" (include "clario360.fullname" .root) .component | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Component labels — standard labels plus the component label.
Usage: {{ include "clario360.componentLabels" (dict "root" . "component" "api-gateway") }}
*/}}
{{- define "clario360.componentLabels" -}}
helm.sh/chart: {{ include "clario360.chart" .root }}
{{ include "clario360.componentSelectorLabels" (dict "root" .root "component" .component) }}
app.kubernetes.io/version: {{ .root.Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
app.kubernetes.io/part-of: clario360
{{- end }}

{{/*
Component selector labels — for matchLabels in Deployments / Services.
Usage: {{ include "clario360.componentSelectorLabels" (dict "root" . "component" "api-gateway") }}
*/}}
{{- define "clario360.componentSelectorLabels" -}}
app.kubernetes.io/name: {{ include "clario360.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
=======================================================================
Image helpers
=======================================================================
*/}}

{{/*
Return the image tag for a component. Falls back to Chart.AppVersion.
Usage: {{ include "clario360.imageTag" (dict "imageConfig" .Values.apiGateway.image "root" .) }}
*/}}
{{- define "clario360.imageTag" -}}
{{- if .imageConfig }}
{{- default .root.Chart.AppVersion .imageConfig.tag }}
{{- else }}
{{- .root.Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Build a fully-qualified image reference: registry/repository:tag
Usage: {{ include "clario360.image" (dict "imageConfig" .Values.apiGateway.image "root" . "defaultRepo" "clario360/api-gateway") }}
*/}}
{{- define "clario360.image" -}}
{{- $registry := "" }}
{{- $repository := .defaultRepo }}
{{- $tag := .root.Chart.AppVersion }}
{{- if .imageConfig }}
  {{- if .imageConfig.registry }}
    {{- $registry = .imageConfig.registry }}
  {{- end }}
  {{- if .imageConfig.repository }}
    {{- $repository = .imageConfig.repository }}
  {{- end }}
  {{- if .imageConfig.tag }}
    {{- $tag = .imageConfig.tag }}
  {{- end }}
{{- end }}
{{- if .root.Values.global }}
  {{- if .root.Values.global.imageRegistry }}
    {{- if not .imageConfig }}
      {{- $registry = .root.Values.global.imageRegistry }}
    {{- else if not .imageConfig.registry }}
      {{- $registry = .root.Values.global.imageRegistry }}
    {{- end }}
  {{- end }}
{{- end }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{/*
=======================================================================
Service Account helper
=======================================================================
*/}}

{{/*
Return the service account name for a component.
If the component has serviceAccount.name set, use that.
If serviceAccount.create is false and no name, use "default".
Otherwise use the component name.
Usage: {{ include "clario360.serviceAccountName" (dict "root" . "component" "api-gateway" "svcConfig" .Values.apiGateway) }}
*/}}
{{- define "clario360.serviceAccountName" -}}
{{- if .svcConfig }}
  {{- if .svcConfig.serviceAccount }}
    {{- if .svcConfig.serviceAccount.name }}
      {{- .svcConfig.serviceAccount.name }}
    {{- else if hasKey .svcConfig.serviceAccount "create" }}
      {{- if .svcConfig.serviceAccount.create }}
        {{- include "clario360.componentName" (dict "root" .root "component" .component) }}
      {{- else }}
        {{- "default" }}
      {{- end }}
    {{- else }}
      {{- include "clario360.componentName" (dict "root" .root "component" .component) }}
    {{- end }}
  {{- else }}
    {{- include "clario360.componentName" (dict "root" .root "component" .component) }}
  {{- end }}
{{- else }}
  {{- include "clario360.componentName" (dict "root" .root "component" .component) }}
{{- end }}
{{- end }}

{{/*
=======================================================================
Security Contexts
=======================================================================
*/}}

{{/*
Standard pod-level security context — non-root, seccomp enabled.
Usage:
  securityContext:
    {{- include "clario360.podSecurityContext" . | nindent 4 }}
*/}}
{{- define "clario360.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 65534
runAsGroup: 65534
fsGroup: 65534
fsGroupChangePolicy: OnRootMismatch
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/*
Standard container-level security context — drop all capabilities, read-only root.
Usage:
  securityContext:
    {{- include "clario360.containerSecurityContext" . | nindent 4 }}
*/}}
{{- define "clario360.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
runAsNonRoot: true
runAsUser: 65534
capabilities:
  drop:
    - ALL
{{- end }}

{{/*
=======================================================================
Health Probes
=======================================================================
*/}}

{{/*
Standard liveness, readiness, and startup probes for Go services.
Port name is passed as the argument.
Usage:
  {{- include "clario360.healthProbes" (dict "portName" "http" "healthPath" "/healthz" "readyPath" "/readyz") | nindent 8 }}
*/}}
{{- define "clario360.healthProbes" -}}
{{- $portName := default "http" .portName -}}
{{- $healthPath := default "/healthz" .healthPath -}}
{{- $readyPath := default "/readyz" .readyPath -}}
livenessProbe:
  httpGet:
    path: {{ $healthPath }}
    port: {{ $portName }}
  initialDelaySeconds: 15
  periodSeconds: 20
  timeoutSeconds: 5
  failureThreshold: 3
  successThreshold: 1
readinessProbe:
  httpGet:
    path: {{ $readyPath }}
    port: {{ $portName }}
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
  successThreshold: 1
startupProbe:
  httpGet:
    path: {{ $healthPath }}
    port: {{ $portName }}
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 30
  successThreshold: 1
{{- end }}

{{/*
=======================================================================
Prometheus Annotations
=======================================================================
*/}}

{{/*
Standard Prometheus scrape annotations.
Usage:
  annotations:
    {{- include "clario360.prometheusAnnotations" (dict "metricsPort" "9090") | nindent 4 }}
*/}}
{{- define "clario360.prometheusAnnotations" -}}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .metricsPort | quote }}
prometheus.io/path: "/metrics"
{{- end }}

{{/*
=======================================================================
Infrastructure URL builders
=======================================================================
*/}}

{{/*
Build a PostgreSQL connection URL.
Usage: {{ include "clario360.databaseUrl" (dict "root" . "dbName" "iam") }}
*/}}
{{- define "clario360.databaseUrl" -}}
{{- $host := default "localhost" .root.Values.global.database.host -}}
{{- $port := default "5432" (.root.Values.global.database.port | toString) -}}
{{- $sslMode := default "require" .root.Values.global.database.sslMode -}}
{{- printf "postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@%s:%s/%s?sslmode=%s" $host $port .dbName $sslMode -}}
{{- end }}

{{/*
Build a Redis connection URL.
Usage: {{ include "clario360.redisUrl" . }}
*/}}
{{- define "clario360.redisUrl" -}}
{{- $host := default "localhost" .Values.global.redis.host -}}
{{- $port := default "6379" (.Values.global.redis.port | toString) -}}
{{- $db := default "0" (.Values.global.redis.db | toString) -}}
{{- printf "redis://:$(REDIS_PASSWORD)@%s:%s/%s" $host $port $db -}}
{{- end }}

{{/*
=======================================================================
Namespace helper
=======================================================================
*/}}

{{/*
Return the namespace to deploy into.
*/}}
{{- define "clario360.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace }}
{{- end }}
