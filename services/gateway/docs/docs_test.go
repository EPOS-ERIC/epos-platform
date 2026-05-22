package docs

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSubFSOrPanic(t *testing.T) {
	tests := []struct {
		name      string
		sourceFS  fs.FS
		dir       string
		wantPanic bool
		checkFile string
	}{
		{
			name:      "returns sub fs for embedded docs ui",
			sourceFS:  swaggerUI,
			dir:       "swagger-ui",
			wantPanic: false,
			checkFile: "index.html",
		},
		{
			name:      "panics when sub directory missing",
			sourceFS:  fstest.MapFS{},
			dir:       "../swagger-ui",
			wantPanic: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			didPanic := false
			var uiFS fs.FS

			func() {
				defer func() {
					if recover() != nil {
						didPanic = true
					}
				}()

				uiFS = subFSOrPanic(tc.sourceFS, tc.dir)
			}()

			if tc.wantPanic {
				if !didPanic {
					t.Fatal("expected panic")
				}
				return
			}

			if didPanic {
				t.Fatal("unexpected panic")
			}

			file, err := uiFS.Open(tc.checkFile)
			if err != nil {
				t.Fatalf("open %s: %v", tc.checkFile, err)
			}

			if err := file.Close(); err != nil {
				t.Fatalf("close %s: %v", tc.checkFile, err)
			}
		})
	}
}

func TestGetScalarUI(t *testing.T) {
	t.Parallel()

	uiFS := GetScalarUI()

	indexFile, err := uiFS.Open("index.html")
	if err != nil {
		t.Fatalf("open scalar index: %v", err)
	}
	if err := indexFile.Close(); err != nil {
		t.Fatalf("close scalar index: %v", err)
	}

	indexContent, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		t.Fatalf("read scalar index: %v", err)
	}

	apiReference, err := fs.ReadFile(uiFS, "api-reference.js")
	if err != nil {
		t.Fatalf("read scalar api reference script: %v", err)
	}

	if !strings.Contains(string(indexContent), "Scalar API Reference") {
		t.Fatal("expected scalar ui title")
	}

	if !strings.Contains(string(indexContent), "src=\"./api-reference.js\"") {
		t.Fatal("expected scalar ui to load local api reference script")
	}

	if !strings.Contains(string(indexContent), "url: '/openapi.yaml'") {
		t.Fatal("expected scalar ui to use embedded openapi spec route")
	}

	if !strings.Contains(string(apiReference), "@scalar/api-reference@") {
		t.Fatal("expected vendored scalar api reference script")
	}
}
