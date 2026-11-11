package grpcapi

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toProtoConcept(value domain.Concept) *pb.Concept {
	return &pb.Concept{
		Id:   value.ID.String(),
		Name: value.Name,
	}
}

func parseConceptID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.Field("id", "Concept ID must be a valid UUID.", err)
	}

	return id, nil
}
