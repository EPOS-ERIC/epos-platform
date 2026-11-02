package clients

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Converter struct {
	log    *slog.Logger
	client converterv1.ConverterServiceClient
}

func NewConverter(log *slog.Logger, address string) (*Converter, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return nil, fmt.Errorf("create converter gRPC client: %w", err)
	}
	return &Converter{log: log, client: converterv1.NewConverterServiceClient(conn)}, nil
}

func converterActor(ctx context.Context, actor string) context.Context {
	if actor == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-epos-user-id", actor)
}

func (c *Converter) wrapError(ctx context.Context, operation string, err error) error {
	return wrapRPCError(c.log, ctx, "converter", operation, err)
}

func (c *Converter) CreatePlugin(ctx context.Context, actor string, request *converterv1.CreatePluginRequest) (*converterv1.CreatePluginResponse, error) {
	response, err := c.client.CreatePlugin(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "create plugin", err)
	}
	return response, nil
}

func (c *Converter) GetPlugin(ctx context.Context, actor string, request *converterv1.GetPluginRequest) (*converterv1.GetPluginResponse, error) {
	response, err := c.client.GetPlugin(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "get plugin", err)
	}
	return response, nil
}

func (c *Converter) ListPlugins(ctx context.Context, actor string) (*converterv1.ListPluginsResponse, error) {
	response, err := c.client.ListPlugins(converterActor(ctx, actor), &converterv1.ListPluginsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list plugins", err)
	}
	return response, nil
}

func (c *Converter) UpdatePlugin(ctx context.Context, actor string, request *converterv1.UpdatePluginRequest) (*converterv1.UpdatePluginResponse, error) {
	response, err := c.client.UpdatePlugin(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "update plugin", err)
	}
	return response, nil
}

func (c *Converter) DeletePlugin(ctx context.Context, actor string, request *converterv1.DeletePluginRequest) error {
	_, err := c.client.DeletePlugin(converterActor(ctx, actor), request)
	if err != nil {
		return c.wrapError(ctx, "delete plugin", err)
	}
	return nil
}

func (c *Converter) SyncPlugin(ctx context.Context, actor string, request *converterv1.SyncPluginRequest) (*converterv1.SyncPluginResponse, error) {
	response, err := c.client.SyncPlugin(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "sync plugin", err)
	}
	return response, nil
}

func (c *Converter) CreateRelation(ctx context.Context, actor string, request *converterv1.CreateRelationRequest) (*converterv1.CreateRelationResponse, error) {
	response, err := c.client.CreateRelation(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "create relation", err)
	}
	return response, nil
}

func (c *Converter) GetRelation(ctx context.Context, actor string, request *converterv1.GetRelationRequest) (*converterv1.GetRelationResponse, error) {
	response, err := c.client.GetRelation(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "get relation", err)
	}
	return response, nil
}

func (c *Converter) ListRelations(ctx context.Context, actor string, request *converterv1.ListRelationsRequest) (*converterv1.ListRelationsResponse, error) {
	response, err := c.client.ListRelations(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "list relations", err)
	}
	return response, nil
}

func (c *Converter) UpdateRelation(ctx context.Context, actor string, request *converterv1.UpdateRelationRequest) (*converterv1.UpdateRelationResponse, error) {
	response, err := c.client.UpdateRelation(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "update relation", err)
	}
	return response, nil
}

func (c *Converter) DeleteRelation(ctx context.Context, actor string, request *converterv1.DeleteRelationRequest) error {
	_, err := c.client.DeleteRelation(converterActor(ctx, actor), request)
	if err != nil {
		return c.wrapError(ctx, "delete relation", err)
	}
	return nil
}

