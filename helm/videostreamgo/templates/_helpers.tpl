{{/*
Expand the name of the chart.
*/}}
{{- define "videostreamgo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "videostreamgo.fullname" -}}
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
{{- define "videostreamgo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "videostreamgo.labels" -}}
helm.sh/chart: {{ include "videostreamgo.chart" . }}
{{ include "videostreamgo.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: videostreamgo
{{- if .Values.global.imageRegistry }}
image.registry: {{ .Values.global.imageRegistry }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "videostreamgo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "videostreamgo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "videostreamgo.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "videostreamgo.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create environment variables for config
*/}}
{{- define "videostreamgo.configEnv" -}}
{{- range $key, $value := .Values.platform-api.config }}
- name: {{ $key | upper }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Create environment variables for secrets
*/}}
{{- define "videostreamgo.secretsEnv" -}}
{{- range $key, $value := .Values.platform-api.secrets }}
- name: {{ $key | upper }}
  valueFrom:
    secretKeyRef:
      name: {{ include "videostreamgo.fullname" . }}-secrets
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Image helper
*/}}
{{- define "videostreamgo.image" -}}
{{- $component := index .ctx.Values (.component) -}}
{{- $registry := default "" .ctx.Values.global.imageRegistry -}}
{{- $repository := printf "%s/%s" $registry $component.image.repository -}}
{{- $tag := default .ctx.Chart.AppVersion $component.image.tag -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end }}
