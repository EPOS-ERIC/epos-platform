package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

// accessRetriever is the subset of the data-retriever adapter used by the
// access proxy. It is satisfied by *clients.DataRetriever.
type accessRetriever interface {
	RetrieveStream(ctx context.Context, request *dataretrieverv1.RetrieveStreamRequest) (dataretrieverv1.DataRetrieverService_RetrieveStreamClient, error)
}

type accessProxy struct {
	retriever accessRetriever
	log       *slog.Logger
}

// AccessProxyMiddleware relays POST /v1/access calls in EXECUTE mode to the
// upstream resource verbatim: the HTTP status, Content-Type, body and, for
// 3xx responses, the Location header are streamed to the caller with bounded
// memory. The persisted retrieval identifier is exposed via the X-Retrieval-Id
// header. DRY requests and anything else fall through to the generated ogen
// server.
//
// The proxy runs outside the request timeout handler: downloads stream until
// the caller cancels the context.
func AccessProxyMiddleware(retriever accessRetriever, log *slog.Logger) func(http.Handler) http.Handler {
	proxy := &accessProxy{retriever: retriever, log: log}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/access" {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeAccessProblem(w, http.StatusBadRequest, "invalid_request", "invalid request")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			var peek struct {
				Mode *gatewayapi.RetrieveMode `json:"mode"`
			}
			if err := json.Unmarshal(body, &peek); err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if peek.Mode != nil && *peek.Mode != gatewayapi.RetrieveModeEXECUTE {
				next.ServeHTTP(w, r)
				return
			}
			proxy.serveExecute(w, r, body)
		})
	}
}

func (p *accessProxy) serveExecute(w http.ResponseWriter, r *http.Request, body []byte) {
	var request gatewayapi.RetrieveResourceVersionRequest
	if err := json.Unmarshal(body, &request); err != nil {
		writeAccessProblem(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	protoRequest, err := toProtoRetrieveResourceVersionRequest(&request)
	if err != nil {
		writeAccessProblem(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	stream, err := p.retriever.RetrieveStream(r.Context(), protoRequest)
	if err != nil {
		writeAccessProblem(w, http.StatusBadGateway, "upstream_unavailable", "upstream service is unavailable")
		return
	}
	headersEvent, err := stream.Recv()
	if err != nil {
		writeAccessProblem(w, http.StatusBadGateway, "upstream_unavailable", "upstream service is unavailable")
		return
	}
	headers := headersEvent.GetHeaders()
	if headers == nil {
		writeAccessProblem(w, http.StatusBadGateway, "upstream_unavailable", "upstream service is unavailable")
		return
	}
	if command := headers.GetCurlCommand(); command != "" {
		writeAccessJSON(w, map[string]any{"mode": "DRY", "curl_command": command})
		return
	}
	status := int(headers.GetUpstreamStatus())
	if status < 100 || status > 599 {
		writeAccessProblem(w, http.StatusBadGateway, "upstream_unavailable", "upstream service is unavailable")
		return
	}
	contentType := headers.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	if location := headers.GetRedirectUrl(); location != "" {
		w.Header().Set("Location", location)
	}
	if disposition := headers.GetContentDisposition(); disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	if record := headers.GetRecord(); record != nil {
		w.Header().Set("X-Retrieval-Id", strconv.FormatInt(record.GetId(), 10))
	}
	w.WriteHeader(status)
	flusher, _ := w.(http.Flusher)
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			p.log.ErrorContext(r.Context(), "upstream response stream interrupted", "error", err)
			return
		}
		if chunk := event.GetChunk(); len(chunk) > 0 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func writeAccessJSON(w http.ResponseWriter, value map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAccessProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	problem := map[string]any{"title": title, "status": status}
	if detail != "" {
		problem["detail"] = detail
	}
	_ = json.NewEncoder(w).Encode(problem)
}
