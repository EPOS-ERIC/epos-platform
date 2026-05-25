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
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
)

type RetrievalRecord struct {
	ID              int64     `json:"id"`
	SourceID        string    `json:"source_id"`
	Method          string    `json:"method"`
	URLTemplate     string    `json:"url_template"`
	GeneratedURL    string    `json:"generated_url"`
	HTTPStatus      int       `json:"http_status"`
	ContentType     string    `json:"content_type"`
	IsBinary        bool      `json:"is_binary"`
	PayloadText     string    `json:"payload_text,omitempty"`
	PayloadHash     string    `json:"payload_hash"`
	PayloadSize     int       `json:"payload_size_bytes"`
	UsedAuth        bool      `json:"used_auth"`
	FetchedAt       time.Time `json:"fetched_at"`
	CreatedAt       time.Time `json:"created_at"`
	RequestBodyText string    `json:"request_body_text,omitempty"`
}

type SearchResult struct {
	RetrievalRecord
	Score float64 `json:"score"`
}

type RetrievalRepository interface {
	Create(ctx context.Context, req domain.ValidatedRequest, result retriever.Result) (RetrievalRecord, error)
	GetByID(ctx context.Context, id int64) (RetrievalRecord, error)
	Search(ctx context.Context, q string, sourceID string, method string, page int, pageSize int) ([]SearchResult, error)
}

type RetrievalPostgresRepository struct {
	db *sql.DB
}

func NewRetrievalPostgresRepository(db *sql.DB) *RetrievalPostgresRepository {
	return &RetrievalPostgresRepository{db: db}
}

func (r *RetrievalPostgresRepository) Create(ctx context.Context, req domain.ValidatedRequest, result retriever.Result) (RetrievalRecord, error) {
	pathParamsJSON, _ := json.Marshal(req.PathParams)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)
	headersJSON, _ := json.Marshal(redactHeaders(req.Headers))

	bodyText := ""
	if len(req.BodyBytes) > 0 {
		bodyText = string(req.BodyBytes)
	}
	bodyHash := sha256.Sum256(req.BodyBytes)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	payloadText := ""
	if !result.IsBinary {
		payloadText = string(result.Payload)
	}

	usedAuth := req.BearerToken != ""

	row := r.db.QueryRowContext(ctx, `
		WITH upserted AS (
			INSERT INTO retrievals (
				source_id, method, url_template, generated_url,
				path_params_json, query_params_json, headers_json,
				request_body_text, request_body_hash, request_body_is_json,
				http_status, content_type, is_binary, payload_text, payload_hash, payload_size_bytes,
				used_auth, fetched_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
			ON CONFLICT (source_id, method, generated_url, request_body_hash)
			DO UPDATE SET
				url_template = EXCLUDED.url_template,
				path_params_json = EXCLUDED.path_params_json,
				query_params_json = EXCLUDED.query_params_json,
				headers_json = EXCLUDED.headers_json,
				request_body_text = EXCLUDED.request_body_text,
				request_body_is_json = EXCLUDED.request_body_is_json,
				http_status = EXCLUDED.http_status,
				content_type = EXCLUDED.content_type,
				is_binary = EXCLUDED.is_binary,
				payload_text = EXCLUDED.payload_text,
				payload_hash = EXCLUDED.payload_hash,
				payload_size_bytes = EXCLUDED.payload_size_bytes,
				used_auth = EXCLUDED.used_auth,
				fetched_at = EXCLUDED.fetched_at,
				updated_at = NOW()
			WHERE retrievals.payload_hash IS DISTINCT FROM EXCLUDED.payload_hash
			RETURNING id, source_id, method, url_template, generated_url, http_status, content_type, is_binary, COALESCE(payload_text,''), payload_hash, payload_size_bytes, used_auth, fetched_at, created_at, COALESCE(request_body_text,'')
		)
		SELECT * FROM upserted
		UNION ALL
		SELECT id, source_id, method, url_template, generated_url, http_status, content_type, is_binary, COALESCE(payload_text,''), payload_hash, payload_size_bytes, used_auth, fetched_at, created_at, COALESCE(request_body_text,'')
		FROM retrievals
		WHERE source_id = $1 AND method = $2 AND generated_url = $4 AND request_body_hash = $9
		LIMIT 1`,
		req.SourceID,
		req.Method,
		req.URLTemplate,
		req.GeneratedURL,
		string(pathParamsJSON),
		string(queryParamsJSON),
		string(headersJSON),
		nullIfEmpty(bodyText),
		bodyHashHex,
		req.Body != nil && len(req.Body.JSON) > 0,
		result.StatusCode,
		result.ContentType,
		result.IsBinary,
		nullIfEmpty(payloadText),
		result.PayloadHash,
		result.PayloadSize,
		usedAuth,
		result.ResponseDate,
	)

	var record RetrievalRecord
	err := row.Scan(
		&record.ID,
		&record.SourceID,
		&record.Method,
		&record.URLTemplate,
		&record.GeneratedURL,
		&record.HTTPStatus,
		&record.ContentType,
		&record.IsBinary,
		&record.PayloadText,
		&record.PayloadHash,
		&record.PayloadSize,
		&record.UsedAuth,
		&record.FetchedAt,
		&record.CreatedAt,
		&record.RequestBodyText,
	)
	if err != nil {
		return RetrievalRecord{}, fmt.Errorf("insert retrieval: %w", err)
	}

	return record, nil
}

