// Package config provides Kubernetes/Helm configuration loading, modeling, and rendering.
package config

import (
	"crypto/rand"
	"embed"
	"fmt"
	"io/fs"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

const defaultImagePullPolicy = "IfNotPresent"

//go:embed all:helm/**
var chartFS embed.FS

// GetDefaultConfig returns a parsed copy of the embedded default K8s values configuration.
func GetDefaultConfig() *Config {
	var config Config

	err := yaml.Unmarshal(GetDefaultConfigBytes(), &config)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling default config yaml: %w", err))
	}
	for _, password := range []*string{
		&config.Components.MetadataDatabase.Password,
		&config.Components.DataRetrieverDatabase.Password,
		&config.Components.ConverterDatabase.Password,
		&config.Components.SharingDatabase.Password,
		&config.Components.AAIService.Password,
		&config.Components.OpenTelemetry.AdminPassword,
	} {
		*password = rand.Text()
	}
	return &config
}

// GetDefaultConfigBytes returns the raw embedded Helm values.yaml bytes.
func GetDefaultConfigBytes() []byte {
	out, err := chartFS.ReadFile("helm/values.yaml")
	if err != nil {
		panic(fmt.Errorf("error reading values.yaml: %w", err))
	}
	return out
}

// LoadConfig loads a K8s configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config yaml: %w", err)
	}

	return &config, nil
}

// Save writes the current configuration as YAML to path.
func (c *Config) Save(path string) error {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := common.CreateFileWithContent(path, string(bytes), false); err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}

	return nil
}

// BuildEnvURLs builds GUI, API, and backoffice URLs from the configuration.
func (c *Config) BuildEnvURLs() (*common.URLs, error) {
	buildURL := func(paths ...string) (string, error) {
		base := &url.URL{
			Scheme: c.Protocol,
			Host:   c.Domain,
		}

		path := base.String()

		if c.URLPrefixNamespace {
			var err error
			path, err = url.JoinPath(path, c.Name)
			if err != nil {
				return "", fmt.Errorf("error joining namespace to base URL: %w", err)
			}
		}

		return url.JoinPath(path, paths...)
	}

	guiURL, err := buildURL(c.Components.PlatformUI.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("error building GUI URL: %w", err)
	}

	apiURL, err := buildURL("/scalar/")
	if err != nil {
		return nil, fmt.Errorf("error building API URL: %w", err)
	}

	apiBaseURL, err := buildURL(c.Components.Gateway.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("error building API base URL: %w", err)
	}
	urls := &common.URLs{
		GUIURL:     guiURL,
		APIDocsURL: strings.TrimRight(apiURL, "/") + "/",
		APIBaseURL: apiBaseURL,
	}

	backofficeURL, err := buildURL(c.Components.Backoffice.GUI.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("error building Backoffice URL: %w", err)
	}
	backofficeURL = strings.TrimRight(backofficeURL, "/") + "/"
	urls.BackofficeURL = &backofficeURL

	if c.Components.OpenTelemetry.Enabled && c.Components.OpenTelemetry.Endpoint == "" {
		grafanaURL, err := buildURL("/grafana/")
		if err != nil {
			return nil, fmt.Errorf("error building Grafana URL: %w", err)
		}
		grafanaURL = strings.TrimRight(grafanaURL, "/") + "/"
		urls.GrafanaURL = &grafanaURL
	}

	return urls, nil
}

func validateIngressTLSSecret(serviceName string, tls TLS) error {
	if tls.Enabled && tls.SecretName == "" {
		return fmt.Errorf("%s tls secret_name is required when tls is enabled", serviceName)
	}

	return nil
}

