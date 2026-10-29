package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"gopkg.in/yaml.v3"
)

// EnvConfig represents the full Docker environment configuration schema.
type EnvConfig struct {
	Name       string     `yaml:"name"`
	Domain     string     `yaml:"domain"`
	Protocol   string     `yaml:"protocol"`
	Components Components `yaml:"components"`
	Images     Images     `yaml:"images"`
}

// Images lists the container images used by the Docker deployment.
type Images struct {
	PlatformUIImage        string `yaml:"platform_ui"`
	BackofficeUIImage      string `yaml:"backoffice_ui"`
	GatewayImage           string `yaml:"gateway"`
	MetadataLifecycleImage string `yaml:"metadata_lifecycle"`
	DataRetrieverImage     string `yaml:"data_retriever"`
	SharingServiceImage    string `yaml:"sharing_service"`
	AAIServiceImage        string `yaml:"aai_service"`
	MetadataDatabaseImage  string `yaml:"metadata_database"`
	DataRetrieverDBImage   string `yaml:"data_retriever_database"`
	SharingDBImage         string `yaml:"sharing_database"`
	OTelLGTMImage          string `yaml:"otel_lgtm"`
}

// UnmarshalYAML rejects obsolete image keys rather than silently dropping them.
func (i *Images) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		for n := 0; n+1 < len(node.Content); n += 2 {
			key := node.Content[n].Value
			if key == "dataportal" || key == "dataportal_image" {
				return fmt.Errorf("obsolete image key %q: use platform_ui", key)
			}
			if strings.HasSuffix(key, "_image") {
				return fmt.Errorf("obsolete image key %q: remove the _image suffix", key)
			}
		}
	}
	type images Images
	return node.Decode((*images)(i))
}

// PlatformUI configures the platform UI endpoint.
type PlatformUI struct {
	BaseURL string `yaml:"base_url"`
	Port    int    `yaml:"port"`
}

// AAI configures AAI integration for gateway services.
type AAI struct {
	Enabled         bool   `yaml:"enabled"`
	ServiceEndpoint string `yaml:"service_endpoint"`
	GatewayEndpoint string `yaml:"gateway_endpoint"`
}

// Gateway configures gateway URLs, port, and auth integrations.
type Gateway struct {
	AAI     AAI     `yaml:"aai"`
	BaseURL string  `yaml:"base_url"`
	Port    int     `yaml:"port"`
	Mailgun Mailgun `yaml:"mailgun"`
}

type Mailgun struct {
	Enable bool   `yaml:"enable"`
	Domain string `yaml:"domain"`
	From   string `yaml:"from"`
	APIKey string `yaml:"api_key"`
}

// GUI configures a generic GUI service endpoint.
type GUI struct {
	BaseURL string `yaml:"base_url"`
	Port    int    `yaml:"port"`
}

// Backoffice configures backoffice UI and service behavior.
type Backoffice struct {
	GUI GUI `yaml:"gui"`
}

// MetadataDatabase configures metadata database connection and pooling.
type MetadataDatabase struct {
	Enabled  bool   `yaml:"enabled"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"db_name"`
}

