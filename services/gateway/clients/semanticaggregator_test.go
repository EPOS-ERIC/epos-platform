package clients

import (
	"context"
	"io"
	"log/slog"
	"testing"

	semanticaggregatorv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/semantic_aggregator/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestSemanticAggregatorImportResourcesForwardsActorMetadata(t *testing.T) {
	stub := &resourceImportStub{}
	client := &SemanticAggregator{
		log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		imports: stub,
	}
	request := &semanticaggregatorv1.ImportResourcesRequest{GroupId: "group-1"}
	if _, err := client.ImportResources(context.Background(), "actor-1", request); err != nil {
		t.Fatalf("ImportResources() error = %v", err)
	}
	if got, want := stub.request, request; got != want {
		t.Errorf("request = %p, want %p", got, want)
	}
	if got, want := stub.actor, "actor-1"; got != want {
		t.Errorf("actor metadata = %q, want %q", got, want)
	}
}

type resourceImportStub struct {
	request *semanticaggregatorv1.ImportResourcesRequest
	actor   string
}

func (s *resourceImportStub) ImportResources(ctx context.Context, request *semanticaggregatorv1.ImportResourcesRequest, _ ...grpc.CallOption) (*semanticaggregatorv1.ImportResourcesResponse, error) {
	s.request = request
	metadata, _ := metadata.FromOutgoingContext(ctx)
	s.actor = metadata.Get(actorMetadataKey)[0]
	return &semanticaggregatorv1.ImportResourcesResponse{}, nil
}
