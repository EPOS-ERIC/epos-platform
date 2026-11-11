package handlers

import (
	"fmt"

	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	api "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toAPIConcept(value *pb.Concept) (*api.Concept, error) {
	if value == nil {
		return nil, fmt.Errorf("concept response is empty")
	}

	id, err := parseUUID("concept id", value.Id)
	if err != nil {
		return nil, err
	}

	return &api.Concept{
		ID:   id,
		Name: value.Name,
	}, nil
}
