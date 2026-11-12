package config_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s/config"
)

const (
	externalAAIAuthRootURL      = "https://auth.example.com"
	externalAAIUserinfoEndpoint = externalAAIAuthRootURL + "/oauth2/userinfo"
	gatewayAAIUserinfoEndpoint  = "https://gateway-auth.example.com/oauth2/userinfo"
	gatewayAAIKeycloakUserinfo  = "https://login.envri.eu/auth/realms/envri/protocol/openid-connect/userinfo"
	dataportalTLSSecret         = "dataportal-tls"
	gatewayTLSSecret            = "gateway-tls"
	certManagerIssuerAnnotation = `cert-manager.io/cluster-issuer: "letsencrypt"`
)

func TestConfigRender(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(cfg *config.Config)
		wantFiles    []string
		wantContains map[string][]string
		notContains  map[string][]string
	}{
		{
			name: "default config renders current stack",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-default"
			},
			wantFiles: []string{
				"templates/dataportal.yaml",
				"templates/gateway.yaml",
				"templates/metadata-lifecycle.yaml",
				"templates/data-retriever.yaml",
				"templates/converter.yaml",
				"templates/sharing.yaml",
			},
			wantContains: map[string][]string{
				"templates/dataportal.yaml":         {"name: dataportal", `value: "http://localhost/test-default/aai"`},
				"templates/gateway.yaml":            {"name: gateway", `url: "http://aai-service:8080/oauth2/userinfo"`, `url: "converter:8200"`, "imagePullPolicy: IfNotPresent", `nginx.ingress.kubernetes.io/proxy-body-size: "10m"`},
				"templates/metadata-lifecycle.yaml": {"name: metadata-lifecycle", "name: metadata-database", "claimName: psqldata"},
				"templates/data-retriever.yaml":     {"name: data-retriever", "name: retriever-db"},
				"templates/converter.yaml":          {"name: converter", "name: converter-db", "claimName: converter-plugins", "type: Recreate", "fsGroup: 1000"},
				"templates/sharing.yaml":            {"name: sharing", "name: sharing-db"},
				"templates/pvc.yaml":                {"name: psqldata", "name: retriever-db", "name: converter-db", "name: converter-plugins", "name: sharing-db", "ReadWriteOnce"},
			},
		},
		{
			name: "external converter database",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-external-converter-db"
				cfg.Components.ConverterDatabase.Enabled = false
				cfg.Components.ConverterDatabase.Host = "external-db"
			},
			wantContains: map[string][]string{
				"templates/converter.yaml": {`value: "external-db"`, "name: converter"},
			},
			notContains: map[string][]string{
				"templates/converter.yaml": {"claimName: converter-db"},
				"templates/pvc.yaml":       {"name: converter-db"},
			},
		},
		{
			name: "image pull policy is configurable",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-image-policy"
				cfg.ImagePullPolicy = "Always"
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml":   {"imagePullPolicy: Always"},
				"templates/converter.yaml": {"imagePullPolicy: Always"},
			},
		},
		{
			name: "backoffice enabled renders UI",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-backoffice"
			},
			wantContains: map[string][]string{
				"templates/backoffice-ui.yaml": {"name: backoffice-ui", `value: "http://localhost/test-backoffice/aai"`},
			},
		},
		{
			name: "sharing database enabled renders sharing service",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-sharing"
				cfg.Components.SharingDatabase.Enabled = true
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {`url: "sharing:8200"`},
				"templates/sharing.yaml": {"name: sharing"},
			},
		},
		{
			name: "embedded aai enabled renders manifest and gateway settings",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-aai"
				cfg.Components.Gateway.AAI.Enabled = true
				cfg.Components.AAIService.Enabled = true
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {
					`url: "http://aai-service:8080/oauth2/userinfo"`,
				},
				"templates/dataportal.yaml": {
					`value: "http://localhost/test-aai/aai"`,
				},
				"templates/aai-service.yaml": {
					"name: aai-service",
					"securityContext:\n        fsGroup: 1000",
					"kind: Ingress",
					"path: /test-aai/aai(/|$)(.*)",
					`INITIAL_ADMIN_NAME: "EPOS"`,
					`INITIAL_ADMIN_SURNAME: "User"`,
					`INITIAL_ADMIN_EMAIL: "epos@epos.eu"`,
					"name: INITIAL_ADMIN_PASSWORD",
					`APP_CORS_ALLOW_ORIGIN: "*"`,
					`OAUTH_GUI_URLS: "http://localhost/test-aai/,http://localhost/test-aai/backoffice/"`,
				},
				"templates/pvc.yaml": {"name: aai"},
			},
		},
		{
			name: "embedded aai backoffice URL omits namespace when prefix is disabled",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-aai-no-prefix"
				cfg.URLPrefixNamespace = false
				cfg.Components.Gateway.AAI.Enabled = true
				cfg.Components.AAIService.Enabled = true
			},
			wantContains: map[string][]string{
				"templates/aai-service.yaml": {
					`OAUTH_GUI_URLS: "http://localhost/,http://localhost/backoffice/"`,
				},
			},
		},
		{
			name: "embedded aai disabled supports external endpoint and auth root without manifest",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-external-aai"
				cfg.Components.Gateway.AAI.Enabled = true
				cfg.Components.AAIService.Enabled = false
				cfg.Components.Gateway.AAI.ServiceEndpoint = externalAAIUserinfoEndpoint
				cfg.Components.Gateway.AAI.GatewayEndpoint = externalAAIUserinfoEndpoint
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {
					`url: "` + externalAAIUserinfoEndpoint + `"`,
				},
				"templates/dataportal.yaml": {
					`value: "` + externalAAIAuthRootURL + `"`,
				},
			},
			notContains: map[string][]string{
				"templates/gateway.yaml":     {"wait-for-aai-service"},
				"templates/aai-service.yaml": {"name: aai-service"},
				"templates/pvc.yaml":         {"name: aai"},
			},
		},
		{
			name: "gateway endpoint overrides gateway aai endpoint only",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-gateway-aai-endpoint"
				cfg.Components.Gateway.AAI.Enabled = true
				cfg.Components.AAIService.Enabled = false
				cfg.Components.Gateway.AAI.ServiceEndpoint = externalAAIAuthRootURL
				cfg.Components.Gateway.AAI.GatewayEndpoint = gatewayAAIUserinfoEndpoint + "/"
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {
					`url: "` + gatewayAAIUserinfoEndpoint + `/"`,
				},
				"templates/dataportal.yaml": {
					`value: "` + externalAAIAuthRootURL + `"`,
				},
			},
		},
		{
			name: "gateway endpoint keeps existing userinfo suffix",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-gateway-aai-keycloak-endpoint"
				cfg.Components.Gateway.AAI.Enabled = true
				cfg.Components.AAIService.Enabled = false
				cfg.Components.Gateway.AAI.ServiceEndpoint = externalAAIAuthRootURL
				cfg.Components.Gateway.AAI.GatewayEndpoint = gatewayAAIKeycloakUserinfo + "/"
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {
					`url: "` + gatewayAAIKeycloakUserinfo + `/"`,
				},
				"templates/dataportal.yaml": {
					`value: "` + externalAAIAuthRootURL + `"`,
				},
			},
		},
		{
			name: "service ingress tls renders dedicated secret per ingress",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-ingress-tls"
				cfg.Components.PlatformUI.TLS.Enabled = true
				cfg.Components.PlatformUI.TLS.SecretName = dataportalTLSSecret
				cfg.Components.Gateway.TLS.Enabled = true
				cfg.Components.Gateway.TLS.SecretName = gatewayTLSSecret
				cfg.Components.Backoffice.TLS.Enabled = true
				cfg.Components.Backoffice.TLS.SecretName = "backoffice-tls"
				cfg.Components.AAIService.Enabled = true
				cfg.Components.AAIService.TLS.Enabled = true
				cfg.Components.AAIService.TLS.SecretName = "aai-tls"
			},
			wantContains: map[string][]string{
				"templates/dataportal.yaml": {
					`secretName: "` + dataportalTLSSecret + `"`,
					certManagerIssuerAnnotation,
				},
				"templates/gateway.yaml": {
					`secretName: "` + gatewayTLSSecret + `"`,
					certManagerIssuerAnnotation,
				},
				"templates/backoffice-ui.yaml": {
					`secretName: "backoffice-tls"`,
					certManagerIssuerAnnotation,
				},
				"templates/aai-service.yaml": {
					`secretName: "aai-tls"`,
					certManagerIssuerAnnotation,
				},
			},
			notContains: map[string][]string{
				"templates/dataportal.yaml":    {gatewayTLSSecret, "backoffice-tls", "aai-tls"},
				"templates/gateway.yaml":       {dataportalTLSSecret, "backoffice-tls", "aai-tls"},
				"templates/backoffice-ui.yaml": {dataportalTLSSecret, gatewayTLSSecret, "aai-tls"},
				"templates/aai-service.yaml":   {dataportalTLSSecret, gatewayTLSSecret, "backoffice-tls"},
			},
		},
		{
			name: "service ingress tls disabled omits ingress tls block",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-ingress-tls-toggle"
				cfg.Components.PlatformUI.TLS.Enabled = true
				cfg.Components.PlatformUI.TLS.SecretName = dataportalTLSSecret
				cfg.Components.Gateway.TLS.Enabled = false
				cfg.Components.Gateway.TLS.SecretName = gatewayTLSSecret
			},
			wantContains: map[string][]string{
				"templates/dataportal.yaml": {
					`secretName: "` + dataportalTLSSecret + `"`,
					certManagerIssuerAnnotation,
				},
			},
			notContains: map[string][]string{
				"templates/gateway.yaml": {"secretName:", "cert-manager.io/cluster-issuer:"},
			},
		},
		{
			name: "gateway cert manager annotation does not depend on aai tls",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-gateway-cert-issuer"
				cfg.Components.Gateway.TLS.Enabled = true
				cfg.Components.Gateway.TLS.SecretName = gatewayTLSSecret
				cfg.Components.AAIService.TLS.Enabled = false
			},
			wantContains: map[string][]string{
				"templates/gateway.yaml": {
					`secretName: "` + gatewayTLSSecret + `"`,
					certManagerIssuerAnnotation,
				},
			},
		},
		{
			name: "cert manager annotation omitted when issuer is empty",
			mutate: func(cfg *config.Config) {
				cfg.Name = "test-empty-cert-issuer"
				cfg.CertManagerIssuer = ""
				cfg.Components.PlatformUI.TLS.Enabled = true
				cfg.Components.PlatformUI.TLS.SecretName = dataportalTLSSecret
			},
			wantContains: map[string][]string{
				"templates/dataportal.yaml": {`secretName: "` + dataportalTLSSecret + `"`},
			},
			notContains: map[string][]string{
				"templates/dataportal.yaml": {"cert-manager.io/cluster-issuer:"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GetDefaultConfig()
			if tt.mutate != nil {
				tt.mutate(cfg)
			}

			files, err := cfg.Render()
			if err != nil {
				t.Fatalf("Render() returned error: %v", err)
			}

			for _, suffix := range tt.wantFiles {
				mustGetRenderedFileBySuffix(t, files, suffix)
			}

			for suffix, substrings := range tt.wantContains {
				content := mustGetRenderedFileBySuffix(t, files, suffix)
				contentContains(t, content, suffix, substrings)
			}

			for suffix, substrings := range tt.notContains {
				content, _ := getRenderedFileBySuffix(files, suffix)
				contentExcludes(t, content, suffix, substrings)
			}
		})
	}
}

func TestIngressHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		host *string
		want string
	}{
		{name: "default", want: `host: "localhost"`},
		{name: "any host", host: new(string)},
		{name: "custom", host: func() *string { s := "ingress.example.org"; return &s }(), want: `host: "ingress.example.org"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.GetDefaultConfig()
			cfg.Name = "test-ingress"
			cfg.IngressHost = tc.host
			cfg.Components.AAIService.Enabled = true
			cfg.Components.OpenTelemetry.Enabled = true
			files, err := cfg.Render()
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"gateway", "dataportal", "backoffice-ui", "aai-service", "otel"} {
				content := files["epos/templates/"+name+".yaml"]
				if content == "" {
					t.Fatalf("missing template %s", name)
				}
				if tc.want == "" && strings.Contains(content, "host:") {
					t.Errorf("%s contains ingress host", name)
				} else if tc.want != "" && !strings.Contains(content, tc.want) {
					t.Errorf("%s missing %s", name, tc.want)
				}
			}
			if !strings.Contains(files["epos/templates/aai-service.yaml"], "http://localhost/test-ingress/aai") {
				t.Error("public AAI URL changed")
			}
		})
	}
}

func TestIngressHostCertManagerValidation(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Name = "test-ingress"
	cfg.Protocol = "https"
	cfg.Components.Gateway.TLS.Enabled = true
	cfg.Components.Gateway.TLS.SecretName = "gateway-tls"
	host := "other.example.org"
	cfg.IngressHost = &host
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "ingress_host must match domain") {
		t.Fatalf("unexpected validation error: %v", err)
	}
	host = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("hostless ingress validation: %v", err)
	}
}

func mustGetRenderedFileBySuffix(t *testing.T, files map[string]string, suffix string) string {
	t.Helper()

	content, ok := getRenderedFileBySuffix(files, suffix)
	if ok {
		return content
	}

	keys := make([]string, 0, len(files))
	for path := range files {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	t.Fatalf("rendered file with suffix %q not found; got files: %v", suffix, keys)

	return ""
}

func getRenderedFileBySuffix(files map[string]string, suffix string) (string, bool) {
	for path, content := range files {
		if strings.HasSuffix(path, suffix) {
			return content, true
		}
	}

	return "", false
}

func contentContains(t *testing.T, content, fileName string, expected []string) {
	t.Helper()

	for _, s := range expected {
		if !strings.Contains(content, s) {
			t.Errorf("file %q missing expected content %q", fileName, s)
		}
	}
}

func contentExcludes(t *testing.T, content, fileName string, unexpected []string) {
	t.Helper()

	for _, s := range unexpected {
		if strings.Contains(content, s) {
			t.Errorf("file %q should not contain %q", fileName, s)
		}
	}
}
