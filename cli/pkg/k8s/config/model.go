package config

import "github.com/EPOS-ERIC/epos-platform/cli/common"

// Config represents the full Kubernetes environment values schema.
type Config struct {
	Name               string           `yaml:"name"`
	Domain             string           `yaml:"domain"`
	IngressHost        *string          `yaml:"ingress_host,omitempty"`
	Protocol           string           `yaml:"protocol"`
	Debug              bool             `yaml:"debug"`
	URLPrefixNamespace bool             `yaml:"url_prefix_namespace"`
	CreateNamespace    bool             `yaml:"create_namespace"`
	ImagePullPolicy    string           `yaml:"image_pull_policy"`
	Components         Components       `yaml:"components"`
	ImagePullSecrets   ImagePullSecrets `yaml:"image_pull_secrets"`
	CertManagerIssuer  string           `yaml:"cert_manager_issuer"`
	Images             common.Images    `yaml:"images"`
}

// TLS configures ingress TLS behavior.
type TLS struct {
	Enabled    bool   `yaml:"enabled"`
	SecretName string `yaml:"secret_name"`
}

// PlatformUI configures the platform UI endpoint.
type PlatformUI struct {
	BaseURL string `yaml:"base_url"`
	TLS     TLS    `yaml:"tls"`
}

// AAI configures AAI integration for gateway services.
type AAI struct {
	Enabled         bool   `yaml:"enabled"`
	ServiceEndpoint string `yaml:"service_endpoint"`
	GatewayEndpoint string `yaml:"gateway_endpoint"`
}

// Gateway configures gateway URLs and auth integrations.
type Gateway struct {
	AAI     AAI     `yaml:"aai"`
	BaseURL string  `yaml:"base_url"`
	TLS     TLS     `yaml:"tls"`
	Mailgun Mailgun `yaml:"mailgun"`
}

type Mailgun struct {
	Enable             bool     `yaml:"enable"`
	Domain             string   `yaml:"domain"`
	From               string   `yaml:"from"`
	APIKey             string   `yaml:"api_key"`
	RecipientOverrides []string `yaml:"recipient_overrides"`
}

// GUI configures a generic GUI service endpoint.
type GUI struct {
	BaseURL string `yaml:"base_url"`
}

// Backoffice configures the backoffice UI.
type Backoffice struct {
	GUI GUI `yaml:"gui"`
	TLS TLS `yaml:"tls"`
}

// MetadataDatabase configures a database connection.
type MetadataDatabase struct {
	Enabled  bool   `yaml:"enabled"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"db_name"`
}

// AAIService configures the embedded AAI service.
type AAIService struct {
	Enabled  bool   `yaml:"enabled"`
	Name     string `yaml:"name"`
	Surname  string `yaml:"surname"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
	TLS      TLS    `yaml:"tls"`
}

// Components groups all service-level component settings.
type Components struct {
	PlatformUI            PlatformUI       `yaml:"platform_ui"`
	Gateway               Gateway          `yaml:"gateway"`
	Backoffice            Backoffice       `yaml:"backoffice"`
	MetadataDatabase      MetadataDatabase `yaml:"metadata_database"`
	DataRetrieverDatabase MetadataDatabase `yaml:"data_retriever_database"`
	ConverterDatabase     MetadataDatabase `yaml:"converter_database"`
	SharingDatabase       MetadataDatabase `yaml:"sharing_database"`
	AAIService            AAIService       `yaml:"aai_service"`
	OpenTelemetry         OpenTelemetry    `yaml:"otel"`
}

// OpenTelemetry configures telemetry export and the optional managed collector.
type OpenTelemetry struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`
	AdminUser     string `yaml:"admin_user"`
	AdminPassword string `yaml:"admin_password"`
	TLS           TLS    `yaml:"tls"`
}

// ImagePullSecrets configures optional private registry pull secret generation.
type ImagePullSecrets struct {
	Enabled          bool   `yaml:"enabled"`
	RegistryServer   string `yaml:"registry_server"`
	RegistryUsername string `yaml:"registry_username"`
	RegistryPassword string `yaml:"registry_password"`
}

// String returns a concise string representation of key Config fields.
func (c *Config) String() string {
	if c == nil {
		return ""
	}

	out := "name=" + c.Name + ", domain=" + c.Domain + ", protocol=" + c.Protocol

	if c.URLPrefixNamespace {
		out += ", url_prefix_namespace=true"
	} else {
		out += ", url_prefix_namespace=false"
	}

	return out
}
