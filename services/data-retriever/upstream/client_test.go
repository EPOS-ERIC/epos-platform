package upstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/google/uuid"
)

func preparedGetRequest(rawURL string, pathParams map[string]string) domain.PreparedRetrieval {
	prepared, err := domain.NewPreparedRetrieval(domain.RetrievalInput{
		ResourceID:  uuid.MustParse("2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192"),
		VersionID:   uuid.MustParse("67f6f345-5a27-467b-967b-0aa9e7f85bb1"),
		Method:      http.MethodGet,
		URLTemplate: rawURL,
		PathParams:  pathParams,
	})
	if err != nil {
		panic(err)
	}
	return prepared
}

func TestExecuteStreamDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/final")
		w.Header().Set("Content-Disposition", `attachment; filename="original.tif"`)
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte("redirect body"))
	}))
	defer server.Close()

	stream, err := NewClient(5*time.Second).ExecuteStream(t.Context(), preparedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	if stream.Head.StatusCode != http.StatusFound || stream.Head.Location != "/final" {
		t.Fatalf("response head = %+v", stream.Head)
	}
	if stream.Head.ContentDisposition != `attachment; filename="original.tif"` {
		t.Fatalf("content disposition = %q", stream.Head.ContentDisposition)
	}
}

func TestExecuteStreamReturnsOpenBody(t *testing.T) {
	payload := []byte(`{"ok":true}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	stream, err := NewClient(5*time.Second).ExecuteStream(t.Context(), preparedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	body, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != string(payload) {
		t.Fatalf("body = %q, want %q", body, payload)
	}
	if stream.Head.ContentLength != int64(len(payload)) || stream.Head.ContentDisposition != "" {
		t.Fatalf("response head = %+v", stream.Head)
	}
}

func TestExecuteStreamSynthesizesDispositionFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/GDM_DEM_radar_4rlks.tiff" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/tiff")
		_, _ = w.Write([]byte("TIFF"))
	}))
	defer server.Close()

	request := preparedGetRequest(server.URL+"/files/{name}", map[string]string{"name": "GDM_DEM_radar_4rlks.tiff"})
	stream, err := NewClient(5*time.Second).ExecuteStream(t.Context(), request)
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	if got, want := stream.Head.ContentDisposition, `inline; filename=GDM_DEM_radar_4rlks.tiff`; got != want {
		t.Fatalf("content disposition = %q, want %q", got, want)
	}
}

func TestStreamedHashMatchesPayload(t *testing.T) {
	payload := make([]byte, 3*64*1024+17)
	for i := range payload {
		payload[i] = byte(i)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	stream, err := NewClient(5*time.Second).ExecuteStream(context.Background(), preparedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()

	hash := sha256.New()
	var size int64
	buffer := make([]byte, 1024)
	for {
		count, readErr := stream.Body.Read(buffer)
		if count > 0 {
			hash.Write(buffer[:count])
			size += int64(count)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("read response body: %v", readErr)
		}
	}
	wantHash := sha256.Sum256(payload)
	if got, want := hex.EncodeToString(hash.Sum(nil)), hex.EncodeToString(wantHash[:]); got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
	if size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", size, len(payload))
	}
}

func TestURLFileNameDisposition(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.org/files/report%20finale.pdf", `inline; filename="report finale.pdf"`},
		{"https://example.org/search?format=json", ""},
		{"https://example.org/files/", ""},
		{"https://example.org/files", ""},
		{`https://example.org/files/we"ird.tif`, `inline; filename="we\"ird.tif"`},
		{"https://example.org/files/dati-ñ.tif", `inline; filename*=utf-8''dati-%C3%B1.tif`},
	}
	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			if got := syntheticDisposition(test.url); got != test.want {
				t.Fatalf("syntheticDisposition() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIsBinaryPayloadUsesDeclaredMediaType(t *testing.T) {
	for _, test := range []struct {
		contentType string
		want        bool
	}{
		{"application/json", false},
		{"application/vnd.example+json; charset=utf-8", false},
		{"text/csv", false},
		{"application/zip", true},
		{"application/octet-stream", true},
		{"", true},
	} {
		t.Run(test.contentType, func(t *testing.T) {
			if got := isBinaryPayload(test.contentType); got != test.want {
				t.Fatalf("isBinaryPayload(%q) = %t, want %t", test.contentType, got, test.want)
			}
		})
	}
}
