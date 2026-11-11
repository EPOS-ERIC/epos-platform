package clients

import (
	"context"

	pb "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) ListConcepts(ctx context.Context) (*pb.ListConceptsResponse, error) {
	result, err := c.concepts.ListConcepts(ctx, &pb.ListConceptsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list concepts", err)
	}

	return result, nil
}

func (c *MetadataLifecycle) GetConcept(ctx context.Context, id string) (*pb.GetConceptResponse, error) {
	result, err := c.concepts.GetConcept(ctx, &pb.GetConceptRequest{Id: id})
	if err != nil {
		return nil, c.wrapError(ctx, "get concept", err)
	}

	return result, nil
}

func (c *MetadataLifecycle) CreateConcept(ctx context.Context, actor, name string) (*pb.CreateConceptResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actor)
	result, err := c.concepts.CreateConcept(ctx, &pb.CreateConceptRequest{Name: name})
	if err != nil {
		return nil, c.wrapError(ctx, "create concept", err)
	}

	return result, nil
}

func (c *MetadataLifecycle) ReplaceConcept(ctx context.Context, actor, id, name string) (*pb.ReplaceConceptResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actor)
	result, err := c.concepts.ReplaceConcept(ctx, &pb.ReplaceConceptRequest{Id: id, Name: name})
	if err != nil {
		return nil, c.wrapError(ctx, "replace concept", err)
	}

	return result, nil
}

func (c *MetadataLifecycle) DeleteConcept(ctx context.Context, actor, id string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actor)
	_, err := c.concepts.DeleteConcept(ctx, &pb.DeleteConceptRequest{Id: id})
	if err != nil {
		return c.wrapError(ctx, "delete concept", err)
	}

	return nil
}
