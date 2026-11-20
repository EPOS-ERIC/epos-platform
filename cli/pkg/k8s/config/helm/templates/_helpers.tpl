{{- define "epos.name" -}}
{{- if .Values.nameOverride -}}
{{- .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "epos.ingressHost" -}}
{{- if hasKey .Values "ingress_host" -}}
{{- .Values.ingress_host -}}
{{- else -}}
{{- .Values.domain -}}
{{- end -}}
{{- end -}}

{{- define "epos.labels" -}}
app.kubernetes.io/name: {{ include "epos.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.Version }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end }}
{{- end -}}

{{- define "epos.selectorLabels" -}}
app.kubernetes.io/name: {{ include "epos.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "epos.image_pull_secrets" -}}
{{- if .Values.image_pull_secrets.enabled }}
imagePullSecrets:
  - name: epos-registry-secret
{{- end }}
{{- end -}}

{{- define "epos.defaultAAIAuthRootURL" -}}
{{- if .Values.url_prefix_namespace -}}
{{- printf "%s://%s/%s/aai" .Values.protocol .Values.domain .Release.Namespace -}}
{{- else -}}
{{- printf "%s://%s/aai" .Values.protocol .Values.domain -}}
{{- end -}}
{{- end -}}

{{- define "epos.backofficeBasePath" -}}
{{- if .Values.url_prefix_namespace -}}
{{- printf "/%s%s" .Release.Namespace .Values.components.backoffice.gui.base_url -}}
{{- else -}}
{{- .Values.components.backoffice.gui.base_url -}}
{{- end -}}
{{- end -}}

{{- define "epos.backofficeURL" -}}
{{- printf "%s://%s%s" .Values.protocol .Values.domain (include "epos.backofficeBasePath" .) -}}
{{- end -}}

{{- define "epos.platformBasePath" -}}
{{- if .Values.url_prefix_namespace -}}
{{- printf "/%s%s" .Release.Namespace .Values.components.platform_ui.base_url -}}
{{- else -}}
{{- .Values.components.platform_ui.base_url -}}
{{- end -}}
{{- end -}}

{{- define "epos.platformURL" -}}
{{- printf "%s://%s%s" .Values.protocol .Values.domain (include "epos.platformBasePath" .) -}}
{{- end -}}

{{- define "epos.aaiAuthRootURL" -}}
{{- $endpoint := .Values.components.gateway.aai.service_endpoint | default "" | trim | trimSuffix "/" | trimSuffix "/oauth2/userinfo" -}}
{{- if $endpoint -}}
{{- $endpoint -}}
{{- else -}}
{{- include "epos.defaultAAIAuthRootURL" . -}}
{{- end -}}
{{- end -}}

{{- define "epos.gatewayAAIServiceEndpoint" -}}
{{- if .Values.components.gateway.aai.enabled -}}
{{- $gatewayEndpoint := .Values.components.gateway.aai.gateway_endpoint | default "" | trim -}}
{{- if $gatewayEndpoint -}}
{{- $gatewayEndpoint -}}
{{- else -}}
{{- if .Values.components.aai_service.enabled -}}
http://aai-service:8080/oauth2/userinfo
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
