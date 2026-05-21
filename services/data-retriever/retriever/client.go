package retriever

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
)

type Result struct {
	StatusCode   int
	ContentType  string
	Payload      []byte
	PayloadHash  string
	IsBinary     bool
	PayloadSize  int
	ResponseDate time.Time
}

type Client struct {
	httpClient   *http.Client
	maxBodyBytes int64
}

func NewClient(timeout time.Duration, maxBodyBytes int64) *Client {
	return &Client{httpClient: &http.Client{Timeout: timeout}, maxBodyBytes: maxBodyBytes}
}

func (c *Client) Execute(ctx context.Context, req domain.ValidatedRequest) (Result, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.GeneratedURL, bytes.NewReader(req.BodyBytes))
	if err != nil {
		return Result{}, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", req.Body.ContentType)
	}
	if strings.TrimSpace(req.BearerToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.BearerToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	reader := io.LimitReader(resp.Body, c.maxBodyBytes+1)
	payload, err := io.ReadAll(reader)
	if err != nil {
		return Result{}, err
	}
	if int64(len(payload)) > c.maxBodyBytes {
		payload = payload[:c.maxBodyBytes]
	}

	hash := sha256.Sum256(payload)
	contentType := resp.Header.Get("Content-Type")

	return Result{
		StatusCode:   resp.StatusCode,
		ContentType:  contentType,
		Payload:      payload,
		PayloadHash:  hex.EncodeToString(hash[:]),
		IsBinary:     isBinaryPayload(contentType, payload),
		PayloadSize:  len(payload),
		ResponseDate: time.Now().UTC(),
	}, nil
}

func isBinaryPayload(contentType string, payload []byte) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(ct, "text/") || ct == "application/json" || ct == "application/xml" || strings.HasSuffix(ct, "+json") || strings.HasSuffix(ct, "+xml") {
		return false
	}
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/") || ct == "application/octet-stream" || ct == "application/pdf" {
		return true
	}

	return http.DetectContentType(payload) == "application/octet-stream"
}
