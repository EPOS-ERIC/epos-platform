package swagger

import (
	"io/fs"
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
			name:      "returns sub fs for embedded swagger ui",
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
