package retrievals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	retrievalsqlc "github.com/EPOS-ERIC/epos-platform/services/data-retriever/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db      *pgxpool.Pool
	queries *retrievalsqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool, queries: retrievalsqlc.New(pool)}
}

// CreatePlaceholder persists retrieval metadata before streaming starts.
func (r *Repo) CreatePlaceholder(ctx context.Context, req domain.PreparedRetrieval, head domain.ResponseHead) (domain.RetrievalRecord, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.RetrievalRecord{}, fmt.Errorf("begin retrieval transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	queries := r.queries.WithTx(tx)

	if err := ensureOutputSchema(ctx, queries, req.ResourceID, req.VersionID); err != nil {
		return domain.RetrievalRecord{}, err
	}

	params, err := upsertRetrievalParams(req, head, max(head.ContentLength, -1))
	if err != nil {
		return domain.RetrievalRecord{}, err
	}
	retrieval, err := queries.UpsertRetrieval(ctx, params)
	if err != nil {
		return domain.RetrievalRecord{}, fmt.Errorf("upsert retrieval: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RetrievalRecord{}, fmt.Errorf("commit retrieval transaction: %w", err)
	}
	return toRetrievalRecordFromGet(retrievalsqlc.GetRetrievalByIDRow(retrieval)), nil
}

func (r *Repo) FinalizeRetrieval(ctx context.Context, id int64, payloadHash string, payloadSize int64) error {
	if err := r.queries.FinalizeRetrieval(ctx, retrievalsqlc.FinalizeRetrievalParams{
		ID:               id,
		PayloadHash:      payloadHash,
		PayloadSizeBytes: payloadSize,
	}); err != nil {
		return fmt.Errorf("finalize retrieval: %w", err)
	}
	return nil
}

func (r *Repo) GetByID(ctx context.Context, id int64) (domain.RetrievalRecord, error) {
	retrieval, err := r.queries.GetRetrievalByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RetrievalRecord{}, apperror.Default(apperror.CodeNotFound)
	}
	if err != nil {
		return domain.RetrievalRecord{}, fmt.Errorf("get retrieval by ID: %w", apperror.Wrap(apperror.CodeInternal, "", err))
	}

	record := toRetrievalRecordFromGet(retrieval)
	if retrieval.ResourceID == nil || retrieval.VersionID == nil {
		return record, nil
	}
	parameters, err := r.queries.GetOutputParameters(ctx, retrievalsqlc.GetOutputParametersParams{
		ResourceID: *retrieval.ResourceID,
		VersionID:  *retrieval.VersionID,
	})
	if err != nil {
		return domain.RetrievalRecord{}, fmt.Errorf("get output parameters: %w", apperror.Wrap(apperror.CodeInternal, "", err))
	}
	record.OutputParameters = toDomainOutputParameters(parameters)
	return record, nil
}

// ReplaceOutputParameters atomically replaces the schema for a resource version.
func (r *Repo) ReplaceOutputParameters(ctx context.Context, resourceID, versionID uuid.UUID, parameters []domain.OutputParameter) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin output parameters transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	queries := r.queries.WithTx(tx)

	if err := replaceOutputSchema(ctx, queries, resourceID, versionID, parameters); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit output parameters transaction: %w", err)
	}
	return nil
}

func ensureOutputSchema(ctx context.Context, queries *retrievalsqlc.Queries, resourceID, versionID uuid.UUID) error {
	if err := queries.UpsertOutputSchema(ctx, retrievalsqlc.UpsertOutputSchemaParams{
		ResourceID: resourceID,
		VersionID:  versionID,
	}); err != nil {
		return fmt.Errorf("upsert output schema: %w", err)
	}
	return nil
}

