package grpcapi

import (
	"context"
	"database/sql"
	"errors"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Metadata keys carrying the caller's identity. The gateway is the only
// client and attaches them after validating the bearer token, mirroring the
// metadata-lifecycle actor convention.
const (
	userIDMetadataKey = "x-epos-user-id"
	adminMetadataKey  = "x-epos-admin"
)

// errNotFound marks a missing configuration. It is also returned when the
// caller may not access a configuration, so private identifiers are not
// leaked to unauthorized callers.
var errNotFound = errors.New("configuration not found")

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Caller is the authenticated identity attached by the gateway. Anonymous
// callers have no Caller and may only access public configurations.
type Caller struct {
	UserID string
	Admin  bool
}

func callerFromContext(ctx context.Context) (Caller, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Caller{}, false
	}
	values := md.Get(userIDMetadataKey)
	if len(values) != 1 || values[0] == "" {
		return Caller{}, false
	}
	caller := Caller{UserID: values[0]}
	for _, value := range md.Get(adminMetadataKey) {
		if value == "true" {
			caller.Admin = true
			break
		}
	}
	return caller, true
}

// Server implements ConfigurationService. Handlers are thin: transport
// mapping, domain validation, repository call, status mapping. Visibility and
// ownership policy is enforced here so it holds regardless of the caller.
type Server struct {
	configurationv1.UnimplementedConfigurationServiceServer
	repo repository.ConfigurationRepository
}

func New(repo repository.ConfigurationRepository) *Server {
	return &Server{repo: repo}
}

func (s *Server) CreateConfiguration(ctx context.Context, request *configurationv1.CreateConfigurationRequest) (*configurationv1.CreateConfigurationResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	if request.GetIsOfficial() && !caller.Admin {
		return nil, status.Error(codes.PermissionDenied, "only administrators may create official configurations")
	}

	id := uuid.New()
	if request.Id != nil {
		parsed, err := uuid.Parse(request.GetId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "id must be a UUID")
		}
		id = parsed
	}

	input, err := toDomainConfigurationInput(request.GetConfiguration(), request.Title, request.Description, request.GetListOfServices(), request.GetIsOfficial(), resolveIsPublic(request.IsPublic))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record, err := s.repo.Create(ctx, id, caller.UserID, input)
	if isDuplicateKey(err) {
		return nil, status.Error(codes.AlreadyExists, "configuration already exists")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "create configuration")
	}
	return &configurationv1.CreateConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) GetConfiguration(ctx context.Context, request *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "id must be a UUID")
	}

	caller, _ := callerFromContext(ctx)
	record, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, errNotFound.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get configuration")
	}
	if !canRead(caller, record) {
		return nil, status.Error(codes.NotFound, errNotFound.Error())
	}
	return &configurationv1.GetConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) ListConfigurations(ctx context.Context, request *configurationv1.ListConfigurationsRequest) (*configurationv1.ListConfigurationsResponse, error) {
	pageSize := defaultPageSize
	var offset int
	var official *bool
	if request != nil {
		if request.PageSize != nil {
			pageSize = int(request.GetPageSize())
		}
		if request.Offset != nil {
			offset = int(request.GetOffset())
		}
		official = request.Official
	}
	if pageSize < 1 || pageSize > maxPageSize {
		return nil, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}
	if offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "offset must not be negative")
	}

	caller, ok := callerFromContext(ctx)
	filter := repository.ListFilter{Limit: pageSize, Offset: offset, Official: official}
	if ok {
		filter.CallerID = &caller.UserID
		filter.IncludeAll = caller.Admin
	}

	records, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "list configurations")
	}
	return &configurationv1.ListConfigurationsResponse{Configurations: toProtoConfigurations(records), Total: total}, nil
}

func (s *Server) UpdateConfiguration(ctx context.Context, request *configurationv1.UpdateConfigurationRequest) (*configurationv1.UpdateConfigurationResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "id must be a UUID")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, errNotFound.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get configuration")
	}
	if !canWrite(caller, existing) {
		if !canRead(caller, existing) {
			return nil, status.Error(codes.NotFound, errNotFound.Error())
		}
		return nil, status.Error(codes.PermissionDenied, "only the owner or an administrator may update this configuration")
	}
	if request.GetIsOfficial() != existing.IsOfficial && !caller.Admin {
		return nil, status.Error(codes.PermissionDenied, "only administrators may change the official flag")
	}

	input, err := toDomainConfigurationInput(request.GetConfiguration(), request.Title, request.Description, request.GetListOfServices(), request.GetIsOfficial(), resolveIsPublic(request.IsPublic))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record, err := s.repo.Update(ctx, id, input)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, errNotFound.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "update configuration")
	}
	return &configurationv1.UpdateConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) DeleteConfiguration(ctx context.Context, request *configurationv1.DeleteConfigurationRequest) (*configurationv1.DeleteConfigurationResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	caller, ok := callerFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "id must be a UUID")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, errNotFound.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get configuration")
	}
	if !canWrite(caller, existing) {
		if !canRead(caller, existing) {
			return nil, status.Error(codes.NotFound, errNotFound.Error())
		}
		return nil, status.Error(codes.PermissionDenied, "only the owner or an administrator may delete this configuration")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, errNotFound.Error())
		}
		return nil, status.Error(codes.Internal, "delete configuration")
	}
	return &configurationv1.DeleteConfigurationResponse{}, nil
}

// canRead reports whether the caller (zero value when anonymous) may read the
// record: public configurations are visible to everyone, private ones only to
// their owner or an administrator.
func canRead(caller Caller, record repository.ConfigurationRecord) bool {
	if record.IsPublic {
		return true
	}
	return caller.UserID != "" && (caller.UserID == record.OwnerID || caller.Admin)
}

// canWrite reports whether the caller may replace or delete the record: the
// owner or an administrator. Callers are always authenticated here.
func canWrite(caller Caller, record repository.ConfigurationRecord) bool {
	return caller.UserID == record.OwnerID || caller.Admin
}

// resolveIsPublic maps an absent visibility flag to public.
func resolveIsPublic(isPublic *bool) bool {
	if isPublic == nil {
		return true
	}
	return *isPublic
}

// isDuplicateKey reports PostgreSQL unique violations (SQLSTATE 23505),
// e.g. creating a configuration with an explicitly provided id that already
// exists.
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
