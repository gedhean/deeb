{{/*
Expand the name of the chart.
*/}}
{{- define "deeb.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "deeb.fullname" -}}
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
{{- define "deeb.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "deeb.labels" -}}
helm.sh/chart: {{ include "deeb.chart" . }}
{{ include "deeb.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: deeb
{{- end }}

{{/*
Selector labels
*/}}
{{- define "deeb.selectorLabels" -}}
app.kubernetes.io/name: {{ include "deeb.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Backend labels
*/}}
{{- define "deeb.backend.labels" -}}
helm.sh/chart: {{ include "deeb.chart" . }}
{{ include "deeb.backend.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: deeb
{{- end }}

{{/*
Backend selector labels
*/}}
{{- define "deeb.backend.selectorLabels" -}}
app.kubernetes.io/name: {{ include "deeb.name" . }}-backend
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Frontend labels
*/}}
{{- define "deeb.frontend.labels" -}}
helm.sh/chart: {{ include "deeb.chart" . }}
{{ include "deeb.frontend.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: deeb
{{- end }}

{{/*
Frontend selector labels
*/}}
{{- define "deeb.frontend.selectorLabels" -}}
app.kubernetes.io/name: {{ include "deeb.name" . }}-frontend
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: frontend
{{- end }}

{{/*
Create the name of the service account to use for backend
*/}}
{{- define "deeb.backend.serviceAccountName" -}}
{{- if .Values.backend.serviceAccount.create }}
{{- default (printf "%s-backend" (include "deeb.fullname" .)) .Values.backend.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.backend.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use for frontend
*/}}
{{- define "deeb.frontend.serviceAccountName" -}}
{{- if .Values.frontend.serviceAccount.create }}
{{- default (printf "%s-frontend" (include "deeb.fullname" .)) .Values.frontend.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.frontend.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Backend image
*/}}
{{- define "deeb.backend.image" -}}
{{- $tag := .Values.backend.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.backend.image.repository $tag }}
{{- end }}

{{/*
Frontend image
*/}}
{{- define "deeb.frontend.image" -}}
{{- $tag := .Values.frontend.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.frontend.image.repository $tag }}
{{- end }}

{{/*
Backend service name
*/}}
{{- define "deeb.backend.serviceName" -}}
{{- printf "%s-backend" (include "deeb.fullname" .) }}
{{- end }}

{{/*
Frontend service name
*/}}
{{- define "deeb.frontend.serviceName" -}}
{{- printf "%s-frontend" (include "deeb.fullname" .) }}
{{- end }}

{{/*
PostgreSQL service name
*/}}
{{- define "deeb.postgresql.serviceName" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "%s-postgresql" .Release.Name }}
{{- else }}
{{- .Values.postgresql.externalHost }}
{{- end }}
{{- end }}

{{/*
API URL for frontend
*/}}
{{- define "deeb.apiUrl" -}}
{{- if .Values.frontend.env.VITE_API_BASE_URL }}
{{- .Values.frontend.env.VITE_API_BASE_URL }}
{{- else }}
{{- printf "https://%s" .Values.global.domain | trimSuffix "/" }}
{{- end }}
{{- end }}

{{/*
Callback URL for OAuth
*/}}
{{- define "deeb.callbackUrl" -}}
{{- printf "https://%s/api/v1/auth/google/callback" .Values.global.domain }}
{{- end }}

{{/*
Frontend URL
*/}}
{{- define "deeb.frontendUrl" -}}
{{- printf "https://%s" .Values.global.domain }}
{{- end }}

{{/*
CORS origins
*/}}
{{- define "deeb.corsOrigins" -}}
{{- list (include "deeb.frontendUrl" .) | toJson }}
{{- end }}
