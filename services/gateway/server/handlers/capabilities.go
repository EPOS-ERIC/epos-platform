package handlers

import (
	"context"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) GetCapabilities(context.Context) (*gatewayapi.Capabilities, error) {
	metadataSync := gatewayapi.CapabilitiesMetadataSync{Management: gatewayapi.CapabilitiesMetadataSyncManagementLocal}
	if h.capabilities.MetadataSyncBaseURL != "" {
		metadataSync.Management = gatewayapi.CapabilitiesMetadataSyncManagementRemoteSync
		metadataSync.BaseURL = gatewayapi.NewOptString(h.capabilities.MetadataSyncBaseURL)
	}

	capabilities := &gatewayapi.Capabilities{
		MetadataSync:   metadataSync,
		Authentication: gatewayapi.CapabilitiesAuthentication{RootURL: h.capabilities.AuthRootURL},
		Notifications:  gatewayapi.CapabilitiesNotifications{EmailEnabled: h.capabilities.EmailEnabled},
	}
	if h.capabilities.IssueTrackerURL != nil && h.capabilities.IssueTrackerOpenNew != nil {
		capabilities.IssueTracker = gatewayapi.NewOptCapabilitiesIssueTracker(gatewayapi.CapabilitiesIssueTracker{
			URL:     *h.capabilities.IssueTrackerURL,
			OpenNew: *h.capabilities.IssueTrackerOpenNew,
		})
	}
	return capabilities, nil
}
