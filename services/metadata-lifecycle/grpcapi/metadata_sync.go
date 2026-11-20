package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadatasync"
)

type MetadataSyncServer struct {
	pb.UnimplementedMetadataSyncServiceServer
	sync *metadatasync.Service
}

func NewMetadataSync(sync *metadatasync.Service) *MetadataSyncServer {
	return &MetadataSyncServer{sync: sync}
}

func (s *MetadataSyncServer) SyncMetadata(ctx context.Context, request *pb.SyncMetadataRequest) (*pb.SyncMetadataResponse, error) {
	orgs, contacts, err := toDomainSnapshots(request)
	if err != nil {
		return nil, err
	}

	if err := s.sync.Sync(ctx, orgs, contacts); err != nil {
		return nil, err
	}

	return &pb.SyncMetadataResponse{}, nil
}

func toDomainSnapshots(request *pb.SyncMetadataRequest) ([]domain.SyncedOrganization, []domain.SyncedContactPoint, error) {
	orgs := make([]domain.SyncedOrganization, 0, len(request.Organizations.Items))
	for i, value := range request.Organizations.Items {
		input, err := toOrganizationInput(value.Name, value.Street, value.Locality, value.PostalCode, value.Country, value.LogoUrl, value.PageUrl, value.Email, value.Identifiers, nil)
		if err != nil {
			return nil, nil, apperror.PrefixTarget(err, fmt.Sprintf("organizations.items[%d]", i))
		}

		orgs = append(orgs, domain.SyncedOrganization{
			NID:        value.Nid,
			ParentNIDs: value.ParentNids,
			Input:      input,
		})
	}

	contacts := make([]domain.SyncedContactPoint, 0, len(request.ContactPoints.Items))
	for i, value := range request.ContactPoints.Items {
		input := domain.ContactPointInput{Email: value.Email}
		if err := input.Normalize(); err != nil {
			return nil, nil, apperror.PrefixTarget(err, fmt.Sprintf("contact_points.items[%d]", i))
		}

		contacts = append(contacts, domain.SyncedContactPoint{
			NID:   value.Nid,
			Input: input,
		})
	}

	return orgs, contacts, nil
}
