package grpcapi

import (
	"context"

	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/concepts"
)

type ConceptServer struct {
	pb.UnimplementedConceptServiceServer
	concepts *concepts.Service
}

func NewConcept(concepts *concepts.Service) *ConceptServer {
	return &ConceptServer{concepts: concepts}
}

func (g *ConceptServer) ListConcepts(ctx context.Context, _ *pb.ListConceptsRequest) (*pb.ListConceptsResponse, error) {
	values, err := g.concepts.ListConcepts(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*pb.Concept, 0, len(values))
	for _, value := range values {
		result = append(result, toProtoConcept(value))
	}

	return &pb.ListConceptsResponse{Concepts: result}, nil
}

func (g *ConceptServer) GetConcept(ctx context.Context, request *pb.GetConceptRequest) (*pb.GetConceptResponse, error) {
	id, err := parseConceptID(request.GetId())
	if err != nil {
		return nil, err
	}

	value, err := g.concepts.GetConcept(ctx, id)
	if err != nil {
		return nil, err
	}

	return &pb.GetConceptResponse{Concept: toProtoConcept(value)}, nil
}

func (g *ConceptServer) CreateConcept(ctx context.Context, request *pb.CreateConceptRequest) (*pb.CreateConceptResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	value, err := g.concepts.CreateConcept(ctx, actor, request.GetName())
	if err != nil {
		return nil, err
	}

	return &pb.CreateConceptResponse{Concept: toProtoConcept(value)}, nil
}

func (g *ConceptServer) ReplaceConcept(ctx context.Context, request *pb.ReplaceConceptRequest) (*pb.ReplaceConceptResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseConceptID(request.GetId())
	if err != nil {
		return nil, err
	}

	value, err := g.concepts.ReplaceConcept(ctx, actor, id, request.GetName())
	if err != nil {
		return nil, err
	}

	return &pb.ReplaceConceptResponse{Concept: toProtoConcept(value)}, nil
}

func (g *ConceptServer) DeleteConcept(ctx context.Context, request *pb.DeleteConceptRequest) (*pb.DeleteConceptResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseConceptID(request.GetId())
	if err != nil {
		return nil, err
	}

	if err := g.concepts.DeleteConcept(ctx, actor, id); err != nil {
		return nil, err
	}

	return &pb.DeleteConceptResponse{}, nil
}
