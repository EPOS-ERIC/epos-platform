// Package imports contains metadata import use-cases.
package imports

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	backofficeclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

type Service struct {
	imports backofficeclient.ImportsAPI
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{imports: backoffice.Imports()}
}

type ResponseError struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("backoffice import request failed with status %d", e.StatusCode)
}

func (s *Service) Turtle(ctx context.Context, actorID string, body io.Reader) (*backofficeclient.ImportResult, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read import body: %w: %v", domain.ErrBadRequest, err)
	}
	result, response, err := s.imports.ImportTurtleMetadata(ctx).XEPOSUserID(actorID).Body(string(content)).Execute()
	if err != nil {
		if response != nil {
			defer response.Body.Close()
			responseBody, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				return nil, fmt.Errorf("read backoffice import error response: %w", readErr)
			}
			return nil, &ResponseError{
				StatusCode:  response.StatusCode,
				ContentType: response.Header.Get("Content-Type"),
				Body:        responseBody,
			}
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("backoffice import request: %w", domain.ErrUpstreamTimeout)
		}
		return nil, fmt.Errorf("backoffice import request: %w", err)
	}
	return result, nil
}
