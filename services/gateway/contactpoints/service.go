// Package contactpoints owns Gateway contact-point management policy.
package contactpoints

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
)

type Service struct {
	client *clients.MetadataLifecycle
	synced bool
}

func New(client *clients.MetadataLifecycle, synced bool) *Service {
	return &Service{
		client: client,
		synced: synced,
	}
}

// Create passes protobuf through unchanged; no response orchestration.
func (s *Service) Create(ctx context.Context, actor, email string) (*pb.ContactPoint, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	return s.client.CreateContactPoint(ctx, actor, email)
}

// Replace passes protobuf through unchanged; no response orchestration.
func (s *Service) Replace(ctx context.Context, actor, id, email string) (*pb.ContactPoint, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	return s.client.ReplaceContactPoint(ctx, actor, id, email)
}

func (s *Service) Delete(ctx context.Context, actor, id string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	return s.client.DeleteContactPoint(ctx, actor, id)
}

func (s *Service) checkWritable() error {
	if s.synced {
		return apperror.New(apperror.CodePermissionDenied, "Contact point management is disabled in synced mode.")
	}

	return nil
}