// Validate checks whether the configuration contains all required values.
func (c *Config) Validate() error {
	for i, recipient := range c.Components.Gateway.Mailgun.RecipientOverrides {
		address, err := mail.ParseAddress(recipient)
		if err != nil || address.Address != recipient {
			return fmt.Errorf("components.gateway.mailgun.recipient_overrides[%d] must be a valid email address", i)
		}
	}
	if m := c.Components.Gateway.Mailgun; m.Enable && (m.Domain == "" || m.From == "" || m.APIKey == "") {
		return fmt.Errorf("Mailgun domain, sender and API key are required when enabled")
	}
	if c.ImagePullPolicy == "" {
		c.ImagePullPolicy = defaultImagePullPolicy
	}
	if c.ImagePullPolicy != "Always" && c.ImagePullPolicy != "IfNotPresent" && c.ImagePullPolicy != "Never" {
		return fmt.Errorf("image_pull_policy must be Always, IfNotPresent, or Never")
	}
	// Basic required fields
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if c.IngressHost != nil && *c.IngressHost != "" && *c.IngressHost != c.Domain && c.CertManagerIssuer != "" &&
		(c.Components.PlatformUI.TLS.Enabled || c.Components.Gateway.TLS.Enabled ||
			c.Components.Backoffice.TLS.Enabled ||
			(c.Components.AAIService.Enabled && c.Components.AAIService.TLS.Enabled) ||
			(c.Components.OpenTelemetry.Enabled && c.Components.OpenTelemetry.Endpoint == "" && c.Components.OpenTelemetry.TLS.Enabled)) {
		return fmt.Errorf("ingress_host must match domain when cert-manager TLS is enabled")
	}
	if c.Protocol != "http" && c.Protocol != "https" {
		return fmt.Errorf("protocol must be http or https")
	}
	if err := validateIngressTLSSecret("platform ui", c.Components.PlatformUI.TLS); err != nil {
		return err
	}
	if err := validateIngressTLSSecret("gateway", c.Components.Gateway.TLS); err != nil {
		return err
	}
	if err := validateIngressTLSSecret("backoffice", c.Components.Backoffice.TLS); err != nil {
		return err
	}
	if c.Components.AAIService.Enabled {
		if err := validateIngressTLSSecret("aai service", c.Components.AAIService.TLS); err != nil {
			return err
		}
	}
	if c.Protocol != "https" && (c.Components.PlatformUI.TLS.Enabled || c.Components.Gateway.TLS.Enabled || c.Components.Backoffice.TLS.Enabled || (c.Components.AAIService.Enabled && c.Components.AAIService.TLS.Enabled) || (c.Components.OpenTelemetry.Enabled && c.Components.OpenTelemetry.Endpoint == "" && c.Components.OpenTelemetry.TLS.Enabled)) {
		return fmt.Errorf("protocol must be https when ingress TLS is enabled")
	}

	// Required images
	if c.Images.PlatformUIImage == "" {
		return fmt.Errorf("platform ui image is required")
	}
	if c.Images.GatewayImage == "" {
		return fmt.Errorf("gateway image is required")
	}
	if c.Components.MetadataDatabase.Enabled && c.Images.MetadataDatabaseImage == "" {
		return fmt.Errorf("metadata database image is required when metadata database is enabled")
	}
	if c.Components.AAIService.Enabled && c.Images.AAIServiceImage == "" {
		return fmt.Errorf("aai service image is required when aai service is enabled")
	}
	if c.Images.MetadataLifecycleImage == "" || c.Images.DataRetrieverImage == "" || c.Images.ConverterImage == "" || c.Images.SharingServiceImage == "" {
		return fmt.Errorf("metadata lifecycle, data retriever, converter, and sharing images are required")
	}
	if c.Components.DataRetrieverDatabase.Enabled && c.Images.DataRetrieverDBImage == "" {
		return fmt.Errorf("data retriever database image is required when managed")
	}
	if c.Components.ConverterDatabase.Enabled && c.Images.ConverterDBImage == "" {
		return fmt.Errorf("converter database image is required when managed")
	}
	if c.Components.SharingDatabase.Enabled && c.Images.SharingDBImage == "" {
		return fmt.Errorf("sharing database image is required when managed")
	}
	if c.Components.OpenTelemetry.Enabled && c.Components.OpenTelemetry.Endpoint == "" {
		if c.Images.OTelLGTMImage == "" || c.Components.OpenTelemetry.AdminUser == "" || c.Components.OpenTelemetry.AdminPassword == "" {
			return fmt.Errorf("managed opentelemetry image and admin credentials are required")
		}
		if err := validateIngressTLSSecret("grafana", c.Components.OpenTelemetry.TLS); err != nil {
			return err
		}
	}
	if c.Components.OpenTelemetry.Enabled && c.Components.OpenTelemetry.Endpoint != "" {
		endpoint := c.Components.OpenTelemetry.Endpoint
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("OpenTelemetry endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
		}
	}

	// Platform UI validation
	if c.Components.PlatformUI.BaseURL == "" {
		return fmt.Errorf("platform ui base url is required")
	}
	if !strings.HasPrefix(c.Components.PlatformUI.BaseURL, "/") {
		return fmt.Errorf("platform ui base url must start with /")
	}
	if !strings.HasSuffix(c.Components.PlatformUI.BaseURL, "/") {
		return fmt.Errorf("platform ui base url must end with /")
	}

	// Gateway validation
	if c.Components.Gateway.BaseURL == "" {
		return fmt.Errorf("gateway base url is required")
	}
	if !strings.HasPrefix(c.Components.Gateway.BaseURL, "/") {
		return fmt.Errorf("gateway base url must start with /")
	}
	if strings.TrimRight(c.Components.Gateway.BaseURL, "/") != "/v1" {
		return fmt.Errorf("gateway base url must be /v1 or /v1/")
	}

	// AAI validation
	if !c.Components.Gateway.AAI.Enabled {
		return fmt.Errorf("gateway aai must be enabled")
	}
	if c.Components.AAIService.Enabled && (c.Components.Gateway.AAI.ServiceEndpoint != "" || c.Components.Gateway.AAI.GatewayEndpoint != "") {
		return fmt.Errorf("external AAI endpoints cannot be used with the managed AAI service")
	}
	if c.Components.AAIService.Enabled {
		if c.Components.AAIService.Name == "" {
			return fmt.Errorf("aai service name is required when aai service is enabled")
		}
		if c.Components.AAIService.Surname == "" {
			return fmt.Errorf("aai service surname is required when aai service is enabled")
		}
		if c.Components.AAIService.Email == "" {
			return fmt.Errorf("aai service email is required when aai service is enabled")
		}
		if c.Components.AAIService.Password == "" {
			return fmt.Errorf("aai service password is required when aai service is enabled")
		}
	}
	if c.Components.Gateway.AAI.Enabled {
		aaiServiceEndpoint := common.TrimAuthURL(c.Components.Gateway.AAI.ServiceEndpoint)

		if aaiServiceEndpoint == "" && !c.Components.AAIService.Enabled {
			return fmt.Errorf("aai service endpoint is required when aai is enabled and embedded aai service is disabled")
		}
	}
	for _, endpoint := range []string{c.Components.Gateway.AAI.ServiceEndpoint, c.Components.Gateway.AAI.GatewayEndpoint} {
		if endpoint == "" {
			continue
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("AAI endpoints must be absolute HTTP(S) URLs without credentials, query or fragment")
		}
	}

	// Backoffice validation
	if c.Components.Backoffice.GUI.BaseURL == "" {
		return fmt.Errorf("backoffice base url is required")
	}
	if !strings.HasPrefix(c.Components.Backoffice.GUI.BaseURL, "/") {
		return fmt.Errorf("backoffice base url must start with /")
	}
	if !strings.HasSuffix(c.Components.Backoffice.GUI.BaseURL, "/") {
		return fmt.Errorf("backoffice base url must end with /")
	}
	if c.Images.BackofficeUIImage == "" {
		return fmt.Errorf("backoffice ui image is required")
	}

	// Metadata database validation
	for _, item := range []struct {
		name string
		db   MetadataDatabase
	}{
		{"data retriever", c.Components.DataRetrieverDatabase},
		{"converter", c.Components.ConverterDatabase},
		{"sharing", c.Components.SharingDatabase},
	} {
		if item.db.Host == "" || item.db.Port < 1 || item.db.Port > 65535 || item.db.User == "" || item.db.Password == "" || item.db.DBName == "" {
			return fmt.Errorf("%s database host, port, user, password, and database name are required", item.name)
		}
	}
	if c.Components.MetadataDatabase.User == "" {
		return fmt.Errorf("metadata database user is required")
	}
	if c.Components.MetadataDatabase.Password == "" {
		return fmt.Errorf("metadata database password is required")
	}
	if c.Components.MetadataDatabase.Host == "" {
		return fmt.Errorf("metadata database host is required")
	}
	if c.Components.MetadataDatabase.Port == 0 {
		return fmt.Errorf("metadata database port is required")
	}
	if c.Components.MetadataDatabase.Port < 1 || c.Components.MetadataDatabase.Port > 65535 {
		return fmt.Errorf("metadata database port must be between 1 and 65535")
	}
	if c.Components.MetadataDatabase.DBName == "" {
		return fmt.Errorf("metadata database db_name is required")
	}
	// Image pull secrets validation
	if c.ImagePullSecrets.Enabled {
		if c.ImagePullSecrets.RegistryServer == "" {
			return fmt.Errorf("image pull secrets registry_server is required when image pull secrets is enabled")
		}
		if c.ImagePullSecrets.RegistryUsername == "" {
			return fmt.Errorf("image pull secrets registry_username is required when image pull secrets is enabled")
		}
		if c.ImagePullSecrets.RegistryPassword == "" {
			return fmt.Errorf("image pull secrets registry_password is required when image pull secrets is enabled")
		}
	}

	return nil
}

// AsValues converts Config into Helm chart values.
func (c *Config) AsValues() (*chartutil.Values, error) {
	configYAML, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal env config to YAML: %w", err)
	}

	values, err := chartutil.ReadValues(configYAML)
	if err != nil {
		return nil, fmt.Errorf("parse chart values from YAML: %w", err)
	}

	return &values, nil
}

// GetChart loads the embedded EPOS Helm chart from the binary filesystem.
func GetChart() (*chart.Chart, error) {
	var files []*loader.BufferedFile

	err := fs.WalkDir(chartFS, "helm", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(chartFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded chart file %s: %w", path, err)
		}

		relPath, err := filepath.Rel("helm", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for embedded chart file %s: %w", path, err)
		}

		files = append(files, &loader.BufferedFile{
			Name: filepath.ToSlash(relPath),
			Data: data,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded chart: %w", err)
	}

	c, err := loader.LoadFiles(files)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded chart: %w", err)
	}

	if c == nil {
		return nil, fmt.Errorf("chart is nil")
	}

	return c, nil
}