type AAIService struct {
	Enabled  bool   `yaml:"enabled"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	Surname  string `yaml:"surname"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// OpenTelemetry configures telemetry export and an optional managed LGTM collector.
type OpenTelemetry struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`
	GrafanaPort   int    `yaml:"grafana_port"`
	AdminUser     string `yaml:"admin_user"`
	AdminPassword string `yaml:"admin_password"`
}

func (o OpenTelemetry) ManagedCollectorEnabled() bool {
	return o.Enabled && o.Endpoint == ""
}

func (o OpenTelemetry) ExporterEndpoint() string {
	if !o.Enabled {
		return ""
	}
	if o.Endpoint != "" {
		return o.Endpoint
	}
	return "http://otel-lgtm:4318"
}

// Components groups all service-level component settings.
type Components struct {
	PlatformUI            PlatformUI       `yaml:"platform_ui"`
	Gateway               Gateway          `yaml:"gateway"`
	Backoffice            Backoffice       `yaml:"backoffice"`
	MetadataDatabase      MetadataDatabase `yaml:"metadata_database"`
	DataRetrieverDatabase MetadataDatabase `yaml:"data_retriever_database"`
	SharingDatabase       MetadataDatabase `yaml:"sharing_database"`
	AAIService            AAIService       `yaml:"aai_service"`
	OpenTelemetry         OpenTelemetry    `yaml:"otel"`
}

// UnmarshalYAML rejects the obsolete platform GUI key.
func (c *Components) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		for n := 0; n+1 < len(node.Content); n += 2 {
			if node.Content[n].Value == "platform_gui" {
				return fmt.Errorf("obsolete component key %q: use platform_ui", "platform_gui")
			}
		}
	}
	type components Components
	return node.Decode((*components)(c))
}

// ActiveImages returns the ordered list of configured images required by enabled services.
func (e *EnvConfig) ActiveImages() []common.NamedImage {
	images := []common.NamedImage{
		{Name: "Platform UI", Ref: e.Images.PlatformUIImage},
		{Name: "Backoffice UI", Ref: e.Images.BackofficeUIImage},
		{Name: "Gateway", Ref: e.Images.GatewayImage},
		{Name: "Metadata Lifecycle", Ref: e.Images.MetadataLifecycleImage},
		{Name: "Data Retriever", Ref: e.Images.DataRetrieverImage},
		{Name: "Sharing", Ref: e.Images.SharingServiceImage},
	}
	if e.Components.MetadataDatabase.Enabled {
		images = append(images, common.NamedImage{Name: "Metadata Database", Ref: e.Images.MetadataDatabaseImage})
	}
	if e.Components.DataRetrieverDatabase.Enabled {
		images = append(images, common.NamedImage{Name: "Data Retriever Database", Ref: e.Images.DataRetrieverDBImage})
	}
	if e.Components.SharingDatabase.Enabled {
		images = append(images, common.NamedImage{Name: "Sharing Database", Ref: e.Images.SharingDBImage})
	}

	if e.Components.AAIService.Enabled {
		images = append(images, common.NamedImage{Name: "AAI Service", Ref: e.Images.AAIServiceImage})
	}
	if e.Components.OpenTelemetry.ManagedCollectorEnabled() {
		images = append(images, common.NamedImage{Name: "OpenTelemetry LGTM", Ref: e.Images.OTelLGTMImage})
	}

	return images
}

// UsingDefaultPorts reports whether configured ports still match defaults.
func (e *EnvConfig) UsingDefaultPorts() bool {
	if e.Components.PlatformUI.Port != 32000 {
		return false
	}
	if e.Components.Gateway.Port != 33000 {
		return false
	}
	if e.Components.Backoffice.GUI.Port != 34000 {
		return false
	}
	if e.Components.AAIService.Enabled && e.Components.AAIService.Port != 35000 {
		return false
	}
	if e.Components.OpenTelemetry.ManagedCollectorEnabled() && e.Components.OpenTelemetry.GrafanaPort != 36000 {
		return false
	}
	return true
}

// AAIAuthRootURL returns the externally reachable auth root URL used by UIs.
func (e *EnvConfig) AAIAuthRootURL() string {
	if !e.Components.Gateway.AAI.Enabled {
		return ""
	}

	endpoint := common.TrimAuthURL(e.Components.Gateway.AAI.ServiceEndpoint)

	if endpoint != "" {
		return endpoint
	}

	if e.Components.AAIService.Enabled {
		return fmt.Sprintf("%s://%s", e.Protocol, net.JoinHostPort(e.Domain, strconv.Itoa(e.Components.AAIService.Port)))
	}

	return ""
}

// GatewayPublicURL returns the externally reachable gateway root URL for the Compose template.
func (e *EnvConfig) GatewayPublicURL() string {
	return fmt.Sprintf("%s://%s", e.Protocol, net.JoinHostPort(e.Domain, strconv.Itoa(e.Components.Gateway.Port)))
}

// AAIServiceEndpoint returns gateway-reachable AAI service endpoint.
func (e *EnvConfig) AAIServiceEndpoint() string {
	if !e.Components.Gateway.AAI.Enabled {
		return ""
	}

	gatewayEndpoint := common.TrimAuthURL(e.Components.Gateway.AAI.GatewayEndpoint)

	if gatewayEndpoint != "" {
		return gatewayEndpoint
	}

	endpoint := common.TrimAuthURL(e.Components.Gateway.AAI.ServiceEndpoint)

	if endpoint != "" {
		return endpoint
	}

	if e.Components.AAIService.Enabled {
		return "http://aai-service:8080"
	}

	return ""
}
