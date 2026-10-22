{{- define "url-shortener.name" -}}
url-shortener
{{- end -}}

{{- define "url-shortener.labels" -}}
app.kubernetes.io/name: {{ include "url-shortener.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "url-shortener.selectorLabels" -}}
app.kubernetes.io/name: {{ include "url-shortener.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "url-shortener.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}
{{- end -}}
