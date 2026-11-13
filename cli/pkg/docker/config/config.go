// Package config provides Docker environment configuration loading, modeling, validation, and rendering.
package config

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var defaultConfig []byte

// GetDefaultConfig creates a default Docker configuration with unique credentials.
func GetDefaultConfig() (*EnvConfig, error) {
	var config EnvConfig
	if err := yaml.Unmarshal(defaultConfig, &config); err != nil {
		return nil, fmt.Errorf("unmarshal default config: %w", err)
	}
	for _, password := range []*string{
		&config.Components.MetadataDatabase.Password,
		&config.Components.DataRetrieverDatabase.Password,
		&config.Components.ConverterDatabase.Password,
		&config.Components.SharingDatabase.Password,
		&config.Components.AAIService.Password,
		&config.Components.OpenTelemetry.AdminPassword,
	} {
		value, err := generatePassword()
		if err != nil {
			return nil, fmt.Errorf("generate environment password: %w", err)
		}
		*password = value
	}
	return &config, nil
}

func generatePassword() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var password [8]byte
	for i := range password {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		password[i] = alphabet[index.Int64()]
	}
	return string(password[:]), nil
}

// LoadConfig loads a Docker configuration from a YAML file.
func LoadConfig(path string) (*EnvConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	config, err := LoadConfigFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return config, nil
}

// LoadConfigFromBytes loads a Docker configuration from YAML bytes.
func LoadConfigFromBytes(data []byte) (*EnvConfig, error) {
	var config EnvConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config yaml: %w", err)
	}
	return &config, nil
}

// Bytes marshals the current configuration to YAML bytes.
func (e *EnvConfig) Bytes() ([]byte, error) {
	data, err := yaml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return data, nil
}

// Save writes the current configuration as YAML to path.
func (e *EnvConfig) Save(path string) error {
	data, err := e.Bytes()
	if err != nil {
		return err
	}
	if err := common.CreateFileWithContent(path, string(data), false); err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	return nil
}

// BuildEnvURLs builds public GUI, API, backoffice, and optional Grafana URLs.
func (e *EnvConfig) BuildEnvURLs() (*common.URLs, error) {
	buildURL := func(port int, basePath string) (string, error) {
		base := &url.URL{Scheme: e.Protocol, Host: net.JoinHostPort(e.Domain, strconv.Itoa(port))}
		return url.JoinPath(base.String(), basePath)
	}
	guiURL, err := buildURL(e.Components.PlatformUI.Port, e.Components.PlatformUI.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("build GUI URL: %w", err)
	}
	apiURL, err := buildURL(e.Components.Gateway.Port, e.Components.Gateway.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("build API URL: %w", err)
	}
	backofficeURL, err := buildURL(e.Components.Backoffice.GUI.Port, e.Components.Backoffice.GUI.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("build backoffice URL: %w", err)
	}
	docsURL, err := buildURL(e.Components.Gateway.Port, "/")
	if err != nil {
		return nil, fmt.Errorf("build API docs URL: %w", err)
	}
	urls := &common.URLs{GUIURL: guiURL, APIDocsURL: docsURL, APIBaseURL: apiURL, BackofficeURL: &backofficeURL}
	if e.Components.OpenTelemetry.ManagedCollectorEnabled() {
		grafanaURL, err := buildURL(e.Components.OpenTelemetry.GrafanaPort, "/")
		if err != nil {
			return nil, fmt.Errorf("build Grafana URL: %w", err)
		}
		urls.GrafanaURL = &grafanaURL
	}
	return urls, nil
}

// BackofficeURL returns the externally reachable backoffice base URL.
func (e *EnvConfig) BackofficeURL() string {
	base := &url.URL{Scheme: e.Protocol, Host: net.JoinHostPort(e.Domain, strconv.Itoa(e.Components.Backoffice.GUI.Port))}
	result, err := url.JoinPath(base.String(), e.Components.Backoffice.GUI.BaseURL)
	if err != nil {
		return ""
	}
	return result
}

// PlatformURL returns the externally reachable platform GUI base URL.
func (e *EnvConfig) PlatformURL() string {
	base := &url.URL{Scheme: e.Protocol, Host: net.JoinHostPort(e.Domain, strconv.Itoa(e.Components.PlatformUI.Port))}
	result, err := url.JoinPath(base.String(), e.Components.PlatformUI.BaseURL)
	if err != nil {
		return ""
	}
	return result
}

// CheckForUpdates checks configured container images for newer tags.
func (e *EnvConfig) CheckForUpdates() ([]display.ImageUpdateInfo, error) {
	updates, err := common.CheckImagesForUpdates(e.ActiveImages())
	if err != nil {
		return nil, fmt.Errorf("check images for updates: %w", err)
	}
	return updates, nil
}

// EnsurePortsFree auto-selects available ports when the defaults are in use.
func (e *EnvConfig) EnsurePortsFree() error {
	if !e.UsingDefaultPorts() {
		return nil
	}
	ports := []struct {
		name string
		port *int
	}{
		{"platform UI", &e.Components.PlatformUI.Port},
		{"gateway", &e.Components.Gateway.Port},
		{"backoffice", &e.Components.Backoffice.GUI.Port},
	}
	if e.Components.AAIService.Enabled {
		ports = append(ports, struct {
			name string
			port *int
		}{"AAI", &e.Components.AAIService.Port})
	}
	if e.Components.OpenTelemetry.ManagedCollectorEnabled() {
		ports = append(ports, struct {
			name string
			port *int
		}{"OpenTelemetry Grafana", &e.Components.OpenTelemetry.GrafanaPort})
	}
	for _, item := range ports {
		free, err := common.IsPortFree(*item.port)
		if err != nil {
			display.Warn("error checking availability of port for %s: %d. Continuing anyway", item.name, *item.port)
			continue
		}
		if !free {
			port, err := common.FindFreePort()
			if err != nil {
				return fmt.Errorf("find free %s port: %w", item.name, err)
			}
			display.Info("Port %d for %s is already used, using %d", *item.port, item.name, port)
			*item.port = port
		}
	}
	return nil
}