// Conversion returns the minimal relation data needed by distribution access.
func (c *Converter) Conversion(ctx context.Context, id uuid.UUID) (domain.ConversionRelation, error) {
	response, err := c.GetRelation(ctx, "", &converterv1.GetRelationRequest{Id: id.String()})
	if err != nil {
		return domain.ConversionRelation{}, err
	}
	relation := response.GetRelation()
	if relation == nil || relation.GetInput() == nil {
		return domain.ConversionRelation{}, apperror.Default(apperror.CodeInternal)
	}
	relationID, err := uuid.Parse(relation.GetId())
	if err != nil {
		return domain.ConversionRelation{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	distributionID, err := uuid.Parse(relation.GetInput().GetDistributionId())
	if err != nil {
		return domain.ConversionRelation{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	return domain.ConversionRelation{
		ID: relationID, DistributionID: distributionID,
		InputFormat: relation.GetInput().GetInputFormat(), OutputFormat: relation.GetInput().GetOutputFormat(),
	}, nil
}

func (c *Converter) AvailableConversions(ctx context.Context) ([]domain.AvailableConversion, error) {
	plugins, err := c.ListPlugins(ctx, "")
	if err != nil {
		return nil, err
	}

	relations, err := c.ListRelations(ctx, "", &converterv1.ListRelationsRequest{})
	if err != nil {
		return nil, err
	}
	available := make(map[string]*converterv1.Plugin)
	for _, p := range plugins.GetPlugins() {
		if p != nil && p.GetEnabled() && p.GetInstalled() && p.GetInput() != nil {
			available[p.GetId()] = p
		}
	}
	result := make([]domain.AvailableConversion, 0, len(relations.GetRelations()))
	for _, relation := range relations.GetRelations() {
		if relation == nil || relation.GetInput() == nil {
			return nil, apperror.Default(apperror.CodeInternal)
		}
		input := relation.GetInput()
		p := available[input.GetPluginId()]
		if p == nil {
			continue
		}
		relationID, err := uuid.Parse(relation.GetId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}
		distributionID, err := uuid.Parse(input.GetDistributionId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}
		pluginID, err := uuid.Parse(p.GetId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}
		result = append(result, domain.AvailableConversion{
			ID:                relationID,
			DistributionID:    distributionID,
			InputFormat:       input.GetInputFormat(),
			OutputFormat:      input.GetOutputFormat(),
			PluginID:          pluginID,
			PluginName:        p.GetInput().GetName(),
			PluginDescription: p.GetInput().GetDescription(),
		})
	}
	return result, nil
}

func (c *Converter) Execute(ctx context.Context, conversionID, distributionID uuid.UUID, sourceURL string, input io.Reader) (io.ReadCloser, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.client.Execute(streamCtx)
	if err != nil {
		cancel()
		return nil, c.wrapError(ctx, "start conversion", err)
	}
	send := func(request *converterv1.ExecuteRequest) error {
		if err := stream.Send(request); err != nil {
			if err == io.EOF {
				_, err = stream.Recv()
			}
			return c.wrapError(ctx, "send conversion input", err)
		}
		return nil
	}
	if err := send(&converterv1.ExecuteRequest{
		Event: &converterv1.ExecuteRequest_Header{
			Header: &converterv1.ExecuteHeader{
				ConversionId:   conversionID.String(),
				DistributionId: distributionID.String(),
				SourceUrl:      sourceURL,
			},
		},
	}); err != nil {
		cancel()
		return nil, err
	}
	buffer := make([]byte, 64*1024)
	for {
		n, readErr := input.Read(buffer)
		if n > 0 {
			if err := send(&converterv1.ExecuteRequest{Event: &converterv1.ExecuteRequest_Chunk{Chunk: buffer[:n]}}); err != nil {
				cancel()
				return nil, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			cancel()
			return nil, readErr
		}
	}
	if err := stream.CloseSend(); err != nil {
		cancel()
		return nil, c.wrapError(ctx, "finish conversion input", err)
	}
	first, err := stream.Recv()
	if err != nil {
		cancel()
		if err == io.EOF {
			return nil, apperror.Default(apperror.CodeInternal)
		}
		return nil, c.wrapError(ctx, "execute conversion", err)
	}
	if len(first.GetChunk()) == 0 {
		cancel()
		return nil, apperror.Default(apperror.CodeInternal)
	}
	return &conversionBody{
		stream:  stream,
		pending: first.GetChunk(),
		cancel:  cancel,
		client:  c,
	}, nil
}

type conversionBody struct {
	stream  grpc.BidiStreamingClient[converterv1.ExecuteRequest, converterv1.ExecuteResponse]
	pending []byte
	cancel  context.CancelFunc
	client  *Converter
}

func (b *conversionBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(b.pending) == 0 {
		response, err := b.stream.Recv()
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, b.client.wrapError(b.stream.Context(), "read conversion output", err)
		}
		b.pending = response.GetChunk()
		if len(b.pending) == 0 {
			return 0, apperror.Default(apperror.CodeInternal)
		}
	}
	n := copy(p, b.pending)
	b.pending = b.pending[n:]
	return n, nil
}

func (b *conversionBody) Close() error {
	b.cancel()
	return nil
}