func (r *RetrievalPostgresRepository) GetByID(ctx context.Context, id int64) (RetrievalRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source_id, method, url_template, generated_url, http_status, content_type, is_binary, COALESCE(payload_text,''), payload_hash, payload_size_bytes, used_auth, fetched_at, created_at, COALESCE(request_body_text,'')
		FROM retrievals
		WHERE id = $1`, id)

	var record RetrievalRecord
	err := row.Scan(
		&record.ID,
		&record.SourceID,
		&record.Method,
		&record.URLTemplate,
		&record.GeneratedURL,
		&record.HTTPStatus,
		&record.ContentType,
		&record.IsBinary,
		&record.PayloadText,
		&record.PayloadHash,
		&record.PayloadSize,
		&record.UsedAuth,
		&record.FetchedAt,
		&record.CreatedAt,
		&record.RequestBodyText,
	)
	if err != nil {
		return RetrievalRecord{}, err
	}
	return record, nil
}

func (r *RetrievalPostgresRepository) Search(ctx context.Context, q string, sourceID string, method string, page int, pageSize int) ([]SearchResult, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, source_id, method, url_template, generated_url, http_status, content_type, is_binary,
		COALESCE(payload_text,''), payload_hash, payload_size_bytes, used_auth, fetched_at, created_at,
		COALESCE(request_body_text,''), ts_rank(search_vector, plainto_tsquery('simple', $1)) AS score
		FROM retrievals
		WHERE ($2 = '' OR source_id = $2)
		AND ($3 = '' OR method = $3)
		AND search_vector @@ plainto_tsquery('simple', $1)
		ORDER BY score DESC, fetched_at DESC
		LIMIT $4 OFFSET $5`, q, sourceID, method, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("search retrievals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	results := []SearchResult{}
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(
			&item.ID,
			&item.SourceID,
			&item.Method,
			&item.URLTemplate,
			&item.GeneratedURL,
			&item.HTTPStatus,
			&item.ContentType,
			&item.IsBinary,
			&item.PayloadText,
			&item.PayloadHash,
			&item.PayloadSize,
			&item.UsedAuth,
			&item.FetchedAt,
			&item.CreatedAt,
			&item.RequestBodyText,
			&item.Score,
		); err != nil {
			return nil, fmt.Errorf("scan retrieval search row: %w", err)
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate retrieval search rows: %w", err)
	}

	return results, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func redactHeaders(headers map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range headers {
		lower := strings.ToLower(k)
		if lower == "authorization" || lower == "cookie" || lower == "set-cookie" {
			out[k] = "[redacted]"
			continue
		}
		out[k] = v
	}
	return out
}
