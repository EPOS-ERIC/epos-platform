package retriever

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
)

func validatedGetRequest(urlTemplate string, pathParams map[string]string) domain.ValidatedRequest {
	validated, err := domain.RetrieveRequest{
		Mode:        domain.RetrieveModeExecute,
		ResourceID:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionID:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
		Method:      http.MethodGet,
		URLTemplate: urlTemplate,
		PathParams:  pathParams,
	}.Validate()
	if err != nil {
		panic(err)
	}
	return validated
}

func TestExecuteStreamDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/final")
		w.Header().Set("Content-Disposition", `attachment; filename="original.tif"`)
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte("redirect body"))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	stream, err := client.ExecuteStream(context.Background(), validatedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	if stream.Head.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", stream.Head.StatusCode, http.StatusFound)
	}
	if stream.Head.Location != "/final" {
		t.Fatalf("location = %q, want %q", stream.Head.Location, "/final")
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

	client := NewClient(5 * time.Second)
	stream, err := client.ExecuteStream(context.Background(), validatedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	body, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != string(payload) {
		t.Fatalf("payload = %q", body)
	}
	if stream.Head.ContentLength != int64(len(payload)) {
		t.Fatalf("content length = %d, want %d", stream.Head.ContentLength, len(payload))
	}
	if stream.Head.ContentDisposition != "" {
		t.Fatalf("unexpected content disposition %q", stream.Head.ContentDisposition)
	}
}

func TestExecuteStreamSynthesizesDispositionFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/GDM_DEM_radar_4rlks.tiff" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/tiff")
		_, _ = w.Write([]byte("TIFF"))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	stream, err := client.ExecuteStream(context.Background(), validatedGetRequest(server.URL+"/files/{name}", map[string]string{"name": "GDM_DEM_radar_4rlks.tiff"}))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()
	if stream.Head.ContentDisposition != `inline; filename="GDM_DEM_radar_4rlks.tiff"` {
		t.Fatalf("content disposition = %q", stream.Head.ContentDisposition)
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

	client := NewClient(5 * time.Second)
	stream, err := client.ExecuteStream(context.Background(), validatedGetRequest(server.URL, nil))
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	defer func() { _ = stream.Body.Close() }()

	hash := sha256.New()
	var size int64
	buffer := make([]byte, 1024)
	for {
		n, readErr := stream.Body.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
			size += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("read body: %v", readErr)
		}
	}
	want := sha256.Sum256(payload)
	if hex.EncodeToString(hash.Sum(nil)) != hex.EncodeToString(want[:]) {
		t.Fatal("streamed hash does not match payload hash")
	}
	if size != int64(len(payload)) {
		t.Fatalf("streamed size = %d, want %d", size, len(payload))
	}
}

func TestContentDispositionFromURLFileName(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://formater.osug.fr/permanent-products-repository/public/fe0bfdd7-32a3-490b-b958-a57ca87ffbe0/iw1_iw2_iw3/GDM_DEM_radar_4rlks.tiff",
			`inline; filename="GDM_DEM_radar_4rlks.tiff"`,
		},
		{"https://example.org/files/report%20finale.pdf", `inline; filename="report finale.pdf"`},
		{"https://example.org/search?format=json", ""},
		{"https://example.org/files/", ""},
		{"https://example.org/files", ""},
		{"https://example.org/", ""},
		{":://not a url", ""},
		{`https://example.org/files/we"ird.tif`, `inline; filename="download"; filename*=UTF-8''we%22ird.tif`},
		{"https://example.org/files/dati-ñ.tif", `inline; filename="download"; filename*=UTF-8''dati-%C3%B1.tif`},
	}
	for _, tc := range cases {
		if got := syntheticDisposition(tc.url); got != tc.want {
			t.Errorf("syntheticDisposition(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
