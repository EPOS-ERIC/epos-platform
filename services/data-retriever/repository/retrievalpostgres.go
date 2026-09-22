package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	retrievalsqlc "github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/google/uuid"
)

type RetrievalRecord struct {
	ID               int64
	ResourceID       string
	VersionID        string
	Method           string
	URLTemplate      string
	GeneratedURL     string
	HTTPStatus       int
	ContentType      string
	IsBinary         bool
	PayloadHash      string
	PayloadSize      int
	UsedAuth         bool
	FetchedAt        time.Time
	CreatedAt        time.Time
	RequestBodyText  string
	OutputParameters []domain.OutputParameter
}

type RetrievalRepository interface {
	Create(ctx context.Context, req domain.ValidatedRequest, result retriever.Result) (RetrievalRecord, error)
	GetByID(ctx context.Context, id int64) (RetrievalRecord, error)
}

type RetrievalPostgresRepository struct {
	db      *sql.DB
	queries *retrievalsqlc.Queries
}

func NewRetrievalPostgresRepository(db *sql.DB) *RetrievalPostgresRepository {
	return &RetrievalPostgresRepository{db: db, queries: retrievalsqlc.New(db)}
}

func (r *RetrievalPostgresRepository) Create(ctx context.Context, req domain.ValidatedRequest, result retriever.Result) (record RetrievalRecord, err error) {
	resourceID, versionID, err := resourceVersionIDs(req)
	if err != nil {
		return RetrievalRecord{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RetrievalRecord{}, fmt.Errorf("begin retrieval transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	queries := r.queries.WithTx(tx)

	if err := replaceOutputSchema(ctx, queries, resourceID, versionID, req.OutputParameters); err != nil {
		return RetrievalRecord{}, err
	}

	retrieval, err := queries.UpsertRetrieval(ctx, upsertRetrievalParams(req, result, resourceID, versionID))
	if err != nil {
		return RetrievalRecord{}, fmt.Errorf("upsert retrieval: %w", err)
	}
	record = toRetrievalRecordFromUpsert(retrieval, req.OutputParameters)

	if err := tx.Commit(); err != nil {
		return RetrievalRecord{}, fmt.Errorf("commit retrieval transaction: %w", err)
	}
	return record, nil
}

func (r *RetrievalPostgresRepository) GetByID(ctx context.Context, id int64) (RetrievalRecord, error) {
	retrieval, err := r.queries.GetRetrievalByID(ctx, id)
	if err != nil {
		return RetrievalRecord{}, err
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
		return RetrievalRecord{}, fmt.Errorf("get output parameters: %w", err)
	}
	record.OutputParameters = toDomainOutputParameters(parameters)
	return record, nil
}

func replaceOutputSchema(ctx context.Context, queries *retrievalsqlc.Queries, resourceID, versionID uuid.UUID, parameters []domain.OutputParameter) error {
	if err := queries.UpsertOutputSchema(ctx, retrievalsqlc.UpsertOutputSchemaParams{ResourceID: resourceID, VersionID: versionID}); err != nil {
		return fmt.Errorf("upsert output schema: %w", err)
	}
	if err := queries.DeleteOutputParameters(ctx, retrievalsqlc.DeleteOutputParametersParams{ResourceID: resourceID, VersionID: versionID}); err != nil {
		return fmt.Errorf("clear output parameters: %w", err)
	}
	for _, parameter := range parameters {
		pattern := sql.NullString{}
		if parameter.Pattern != nil {
			pattern = sql.NullString{String: *parameter.Pattern, Valid: true}
		}
		if err := queries.CreateOutputParameter(ctx, retrievalsqlc.CreateOutputParameterParams{
			ResourceID: resourceID,
			VersionID:  versionID,
			Name:       parameter.Name,
			Kind:       parameter.Kind,
			Pattern:    pattern,
		}); err != nil {
			return fmt.Errorf("insert output parameter: %w", err)
		}
	}
	return nil
}

func resourceVersionIDs(req domain.ValidatedRequest) (uuid.UUID, uuid.UUID, error) {
	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("parse resource ID: %w", err)
	}
	versionID, err := uuid.Parse(req.VersionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("parse version ID: %w", err)
	}
	return resourceID, versionID, nil
}

func upsertRetrievalParams(req domain.ValidatedRequest, result retriever.Result, resourceID, versionID uuid.UUID) retrievalsqlc.UpsertRetrievalParams {
	pathParamsJSON, _ := json.Marshal(req.PathParams)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)
	headersJSON, _ := json.Marshal(redactHeaders(req.Headers))
	bodyHash := sha256.Sum256(req.BodyBytes)

	requestBodyText := sql.NullString{}
	if len(req.BodyBytes) > 0 {
		requestBodyText = sql.NullString{String: string(req.BodyBytes), Valid: true}
	}
	return retrievalsqlc.UpsertRetrievalParams{
		ResourceID:        &resourceID,
		VersionID:         &versionID,
		Method:            req.Method,
		UrlTemplate:       req.URLTemplate,
		GeneratedUrl:      req.GeneratedURL,
		PathParamsJson:    pathParamsJSON,
		QueryParamsJson:   queryParamsJSON,
		HeadersJson:       headersJSON,
		RequestBodyText:   requestBodyText,
		RequestBodyHash:   hex.EncodeToString(bodyHash[:]),
		RequestBodyIsJson: req.Body != nil && len(req.Body.JSON) > 0,
		HttpStatus:        int32(result.StatusCode),
		ContentType:       result.ContentType,
		IsBinary:          result.IsBinary,
		PayloadHash:       result.PayloadHash,
		PayloadSizeBytes:  int32(result.PayloadSize),
		UsedAuth:          req.BearerToken != "",
		FetchedAt:         result.ResponseDate,
	}
}

func toRetrievalRecordFromUpsert(retrieval retrievalsqlc.UpsertRetrievalRow, parameters []domain.OutputParameter) RetrievalRecord {
	return RetrievalRecord{
		ID:               retrieval.ID,
		ResourceID:       uuidString(retrieval.ResourceID),
		VersionID:        uuidString(retrieval.VersionID),
		Method:           retrieval.Method,
		URLTemplate:      retrieval.UrlTemplate,
		GeneratedURL:     retrieval.GeneratedUrl,
		HTTPStatus:       int(retrieval.HttpStatus),
		ContentType:      retrieval.ContentType,
		IsBinary:         retrieval.IsBinary,
		PayloadHash:      retrieval.PayloadHash,
		PayloadSize:      int(retrieval.PayloadSizeBytes),
		UsedAuth:         retrieval.UsedAuth,
		FetchedAt:        retrieval.FetchedAt,
		CreatedAt:        retrieval.CreatedAt,
		RequestBodyText:  nullString(retrieval.RequestBodyText),
		OutputParameters: parameters,
	}
}

func toRetrievalRecordFromGet(retrieval retrievalsqlc.GetRetrievalByIDRow) RetrievalRecord {
	return RetrievalRecord{
		ID:              retrieval.ID,
		ResourceID:      uuidString(retrieval.ResourceID),
		VersionID:       uuidString(retrieval.VersionID),
		Method:          retrieval.Method,
		URLTemplate:     retrieval.UrlTemplate,
		GeneratedURL:    retrieval.GeneratedUrl,
		HTTPStatus:      int(retrieval.HttpStatus),
		ContentType:     retrieval.ContentType,
		IsBinary:        retrieval.IsBinary,
		PayloadHash:     retrieval.PayloadHash,
		PayloadSize:     int(retrieval.PayloadSizeBytes),
		UsedAuth:        retrieval.UsedAuth,
		FetchedAt:       retrieval.FetchedAt,
		CreatedAt:       retrieval.CreatedAt,
		RequestBodyText: nullString(retrieval.RequestBodyText),
	}
}

func toDomainOutputParameters(parameters []retrievalsqlc.GetOutputParametersRow) []domain.OutputParameter {
	output := make([]domain.OutputParameter, 0, len(parameters))
	for _, parameter := range parameters {
		outputParameter := domain.OutputParameter{Name: parameter.Name, Kind: parameter.Kind}
		if parameter.Pattern.Valid {
			pattern := parameter.Pattern.String
			outputParameter.Pattern = &pattern
		}
		output = append(output, outputParameter)
	}
	return output
}

func uuidString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func redactHeaders(headers map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range headers {
		lower := strings.ToLower(key)
		if lower == "authorization" || lower == "cookie" || lower == "set-cookie" {
			out[key] = "[redacted]"
			continue
		}
		out[key] = value
	}
	return out
}
