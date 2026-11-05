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
{{- if .Values.controller.enabled -}}
{{- if not .Values.workers.enabled -}}{{ fail "Separate execution controllers require the Kubernetes worker profile" }}{{- end -}}
{{- if and (eq .Values.artifactStore.driver "filesystem") (ne .Values.persistence.accessMode "ReadWriteMany") -}}{{ fail "Separate execution controllers require shared ReadWriteMany evidence or an object store" }}{{- end -}}
{{- range $name, $_ := .Values.controller.podLabels -}}
{{- if has $name (list "app.kubernetes.io/name" "app.kubernetes.io/instance" "app.kubernetes.io/component") -}}{{ fail "Controller labels cannot replace its selector" }}{{- end -}}
{{- end -}}
{{- end -}}
{{- if .Values.workers.enabled -}}
{{- if or (eq .Values.workers.namespace .Release.Namespace) (eq .Values.workers.namespace "") -}}{{ fail "Workers require a separate dedicated namespace" }}{{- end -}}
{{- if or (eq .Values.workers.origin .Values.origins.ui) (eq .Values.workers.origin .Values.origins.artifacts) -}}{{ fail "Workers require a separate private HTTPS origin" }}{{- end -}}
{{- if and .Values.workers.privateIngress.enabled (eq .Values.workers.privateIngress.className .Values.ingress.className) -}}{{ fail "Worker ingress must use a separate private ingress class" }}{{- end -}}
{{- if not .Values.networkPolicy.enabled -}}{{ fail "Workers require ingress network policy" }}{{- end -}}
{{- include "teststudio.registryAuthentication" .Values.workers.registry -}}
{{- end -}}
{{- if .Values.preparation.enabled -}}
{{- if not .Values.workers.enabled -}}{{ fail "Preparation requires the protected Kubernetes worker profile" }}{{- end -}}
{{- if and (gt (int .Values.preparation.replicas) 1) (ne .Values.preparation.persistence.accessMode "ReadWriteMany") -}}{{ fail "Multiple preparation replicas require shared ReadWriteMany journals" }}{{- end -}}
{{- $journals := .Values.preparation.persistence.existingClaim | default (printf "%s-journals" (include "teststudio.preparationName" .)) -}}
{{- $artifacts := .Values.persistence.existingClaim | default (printf "%s-artifacts" (include "teststudio.name" .)) -}}
{{- if and (eq .Values.artifactStore.driver "filesystem") (eq $journals $artifacts) -}}{{ fail "Preparation journals require a separate private claim" }}{{- end -}}
{{- if and .Values.preparation.acceptNewWork (not (or .Values.preparation.goRuntime .Values.preparation.playwrightRuntime)) -}}{{ fail "New preparation requires at least one immutable registry runtime" }}{{- end -}}
{{- $prefix := printf "%s/%s" (trimPrefix "https://" .Values.workers.registry.origin) .Values.workers.registry.repositoryPrefix -}}
{{- range $image := list .Values.preparation.goRuntime .Values.preparation.playwrightRuntime -}}
{{- if and $image (not (or (hasPrefix (printf "%s/" $prefix) $image) (hasPrefix (printf "%s@" $prefix) $image))) -}}{{ fail "Preparation runtimes must use the configured registry namespace" }}{{- end -}}
{{- end -}}
{{- $registry := .Values.preparation.registry -}}
{{- if not (or $registry.usernameEnvironment $registry.bearerEnvironment) -}}{{ fail "Preparation requires separate registry writer credential references" }}{{- end -}}
{{- include "teststudio.registryAuthentication" $registry -}}
{{- $selected := dict -}}
{{- range $name := list $registry.usernameEnvironment $registry.passwordEnvironment $registry.bearerEnvironment -}}
{{- if $name -}}{{- $_ := set $selected $name true -}}{{- end -}}
{{- end -}}
{{- range $name, $reference := $registry.credentials -}}
{{- if not (hasKey $selected $name) -}}{{ fail "Preparation cannot inject unused registry credentials" }}{{- end -}}
{{- range $_, $reader := $.Values.workers.registry.credentials -}}
{{- if eq $reference.existingSecret $reader.existingSecret -}}{{ fail "Preparation writer and API reader must use separate Secrets" }}{{- end -}}
{{- end -}}
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
{{- include "teststudio.workerSettings" (dict "root" . "registry" .Values.workers.registry) -}}
{{- end -}}
{{- define "teststudio.preparationConfiguration" -}}
{{- $registry := mergeOverwrite (deepCopy .Values.workers.registry) .Values.preparation.registry -}}
{{- include "teststudio.workerSettings" (dict "root" . "registry" $registry) -}}
{{- end -}}
{{- define "teststudio.workerSettings" -}}
{{- $w := .root.Values.workers -}}
{{- $worker := dict "apiOrigin" $w.apiOrigin "namespace" $w.namespace "serviceAccount" $w.serviceAccount "tokenFile" "/var/run/teststudio-controller/token" "caFile" "/var/run/teststudio-controller/ca.crt" "architecture" .root.Values.image.architecture "runtimeClass" $w.runtimeClass "policyName" $w.policyName "exchangeOrigin" $w.origin "exchangePeer" (dict "namespace" $w.privateIngress.namespace "labels" $w.privateIngress.podLabels) "exchangePort" $w.privateIngress.port "dnsPeer" (dict "namespace" $w.dns.namespace "labels" $w.dns.podLabels) "imagePullSecrets" $w.imagePullSecrets "nodeSelector" $w.nodeSelector -}}
{{- $r := .registry -}}
{{- $registry := dict "origin" $r.origin "repositoryPrefix" $r.repositoryPrefix "tokenURL" $r.tokenURL "tokenService" $r.tokenService "blobRedirectOrigins" $r.blobRedirectOrigins "usernameEnvironment" $r.usernameEnvironment "passwordEnvironment" $r.passwordEnvironment "bearerEnvironment" $r.bearerEnvironment -}}
{{- if $r.caSecret -}}{{- $_ := set $registry "caFile" "/etc/teststudio-registry/ca.crt" -}}{{- end -}}
{{- dict "worker" $worker "registry" $registry | toPrettyJson -}}
{{- end -}}
{{- define "teststudio.registryAuthentication" -}}
{{- if or (and .bearerEnvironment (or .usernameEnvironment .passwordEnvironment)) (and .usernameEnvironment (not .passwordEnvironment)) (and .passwordEnvironment (not .usernameEnvironment)) -}}{{ fail "Registry authentication requires a Basic pair or one Bearer reference" }}{{- end -}}
{{- range $name := list .usernameEnvironment .passwordEnvironment .bearerEnvironment -}}
{{- if and $name (not (hasKey $.credentials $name)) -}}{{ fail "Registry credential references require an existing Secret mapping" }}{{- end -}}
{{- end -}}
{{- end -}}
{{- define "teststudio.preparationSelector" -}}
app.kubernetes.io/name: teststudio
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/component: preparation
{{- end -}}
{{- define "teststudio.preparationName" -}}
{{- if gt (len .Release.Name) 44 -}}
{{- printf "%s-%s-prepare" (.Release.Name | trunc 35 | trimSuffix "-") (.Release.Name | sha256sum | trunc 8) -}}
{{- else -}}
{{- printf "%s-prepare" .Release.Name -}}
{{- end -}}
{{- end -}}

{{- define "teststudio.controllerName" -}}
{{- if gt (len .Release.Name) 41 -}}
{{- printf "%s-%s-controller" (.Release.Name | trunc 30 | trimSuffix "-") (.Release.Name | sha256sum | trunc 8) -}}
{{- else -}}
{{- printf "%s-controller" .Release.Name -}}
{{- end -}}
{{- end -}}
{{- define "teststudio.controllerSelector" -}}
app.kubernetes.io/name: teststudio
app.kubernetes.io/instance: {{ .Release.Name | quote }}
app.kubernetes.io/component: controller
{{- end -}}
{{- define "teststudio.artifactConfiguration" -}}
{{- if eq .Values.artifactStore.driver "filesystem" -}}
{{- dict "driver" "filesystem" "directory" "/data/artifacts" | toPrettyJson -}}
{{- else -}}
{{- .Values.artifactStore | toPrettyJson -}}
{{- end -}}
{{- end -}}
{{- define "teststudio.controllerConfiguration" -}}
{{- dict "artifact-store.json" (include "teststudio.artifactConfiguration" .) "kubernetes-workers.json" (include "teststudio.workerConfiguration" .) | toPrettyJson -}}
{{- end -}}
