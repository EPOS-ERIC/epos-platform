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

func (c *Converter) CreatePlugin(ctx context.Context, actor string, input domain.PluginInput) (domain.Plugin, error) {
	request, err := toProtoPluginInput(input)
	if err != nil {
		return domain.Plugin{}, err
	}

	response, err := c.client.CreatePlugin(converterActor(ctx, actor), &converterv1.CreatePluginRequest{Input: request})
	if err != nil {
		return domain.Plugin{}, c.wrapError(ctx, "create plugin", err)
	}

	return fromProtoPlugin(response.GetPlugin())
}

func (c *Converter) GetPlugin(ctx context.Context, actor string, id uuid.UUID) (domain.Plugin, error) {
	response, err := c.client.GetPlugin(converterActor(ctx, actor), &converterv1.GetPluginRequest{Id: id.String()})
	if err != nil {
		return domain.Plugin{}, c.wrapError(ctx, "get plugin", err)
	}

	return fromProtoPlugin(response.GetPlugin())
}

func (c *Converter) ListPlugins(ctx context.Context, actor string) ([]domain.Plugin, error) {
	response, err := c.client.ListPlugins(converterActor(ctx, actor), &converterv1.ListPluginsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list plugins", err)
	}

	plugins := make([]domain.Plugin, 0, len(response.GetPlugins()))
	for _, value := range response.GetPlugins() {
		plugin, err := fromProtoPlugin(value)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func (c *Converter) UpdatePlugin(ctx context.Context, actor string, id uuid.UUID, input domain.PluginInput, enabled bool) (domain.Plugin, error) {
	request, err := toProtoPluginInput(input)
	if err != nil {
		return domain.Plugin{}, err
	}

	response, err := c.client.UpdatePlugin(converterActor(ctx, actor), &converterv1.UpdatePluginRequest{Id: id.String(), Input: request, Enabled: enabled})
	if err != nil {
		return domain.Plugin{}, c.wrapError(ctx, "update plugin", err)
	}

	return fromProtoPlugin(response.GetPlugin())
}

func (c *Converter) DeletePlugin(ctx context.Context, actor string, id uuid.UUID) error {
	_, err := c.client.DeletePlugin(converterActor(ctx, actor), &converterv1.DeletePluginRequest{Id: id.String()})
	if err != nil {
		return c.wrapError(ctx, "delete plugin", err)
	}

	return nil
}

func (c *Converter) SyncPlugin(ctx context.Context, actor string, id uuid.UUID) (domain.Plugin, error) {
	response, err := c.client.SyncPlugin(converterActor(ctx, actor), &converterv1.SyncPluginRequest{Id: id.String()})
	if err != nil {
		return domain.Plugin{}, c.wrapError(ctx, "sync plugin", err)
	}

	return fromProtoPlugin(response.GetPlugin())
}

func (c *Converter) getRelation(ctx context.Context, actor string, request *converterv1.GetRelationRequest) (*converterv1.GetRelationResponse, error) {
	response, err := c.client.GetRelation(converterActor(ctx, actor), request)
	if err != nil {
		return nil, c.wrapError(ctx, "get relation", err)
	}

	return response, nil
}

func (c *Converter) ListRelations(ctx context.Context, actor string) ([]domain.Relation, error) {
	response, err := c.client.ListRelations(converterActor(ctx, actor), &converterv1.ListRelationsRequest{})
	if err != nil {
		return nil, c.wrapError(ctx, "list relations", err)
	}

	result := make([]domain.Relation, 0, len(response.GetRelations()))
	for _, relation := range response.GetRelations() {
		mapped, err := fromProtoRelation(relation)
		if err != nil {
			return nil, err
		}

		result = append(result, mapped)
	}

	return result, nil
}

func (c *Converter) CreateRelation(ctx context.Context, actor string, relation domain.Relation) (domain.Relation, error) {
	response, err := c.client.CreateRelation(converterActor(ctx, actor), &converterv1.CreateRelationRequest{Input: toProtoRelationInput(relation)})
	if err != nil {
		return domain.Relation{}, c.wrapError(ctx, "create relation", err)
	}

	return fromProtoRelation(response.GetRelation())
}

func (c *Converter) GetRelation(ctx context.Context, actor string, id uuid.UUID) (domain.Relation, error) {
	response, err := c.getRelation(ctx, actor, &converterv1.GetRelationRequest{Id: id.String()})
	if err != nil {
		return domain.Relation{}, err
	}

	return fromProtoRelation(response.GetRelation())
}

func (c *Converter) DistributionRelations(ctx context.Context, actor string, distributionID uuid.UUID) ([]domain.DistributionPluginRelation, error) {
	response, err := c.client.ListDistributionRelations(converterActor(ctx, actor), &converterv1.ListDistributionRelationsRequest{DistributionId: distributionID.String()})
	if err != nil {
		return nil, c.wrapError(ctx, "list distribution relations", err)
	}

	result := make([]domain.DistributionPluginRelation, 0, len(response.GetRelations()))
	for _, item := range response.GetRelations() {
		p, r := item.GetPlugin(), item.GetRelation()
		if p == nil || p.GetInput() == nil || r == nil || r.GetInput() == nil {
			return nil, apperror.Default(apperror.CodeInternal)
		}

		pid, err := uuid.Parse(p.GetId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}

		rid, err := uuid.Parse(r.GetId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}

		did, err := uuid.Parse(r.GetInput().GetDistributionId())
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "", err)
		}

		if did != distributionID || r.GetInput().GetPluginId() != p.GetId() {
			return nil, apperror.Default(apperror.CodeInternal)
		}

		runtime, err := fromProtoRuntime(p.GetInput().GetRuntime())
		if err != nil {
			return nil, apperror.Default(apperror.CodeInternal)
		}

		result = append(result, domain.DistributionPluginRelation{
			PluginID:       pid,
			Name:           p.GetInput().GetName(),
			Description:    p.GetInput().GetDescription(),
			Repository:     p.GetInput().GetRepository(),
			Branch:         p.GetInput().GetBranch(),
			Runtime:        runtime,
			Arguments:      p.GetInput().GetArguments(),
			Enabled:        p.GetEnabled(),
			Installed:      p.GetInstalled(),
			RelationID:     rid,
			DistributionID: did,
			InputFormat:    r.GetInput().GetInputFormat(),
			OutputFormat:   r.GetInput().GetOutputFormat(),
		})
	}

	return result, nil
}

func (c *Converter) UpdateRelation(ctx context.Context, actor string, id uuid.UUID, relation domain.Relation) (domain.Relation, error) {
	response, err := c.client.UpdateRelation(converterActor(ctx, actor), &converterv1.UpdateRelationRequest{
		Id:    id.String(),
		Input: toProtoRelationInput(relation),
	})
	if err != nil {
		return domain.Relation{}, c.wrapError(ctx, "update relation", err)
	}

	return fromProtoRelation(response.GetRelation())
}

func (c *Converter) DeleteRelation(ctx context.Context, actor string, id uuid.UUID) error {
	_, err := c.client.DeleteRelation(converterActor(ctx, actor), &converterv1.DeleteRelationRequest{Id: id.String()})
	if err != nil {
		return c.wrapError(ctx, "delete relation", err)
	}

	return nil
}

// Conversion returns the minimal relation data needed by distribution access.
func (c *Converter) Conversion(ctx context.Context, id uuid.UUID) (domain.ConversionRelation, error) {
	response, err := c.getRelation(ctx, "", &converterv1.GetRelationRequest{Id: id.String()})
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
		ID:             relationID,
		DistributionID: distributionID,
		InputFormat:    relation.GetInput().GetInputFormat(),
	}, nil
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
