// Package lifecycle contains resource lifecycle use-cases.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	backofficeclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

type Service struct {
	resources backofficeclient.ResourcesAPI
}

// NewService builds lifecycle service for Backoffice lifecycle operations.
func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{resources: backoffice.Resources()}
}

// ResponseError preserves a Backoffice API error response for the HTTP adapter.
type ResponseError struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("backoffice lifecycle request failed with status %d", e.StatusCode)
}

func (s *Service) Create(ctx context.Context, actorID, groupID, resourceType string, request backofficeclient.CreateResourceRequest) (*backofficeclient.ResourceRecord, error) {
	request.GroupId = groupID
	return resourceResult(s.resources.CreateResource(ctx, resourceType).XEPOSUserID(actorID).CreateResourceRequest(request).Execute())
}

func (s *Service) Get(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	request := s.resources.GetResourceVersion(ctx, resourceType, id, versionID)
	if actorID != "" {
		request = request.XEPOSUserID(actorID)
	}
	return resourceResult(request.Execute())
}

func (s *Service) ReplaceDraft(ctx context.Context, actorID, resourceType, id, versionID string, request backofficeclient.ReplaceResourceDraftRequest) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.ReplaceResourceDraft(ctx, resourceType, id, versionID).XEPOSUserID(actorID).ReplaceResourceDraftRequest(request).Execute())
}

func (s *Service) Discard(ctx context.Context, actorID, resourceType, id, versionID string) error {
	response, err := s.resources.DiscardResourceDraft(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute()
	return responseError(response, err)
}

func (s *Service) Submit(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.SubmitResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute())
}

func (s *Service) Withdraw(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.WithdrawResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute())
}

func (s *Service) Reject(ctx context.Context, actorID, resourceType, id, versionID string, request backofficeclient.RejectRequest) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.RejectResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).RejectRequest(request).Execute())
}

func (s *Service) Reopen(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.ReopenResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute())
}

func (s *Service) Publish(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.PublishResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute())
}

func (s *Service) Unpublish(ctx context.Context, actorID, resourceType, id, versionID string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.UnpublishResource(ctx, resourceType, id, versionID).XEPOSUserID(actorID).Execute())
}

func (s *Service) DraftFromPublication(ctx context.Context, actorID, resourceType, id string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.CreateResourceDraftFromPublication(ctx, resourceType, id).XEPOSUserID(actorID).Execute())
}

func (s *Service) DraftFromArchive(ctx context.Context, actorID, resourceType, id string) (*backofficeclient.ResourceRecord, error) {
	return resourceResult(s.resources.CreateResourceDraftFromArchive(ctx, resourceType, id).XEPOSUserID(actorID).Execute())
}

func (s *Service) Copy(ctx context.Context, actorID, groupID, resourceType, id string, request backofficeclient.CopyResourceRequest) (*backofficeclient.ResourceRecord, error) {
	request.TargetGroupId = groupID
	return resourceResult(s.resources.CopyResource(ctx, resourceType, id).XEPOSUserID(actorID).CopyResourceRequest(request).Execute())
}

func resourceResult(record *backofficeclient.ResourceRecord, response *http.Response, err error) (*backofficeclient.ResourceRecord, error) {
	if err := responseError(response, err); err != nil {
		return nil, err
	}
	return record, nil
}

func responseError(response *http.Response, err error) error {
	if err == nil {
		return nil
	}
	if response != nil {
		body, readErr := responseBody(response)
		if readErr != nil {
			return fmt.Errorf("read backoffice lifecycle error response: %w", readErr)
		}
		return &ResponseError{
			StatusCode:  response.StatusCode,
			ContentType: response.Header.Get("Content-Type"),
			Body:        body,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("backoffice lifecycle request: %w", domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("backoffice lifecycle request: %w", err)
}

func responseBody(response *http.Response) ([]byte, error) {
	if response.Body == nil {
		return nil, nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}
