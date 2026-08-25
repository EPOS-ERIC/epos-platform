package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

func (c *Backoffice) CreateResource(ctx context.Context, actorID, groupID, resourceType string, input domain.CreateResourceInput) (domain.ResourceVersion, error) {
	metadata, err := createResourceMetadata(input.Metadata)
	if err != nil {
		return domain.ResourceVersion{}, err
	}

	record, response, err := c.resources.CreateResource(ctx, resourceType).
		XEPOSUserID(actorID).
		CreateResourceRequest(client.CreateResourceRequest{
			GroupId:  groupID,
			Metadata: metadata,
		}).Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	return resourceVersion(record)
}

func (c *Backoffice) GetResourceVersion(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	request := c.resources.GetResourceVersion(ctx, resourceType, resourceID, versionID)
	if actorID != "" {
		request = request.XEPOSUserID(actorID)
	}

	record, response, err := request.Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	return resourceVersion(record)
}

func (c *Backoffice) ReplaceResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.ReplaceResourceDraftInput) (domain.ResourceVersion, error) {
	request, err := replaceResourceDraftRequest(input)
	if err != nil {
		return domain.ResourceVersion{}, err
	}

	record, response, err := c.resources.ReplaceResourceDraft(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		ReplaceResourceDraftRequest(request).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}

	return resourceVersion(record)
}

