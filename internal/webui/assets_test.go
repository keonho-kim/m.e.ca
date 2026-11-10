// Purpose: Verify embedded asset negotiation, caching and static routing without a browser build.
// Pattern: HTTP adapter contract tests over a deterministic in-memory bundle.
// Usage: go test ./internal/webui.
// Related: internal/webui/assets.go, internal/webui/encoding.go.
package webui

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEmbeddedAssetDelivery(t *testing.T) {
	var compressed bytes.Buffer
	zip := gzip.NewWriter(&compressed)
	_, _ = io.WriteString(zip, "<html>MECA</html>")
	if err := zip.Close(); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(map[string]Asset{"index.html": {File: "index.gz", Type: "text/html", ETag: `W/"test"`, Bytes: 17}})
	handler, err := newHandler(fstest.MapFS{"manifest.json": {Data: manifest}, "index.gz": {Data: compressed.Bytes()}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, encoding, etag string
		status               int
		gzip                 bool
	}{
		{name: "identity", status: 200},
		{name: "gzip", encoding: "gzip", status: 200, gzip: true},
		{name: "reject", encoding: "*;q=0", status: 406},
		{name: "explicit identity", encoding: "*;q=0, identity;q=1", status: 200},
		{name: "weak comparison", etag: `"other", "test"`, status: 304},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ui/w/project/c/conversation", nil)
			req.Header.Set("Accept", "text/html")
			req.Header.Set("Accept-Encoding", tc.encoding)
			req.Header.Set("If-None-Match", tc.etag)
			response := httptest.NewRecorder()
			response.Header().Add("Vary", "Origin")
			handler.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status %d", response.Code)
			}
			if tc.gzip && response.Header().Get("Content-Encoding") != "gzip" {
				t.Fatal("missing gzip")
			}
			if tc.status == 200 && !tc.gzip && response.Body.String() != "<html>MECA</html>" {
				t.Fatal(response.Body.String())
			}
			if !strings.Contains(strings.Join(response.Header().Values("Vary"), ","), "Origin") {
				t.Fatal("lost origin variance")
			}
		})
	}
	for _, path := range []string{"/ui/missing.js", "/ui/../assets/private"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 404 {
			t.Fatalf("%s: %d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("HEAD", "/ui/", nil))
	if response.Code != 200 || response.Body.Len() != 0 {
		t.Fatal("HEAD body")
	}
}
