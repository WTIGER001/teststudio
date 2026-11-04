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
{{- if .Values.workers.enabled -}}
{{- if or (eq .Values.workers.namespace .Release.Namespace) (eq .Values.workers.namespace "") -}}{{ fail "Workers require a separate dedicated namespace" }}{{- end -}}
{{- if or (eq .Values.workers.origin .Values.origins.ui) (eq .Values.workers.origin .Values.origins.artifacts) -}}{{ fail "Workers require a separate private HTTPS origin" }}{{- end -}}
{{- if eq .Values.workers.privateIngress.className .Values.ingress.className -}}{{ fail "Worker ingress must use a separate private ingress class" }}{{- end -}}
{{- if not .Values.networkPolicy.enabled -}}{{ fail "Workers require ingress network policy" }}{{- end -}}
{{- $registry := .Values.workers.registry -}}
{{- if or (and $registry.bearerEnvironment (or $registry.usernameEnvironment $registry.passwordEnvironment)) (and $registry.usernameEnvironment (not $registry.passwordEnvironment)) (and $registry.passwordEnvironment (not $registry.usernameEnvironment)) -}}{{ fail "Registry authentication requires a Basic pair or one Bearer reference" }}{{- end -}}
{{- range $name := list $registry.usernameEnvironment $registry.passwordEnvironment $registry.bearerEnvironment -}}
{{- if and $name (not (hasKey $registry.credentials $name)) -}}{{ fail "Registry credential references require an existing Secret mapping" }}{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- define "teststudio.oidcReferences" -}}
{{- $references := list -}}
{{- range $name, $_ := .Values.oidc.credentials -}}
{{- $references = append $references (printf "env:%s" $name) -}}
{{- end -}}
{{- join "," $references -}}
{{- end -}}

{{- define "teststudio.workerConfiguration" -}}
{{- $w := .Values.workers -}}
{{- $worker := dict "apiOrigin" $w.apiOrigin "namespace" $w.namespace "serviceAccount" $w.serviceAccount "tokenFile" "/var/run/teststudio-controller/token" "caFile" "/var/run/teststudio-controller/ca.crt" "architecture" .Values.image.architecture "runtimeClass" $w.runtimeClass "policyName" $w.policyName "exchangeOrigin" $w.origin "exchangePeer" (dict "namespace" $w.privateIngress.namespace "labels" $w.privateIngress.podLabels) "exchangePort" $w.privateIngress.port "dnsPeer" (dict "namespace" $w.dns.namespace "labels" $w.dns.podLabels) "imagePullSecrets" $w.imagePullSecrets "nodeSelector" $w.nodeSelector -}}
{{- $r := $w.registry -}}
{{- $registry := dict "origin" $r.origin "repositoryPrefix" $r.repositoryPrefix "tokenURL" $r.tokenURL "tokenService" $r.tokenService "blobRedirectOrigins" $r.blobRedirectOrigins "usernameEnvironment" $r.usernameEnvironment "passwordEnvironment" $r.passwordEnvironment "bearerEnvironment" $r.bearerEnvironment -}}
{{- if $r.caSecret -}}{{- $_ := set $registry "caFile" "/etc/teststudio-registry/ca.crt" -}}{{- end -}}
{{- dict "worker" $worker "registry" $registry | toPrettyJson -}}
{{- end -}}