func (c *Backoffice) DiscardResourceDraft(ctx context.Context, actorID, resourceType, resourceID, versionID string) error {
	response, err := c.resources.DiscardResourceDraft(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	return lifecycleError(response, err)
}

func (c *Backoffice) SubmitResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.SubmitResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) WithdrawResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.WithdrawResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) RejectResource(ctx context.Context, actorID, resourceType, resourceID, versionID string, input domain.RejectResourceInput) (domain.ResourceVersion, error) {
	record, response, err := c.resources.RejectResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		RejectRequest(client.RejectRequest{Note: input.Note}).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) ReopenResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.ReopenResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) PublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.PublishResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) UnpublishResource(ctx context.Context, actorID, resourceType, resourceID, versionID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.UnpublishResource(ctx, resourceType, resourceID, versionID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) CreateResourceDraftFromPublication(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CreateResourceDraftFromPublication(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) CreateResourceDraftFromArchive(ctx context.Context, actorID, resourceType, resourceID string) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CreateResourceDraftFromArchive(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func (c *Backoffice) CopyResource(ctx context.Context, actorID, groupID, resourceType, resourceID string, input domain.CopyResourceInput) (domain.ResourceVersion, error) {
	record, response, err := c.resources.CopyResource(ctx, resourceType, resourceID).
		XEPOSUserID(actorID).
		CopyResourceRequest(client.CopyResourceRequest{
			TargetGroupId: groupID,
			SourceIri:     input.SourceIRI,
		}).
		Execute()
	if err != nil {
		return domain.ResourceVersion{}, lifecycleError(response, err)
	}
	return resourceVersion(record)
}

func createResourceMetadata(metadata domain.DraftMetadata) (client.CreateResourceRequestMetadata, error) {
	switch metadata := metadata.(type) {
	case domain.DatasetDraftMetadata:
		value := datasetDraftMetadata(metadata)
		return client.DatasetDraftMetadataAsCreateResourceRequestMetadata(&value), nil
	case domain.SoftwareApplicationDraftMetadata:
		value := softwareApplicationDraftMetadata(metadata)
		return client.SoftwareApplicationDraftMetadataAsCreateResourceRequestMetadata(&value), nil
	case domain.SoftwareSourceCodeDraftMetadata:
		value := softwareSourceCodeDraftMetadata(metadata)
		return client.SoftwareSourceCodeDraftMetadataAsCreateResourceRequestMetadata(&value), nil
	default:
		return client.CreateResourceRequestMetadata{}, fmt.Errorf("unsupported lifecycle metadata type %T", metadata)
	}
}

func replaceResourceDraftRequest(input domain.ReplaceResourceDraftInput) (client.ReplaceResourceDraftRequest, error) {
	switch metadata := input.Metadata.(type) {
	case domain.DatasetDraftMetadata:
		return client.ReplaceResourceDraftRequest{
			ReplaceDraftRequest: &client.ReplaceDraftRequest{
				Revision: input.Revision,
				Metadata: datasetDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareApplicationDraftMetadata:
		return client.ReplaceResourceDraftRequest{
			ReplaceSoftwareApplicationDraftRequest: &client.ReplaceSoftwareApplicationDraftRequest{
				Revision: input.Revision,
				Metadata: softwareApplicationDraftMetadata(metadata),
			},
		}, nil
	case domain.SoftwareSourceCodeDraftMetadata:
		return client.ReplaceResourceDraftRequest{
			ReplaceSoftwareSourceCodeDraftRequest: &client.ReplaceSoftwareSourceCodeDraftRequest{
				Revision: input.Revision,
				Metadata: softwareSourceCodeDraftMetadata(metadata),
			},
		}, nil
	default:
		return client.ReplaceResourceDraftRequest{}, fmt.Errorf("unsupported lifecycle metadata type %T", input.Metadata)
	}
}

func datasetDraftMetadata(metadata domain.DatasetDraftMetadata) client.DatasetDraftMetadata {
	// TODO: Expand Gateway lifecycle DTOs to cover every Backoffice dataset
	// field before enabling full-object create and replace requests.
	result := client.DatasetDraftMetadata{
		SourceIri:   metadata.SourceIRI,
		Title:       metadata.Title,
		Description: metadata.Description,
		Keywords:    metadata.Keywords,
	}
	if metadata.Description != nil {
		result.SetDescription(*metadata.Description)
	}
	if metadata.Keywords != nil {
		result.SetKeywords(*metadata.Keywords)
	}
	return result
}

func softwareApplicationDraftMetadata(metadata domain.SoftwareApplicationDraftMetadata) client.SoftwareApplicationDraftMetadata {
	result := client.SoftwareApplicationDraftMetadata{
		SourceIri:   metadata.SourceIRI,
		Name:        metadata.Name,
		Description: metadata.Description,
		Keywords:    metadata.Keywords,
	}
	if metadata.Description != nil {
		result.SetDescription(*metadata.Description)
	}
	if metadata.Keywords != nil {
		result.SetKeywords(*metadata.Keywords)
	}
	if metadata.DownloadURL != nil {
		result.SetDownloadUrl(*metadata.DownloadURL)
	}
	if metadata.MainEntityOfPage != nil {
		result.SetMainEntityOfPage(*metadata.MainEntityOfPage)
	}
	return result
}

func softwareSourceCodeDraftMetadata(metadata domain.SoftwareSourceCodeDraftMetadata) client.SoftwareSourceCodeDraftMetadata {
	result := client.SoftwareSourceCodeDraftMetadata{
		SourceIri:   metadata.SourceIRI,
		Name:        metadata.Name,
		Description: metadata.Description,
		Keywords:    metadata.Keywords,
	}
	if metadata.Description != nil {
		result.SetDescription(*metadata.Description)
	}
	if metadata.Keywords != nil {
		result.SetKeywords(*metadata.Keywords)
	}
	if metadata.Repository != nil {
		result.SetRepository(*metadata.Repository)
	}
	if metadata.ProgrammingLanguage != nil {
		result.SetProgrammingLanguage(*metadata.ProgrammingLanguage)
	}
	return result
}

func resourceVersion(record *client.ResourceRecord) (domain.ResourceVersion, error) {
	if record == nil {
		return domain.ResourceVersion{}, fmt.Errorf("decode backoffice resource record: response is empty")
	}

	result := domain.ResourceVersion{
		ResourceID: record.ResourceId,
		VersionID:  record.VersionId,
		Revision:   record.Revision,
		Status:     string(record.Status),
		CreatedBy:  record.CreatedBy,
		Group: domain.ResourceGroupReference{
			ID:   record.Group.Id,
			Name: record.Group.Name,
		},
	}

	switch {
	case record.Metadata.DatasetMetadata != nil:
		metadata := record.Metadata.DatasetMetadata
		result.Metadata = domain.DatasetResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Title:       metadata.Title,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
			Status:      string(metadata.Status),
			Created:     metadata.Created.Format(time.RFC3339Nano),
			Updated:     metadata.Updated.Format(time.RFC3339Nano),
		}
	case record.Metadata.SoftwareApplicationMetadata != nil:
		metadata := record.Metadata.SoftwareApplicationMetadata
		result.Metadata = domain.SoftwareApplicationResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Name:        metadata.Name,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
		}
	case record.Metadata.SoftwareSourceCodeMetadata != nil:
		metadata := record.Metadata.SoftwareSourceCodeMetadata
		result.Metadata = domain.SoftwareSourceCodeResourceMetadata{
			ID:          metadata.Id,
			SourceIRI:   metadata.SourceIri,
			Name:        metadata.Name,
			Description: metadata.Description,
			Keywords:    metadata.Keywords,
		}
	default:
		return domain.ResourceVersion{}, fmt.Errorf("decode backoffice resource metadata: response is empty")
	}

	return result, nil
}

func lifecycleError(response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	if response != nil {
		if response.Body != nil {
			response.Body.Close()
		}
		return &domain.UpstreamError{StatusCode: response.StatusCode}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("backoffice lifecycle request: %w", domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("backoffice lifecycle request: %w", err)
}