func replaceOutputSchema(ctx context.Context, queries *retrievalsqlc.Queries, resourceID, versionID uuid.UUID, parameters []domain.OutputParameter) error {
	if err := ensureOutputSchema(ctx, queries, resourceID, versionID); err != nil {
		return err
	}
	if err := queries.DeleteOutputParameters(ctx, retrievalsqlc.DeleteOutputParametersParams{
		ResourceID: resourceID,
		VersionID:  versionID,
	}); err != nil {
		return fmt.Errorf("clear output parameters: %w", err)
	}
	for _, parameter := range parameters {
		if err := queries.CreateOutputParameter(ctx, retrievalsqlc.CreateOutputParameterParams{
			ResourceID: resourceID,
			VersionID:  versionID,
			Name:       parameter.Name,
			Kind:       parameter.Kind,
		}); err != nil {
			return fmt.Errorf("insert output parameter: %w", err)
		}
	}
	return nil
}

func upsertRetrievalParams(req domain.PreparedRetrieval, head domain.ResponseHead, payloadSize int64) (retrievalsqlc.UpsertRetrievalParams, error) {
	headersJSON, err := json.Marshal(redactHeaders(req.Headers))
	if err != nil {
		return retrievalsqlc.UpsertRetrievalParams{}, fmt.Errorf("marshal request headers: %w", err)
	}

	bodyHash := sha256.Sum256(req.Body)
	var requestBodyText *string
	if req.Body != nil {
		requestBodyText = new(string(req.Body))
	}
	return retrievalsqlc.UpsertRetrievalParams{
		ResourceID:        &req.ResourceID,
		VersionID:         &req.VersionID,
		Method:            req.Method,
		UrlTemplate:       req.URL,
		GeneratedUrl:      req.URL,
		PathParamsJson:    []byte(`{}`),
		QueryParamsJson:   []byte(`{}`),
		HeadersJson:       headersJSON,
		RequestBodyText:   requestBodyText,
		RequestBodyHash:   hex.EncodeToString(bodyHash[:]),
		RequestBodyIsJson: false,
		HttpStatus:        int32(head.StatusCode),
		ContentType:       head.ContentType,
		IsBinary:          head.IsBinary,
		PayloadHash:       "",
		PayloadSizeBytes:  payloadSize,
		UsedAuth:          hasAuthorizationHeader(req.Headers),
		FetchedAt:         time.Now().UTC(),
	}, nil
}

func toRetrievalRecordFromGet(retrieval retrievalsqlc.GetRetrievalByIDRow) domain.RetrievalRecord {
	return domain.RetrievalRecord{
		ID:           retrieval.ID,
		ResourceID:   uuidValue(retrieval.ResourceID),
		VersionID:    uuidValue(retrieval.VersionID),
		Method:       retrieval.Method,
		URLTemplate:  retrieval.UrlTemplate,
		GeneratedURL: retrieval.GeneratedUrl,
		HTTPStatus:   int(retrieval.HttpStatus),
		ContentType:  retrieval.ContentType,
		IsBinary:     retrieval.IsBinary,
		PayloadHash:  retrieval.PayloadHash,
		PayloadSize:  retrieval.PayloadSizeBytes,
		UsedAuth:     retrieval.UsedAuth,
		FetchedAt:    retrieval.FetchedAt,
		CreatedAt:    retrieval.CreatedAt,
	}
}

func toDomainOutputParameters(parameters []retrievalsqlc.GetOutputParametersRow) []domain.OutputParameter {
	output := make([]domain.OutputParameter, 0, len(parameters))
	for _, parameter := range parameters {
		output = append(output, domain.OutputParameter{Name: parameter.Name, Kind: parameter.Kind})
	}
	return output
}

func uuidValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func redactHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		switch strings.ToLower(key) {
		case "authorization", "cookie", "set-cookie":
			out[key] = "[redacted]"
		default:
			out[key] = value
		}
	}
	return out
}

func hasAuthorizationHeader(headers map[string]string) bool {
	for key := range headers {
		if strings.EqualFold(key, "Authorization") {
			return true
		}
	}
	return false
}
