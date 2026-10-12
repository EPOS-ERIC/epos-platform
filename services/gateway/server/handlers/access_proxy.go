package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

type accessRetriever interface {
	RetrieveStream(context.Context, *dataretrieverv1.RetrieveStreamRequest) (clients.RetrievalStream, error)
}

type accessProxy struct {
	retriever accessRetriever
	log       *slog.Logger
}

// AccessProxyMiddleware streams EXECUTE responses without a request deadline.
// DRY requests and invalid input fall through to the generated API handler.
func AccessProxyMiddleware(retriever accessRetriever, log *slog.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	proxy := &accessProxy{retriever: retriever, log: log}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/access" {
				next.ServeHTTP(w, r)
				return
			}
			contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || contentType != "application/json" {
				next.ServeHTTP(w, r)
				return
			}
			originalBody := r.Body
			defer originalBody.Close()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
					writeError(proxy.log, r.Context(), w, apperror.New(apperror.CodeRequestTooLarge, "Request body is too large."))
					return
				}
				writeError(proxy.log, r.Context(), w, apperror.Field("request", "The request is invalid.", err))
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			var peek struct {
				Mode json.RawMessage `json:"mode"`
			}
			if err := json.Unmarshal(body, &peek); err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if len(peek.Mode) > 0 {
				var mode gatewayapi.RetrieveMode
				if err := json.Unmarshal(peek.Mode, &mode); err != nil || mode != gatewayapi.RetrieveModeEXECUTE {
					next.ServeHTTP(w, r)
					return
				}
			}
			proxy.serveExecute(w, r, body)
		})
	}
}

func (p *accessProxy) serveExecute(w http.ResponseWriter, r *http.Request, body []byte) {
	var request gatewayapi.RetrieveResourceVersionRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&request); err != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
		writeError(p.log, r.Context(), w, apperror.Field("request", "The request is invalid.", err))
		return
	}
	if err := request.Validate(); err != nil {
		writeError(p.log, r.Context(), w, err)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(p.log, r.Context(), w, apperror.Field("request", "The request is invalid.", errors.New("request contains trailing JSON")))
		return
	}
	protoRequest, err := toProtoRetrieveResourceVersionRequest(&request)
	if err != nil {
		writeError(p.log, r.Context(), w, err)
		return
	}
	stream, err := p.retriever.RetrieveStream(r.Context(), protoRequest)
	if err != nil {
		p.writeStreamError(w, r, err)
		return
	}
	firstEvent, err := stream.Recv(r.Context())
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = apperror.Default(apperror.CodeUpstreamUnavailable)
		}
		p.writeStreamError(w, r, err)
		return
	}
	headers := firstEvent.GetHeaders()
	if headers == nil {
		p.writeStreamError(w, r, apperror.Default(apperror.CodeUpstreamUnavailable))
		return
	}
	if command := headers.GetCurlCommand(); command != "" {
		p.writeDryResponse(w, command)
		return
	}
	status := int(headers.GetUpstreamStatus())
	if status < 100 || status > 599 {
		p.writeStreamError(w, r, apperror.Default(apperror.CodeUpstreamUnavailable))
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
	if record := headers.GetRecord(); record != nil && record.GetId() > 0 {
		w.Header().Set("X-Retrieval-Id", strconv.FormatInt(record.GetId(), 10))
	}
	w.WriteHeader(status)
	flusher, _ := w.(http.Flusher)
	for {
		event, err := stream.Recv(r.Context())
		if err == io.EOF {
			return
		}
		if err != nil {
			if r.Context().Err() == nil {
				p.log.ErrorContext(r.Context(), "upstream response stream interrupted", "error", err)
			}
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

func (p *accessProxy) writeStreamError(w http.ResponseWriter, r *http.Request, err error) {
	if r.Context().Err() != nil {
		return
	}
	writeError(p.log, r.Context(), w, err)
}

func (p *accessProxy) writeDryResponse(w http.ResponseWriter, command string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"mode": "DRY", "curl_command": command}); err != nil {
		p.log.Error("encode dry access response", "error", err)
	}
}