// Validate checks whether the configuration contains all required values.
func (e *EnvConfig) Validate() error {
	if err := e.Components.Gateway.MetadataSync.Validate(); err != nil {
		return err
	}
	for i, recipient := range e.Components.Gateway.Mailgun.RecipientOverrides {
		address, err := mail.ParseAddress(recipient)
		if err != nil || address.Address != recipient {
			return fmt.Errorf("components.gateway.mailgun.recipient_overrides[%d] must be a valid email address", i)
		}
	}
	if m := e.Components.Gateway.Mailgun; m.Enable && (m.Domain == "" || m.From == "" || m.APIKey == "") {
		return fmt.Errorf("Mailgun domain, sender and API key are required when enabled")
	}
	if e.Name == "" || e.Domain == "" {
		return fmt.Errorf("environment name and domain are required")
	}
	if e.Protocol != "http" && e.Protocol != "https" {
		return fmt.Errorf("protocol must be http or https")
	}
	if strings.ContainsAny(e.Domain, "/@?#[]\\\" \t\r\n") || (strings.Contains(e.Domain, ":") && net.ParseIP(e.Domain) == nil) {
		return fmt.Errorf("domain must be a hostname or unbracketed IP address")
	}
	if e.Components.Gateway.BaseURL != "/v1" {
		return fmt.Errorf("gateway base_url must be /v1")
	}
	if e.Components.PlatformUI.BaseURL == "" || !strings.HasPrefix(e.Components.PlatformUI.BaseURL, "/") {
		return fmt.Errorf("platform UI base_url must start with /")
	}
	if e.Components.Backoffice.GUI.BaseURL != "/" {
		return fmt.Errorf("backoffice base_url must be /")
	}
	ports := []int{e.Components.PlatformUI.Port, e.Components.Gateway.Port, e.Components.Backoffice.GUI.Port}
	if e.Components.AAIService.Enabled {
		ports = append(ports, e.Components.AAIService.Port)
	}
	if e.Components.OpenTelemetry.ManagedCollectorEnabled() {
		ports = append(ports, e.Components.OpenTelemetry.GrafanaPort)
	}
	seen := make(map[int]bool)
	for _, port := range ports {
		if port < 1 || port > 65535 || seen[port] {
			return fmt.Errorf("service ports must be unique and between 1 and 65535")
		}
		seen[port] = true
	}
	if !e.Components.Gateway.AAI.Enabled || e.AAIAuthRootURL() == "" || e.AAIServiceEndpoint() == "" {
		return fmt.Errorf("gateway AAI and its public and gateway endpoints are required")
	}
	if e.Components.AAIService.Enabled && (e.Components.Gateway.AAI.ServiceEndpoint != "" || e.Components.Gateway.AAI.GatewayEndpoint != "") {
		return fmt.Errorf("external AAI endpoints cannot be used with the managed AAI service")
	}
	for _, endpoint := range []string{e.AAIAuthRootURL(), e.AAIServiceEndpoint()} {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(endpoint, "\"\\\r\n") {
			return fmt.Errorf("AAI endpoints must be absolute HTTP(S) URLs without credentials, query or fragment")
		}
	}
	if otel := e.Components.OpenTelemetry; otel.Enabled && otel.Endpoint != "" {
		parsed, err := url.Parse(otel.Endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(otel.Endpoint, "\"\\\r\n") {
			return fmt.Errorf("OpenTelemetry endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
		}
	}
	if aai := e.Components.AAIService; aai.Enabled {
		if strings.TrimSpace(aai.Name) == "" || strings.TrimSpace(aai.Surname) == "" || strings.TrimSpace(aai.Email) == "" || strings.TrimSpace(aai.Password) == "" {
			return fmt.Errorf("AAI administrator credentials are required")
		}
	}
	if otel := e.Components.OpenTelemetry; otel.ManagedCollectorEnabled() {
		if strings.TrimSpace(otel.AdminUser) == "" || strings.TrimSpace(otel.AdminPassword) == "" {
			return fmt.Errorf("grafana admin credentials are required when managed LGTM is enabled")
		}
	}
	for _, db := range []struct {
		name  string
		value MetadataDatabase
	}{
		{"metadata", e.Components.MetadataDatabase},
		{"data retriever", e.Components.DataRetrieverDatabase},
		{"converter", e.Components.ConverterDatabase},
		{"sharing", e.Components.SharingDatabase},
	} {
		if db.value.User == "" || db.value.Password == "" || db.value.DBName == "" {
			return fmt.Errorf("%s database name, user and password are required", db.name)
		}
		if !db.value.Enabled && (db.value.Host == "" || db.value.Port < 1 || db.value.Port > 65535) {
			return fmt.Errorf("%s external database host and valid port are required", db.name)
		}
	}
	for _, image := range e.ActiveImages() {
		if strings.TrimSpace(image.Ref) == "" {
			return fmt.Errorf("%s image is required", image.Name)
		}
	}
	return nil
}
