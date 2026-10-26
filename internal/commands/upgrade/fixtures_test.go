// Purpose: Supply isolated GitHub release endpoints and small archives for updater boundary tests.
// Pattern: Local HTTPS fixtures.
// Usage: Upgrade tests own and close each fixture through testing cleanup.
// Related: internal/commands/upgrade/release_test.go, internal/commands/upgrade/archive_test.go.
package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

type archiveEntry struct {
	name, content string
	kind          byte
}

func tarball(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var body bytes.Buffer
	compressed := gzip.NewWriter(&body)
	writer := tar.NewWriter(compressed)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Typeflag: kind, Mode: 0755, Size: int64(len(entry.content))}
		if kind != tar.TypeReg {
			header.Size, header.Linkname = 0, "../outside"
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := writer.Write([]byte(entry.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func releaseFixture(t *testing.T, archive []byte, alter func(*release, *string), intercept func(http.ResponseWriter, *http.Request) bool) *Client {
	t.Helper()
	name, err := archiveName("v0.0.20261026", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name)
	latest := release{Version: "v0.0.20261026", Assets: []asset{{ID: 1, Name: "checksums.txt", Size: int64(len(manifest))}, {ID: 2, Name: name, Size: int64(len(archive))}}}
	if alter != nil {
		alter(&latest, &manifest)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("release request lost its authentication")
		}
		if intercept != nil && intercept(writer, request) {
			return
		}
		switch request.URL.Path {
		case "/releases/latest":
			if request.Header.Get("Accept") != "application/vnd.github+json" {
				t.Error("invalid release metadata accept header")
			}
			_ = json.NewEncoder(writer).Encode(latest)
		case "/releases/assets/1":
			fmt.Fprint(writer, manifest)
		case "/releases/assets/2":
			if request.Header.Get("Accept") != "application/octet-stream" {
				t.Error("asset was not requested as binary data")
			}
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	client := &Client{http: server.Client(), baseURL: server.URL, token: "fixture-token"}
	client.http.CheckRedirect = releaseRedirect
	t.Cleanup(client.Close)
	return client
}

func temporaryRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}
