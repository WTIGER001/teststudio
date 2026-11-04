{{- define "teststudio.name" -}}
{{- .Release.Name -}}
{{- end -}}
{{- define "teststudio.selector" -}}
app.kubernetes.io/name: teststudio
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/component: api
{{- end -}}
{{- define "teststudio.labels" -}}
{{ include "teststudio.selector" . }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}
{{- define "teststudio.validate" -}}
{{- range $name, $_ := .Values.podLabels -}}
{{- if has $name (list "app.kubernetes.io/name" "app.kubernetes.io/instance" "app.kubernetes.io/component") -}}{{ fail "Pod labels cannot replace the service selector" }}{{- end -}}
{{- end -}}
{{- if hasKey .Values.podAnnotations "checksum/config" -}}{{ fail "Pod annotations cannot replace the configuration checksum" }}{{- end -}}
{{- if eq .Values.origins.ui .Values.origins.artifacts -}}{{ fail "API/UI and artifact origins must use different HTTPS hosts" }}{{- end -}}
{{- $configured := false -}}
{{- $ids := dict -}}
{{- range .Values.oidc.providers -}}
{{- if hasKey $ids .id -}}{{ fail "OIDC provider identifiers must be unique" }}{{- end -}}
{{- $_ := set $ids .id true -}}
{{- if eq .issuer $.Values.bootstrap.issuer -}}{{- $configured = true -}}{{- end -}}
{{- if .secretRef -}}
{{- $name := trimPrefix "env:" .secretRef -}}
{{- if not (hasKey $.Values.oidc.credentials $name) -}}{{ fail "Every OIDC secretRef requires an explicit existing Secret reference" }}{{- end -}}
{{- end -}}
{{- end -}}
{{- if not $configured -}}{{ fail "Bootstrap issuer must match an initially configured OIDC provider" }}{{- end -}}
{{- if and (eq .Values.artifactStore.driver "filesystem") (gt (int .Values.replicas) 1) (ne .Values.persistence.accessMode "ReadWriteMany") -}}{{ fail "Multiple filesystem replicas require a shared ReadWriteMany artifact claim" }}{{- end -}}
{{- end -}}
{{- define "teststudio.oidcReferences" -}}
{{- $references := list -}}
{{- range $name, $_ := .Values.oidc.credentials -}}
{{- $references = append $references (printf "env:%s" $name) -}}
{{- end -}}
{{- join "," $references -}}
{{- end -}}
