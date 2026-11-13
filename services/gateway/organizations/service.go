// Package organizations owns Gateway organization operations and management policy.
package organizations

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Client interface {
	ListOrganizations(context.Context) (*pb.ListOrganizationsResponse, error)
	GetOrganization(context.Context, string) (*pb.GetOrganizationResponse, error)
	CreateOrganization(context.Context, string, domain.OrganizationInput) (*pb.Organization, error)
	ReplaceOrganization(context.Context, string, string, domain.OrganizationInput) (*pb.Organization, error)
	DeleteOrganization(context.Context, string, string) error
}

type Service struct {
	client Client
	synced bool
}

func New(client Client, synced bool) *Service { return &Service{client: client, synced: synced} }

// List passes protobuf through unchanged; no response orchestration.
func (s *Service) List(ctx context.Context) (*pb.ListOrganizationsResponse, error) {
	return s.client.ListOrganizations(ctx)
}

// Get passes protobuf through unchanged; no response orchestration.
func (s *Service) Get(ctx context.Context, id string) (*pb.GetOrganizationResponse, error) {
	return s.client.GetOrganization(ctx, id)
}

// Create passes protobuf through unchanged; no response orchestration.
func (s *Service) Create(ctx context.Context, actor string, input domain.OrganizationInput) (*pb.Organization, error) {
	if s.synced {
		return nil, apperror.New(apperror.CodePermissionDenied, "Organization management is disabled in synced mode.")
	}

	return s.client.CreateOrganization(ctx, actor, input)
}

// Replace passes protobuf through unchanged; no response orchestration.
func (s *Service) Replace(ctx context.Context, actor, id string, input domain.OrganizationInput) (*pb.Organization, error) {
	if s.synced {
		return nil, apperror.New(apperror.CodePermissionDenied, "Organization management is disabled in synced mode.")
	}

	return s.client.ReplaceOrganization(ctx, actor, id, input)
}

func (s *Service) Delete(ctx context.Context, actor, id string) error {
	if s.synced {
		return apperror.New(apperror.CodePermissionDenied, "Organization management is disabled in synced mode.")
	}

	return s.client.DeleteOrganization(ctx, actor, id)
}
