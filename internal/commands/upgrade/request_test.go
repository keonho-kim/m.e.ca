// Purpose: Verify release redirects protect credentials and failed requests omit signed download URLs.
// Pattern: HTTPS transport contract tests.
// Usage: go test ./internal/commands/upgrade -run TestRelease.
// Related: internal/commands/upgrade/request.go, internal/commands/upgrade/fixtures_test.go.
package upgrade

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseRedirectDropsCredential(t *testing.T) {
	destination := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "" {
			t.Error("GitHub credential leaked to the asset host")
		}
		_, _ = writer.Write([]byte("asset"))
	}))
	defer destination.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, destination.URL+"/asset", http.StatusFound)
	}))
	defer origin.Close()
	client := &Client{http: origin.Client(), baseURL: origin.URL, token: "fixture-token"}
	client.http.CheckRedirect = releaseRedirect
	defer client.Close()
	if err := client.read(context.Background(), "/redirect", "application/octet-stream", func(reader io.Reader) error {
		_, err := io.Copy(io.Discard, reader)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseErrorsHideSignedURL(t *testing.T) {
	client := releaseFixture(t, nil, nil, func(writer http.ResponseWriter, request *http.Request) bool {
		http.Redirect(writer, request, "http://example.invalid/asset?signed-secret=private", http.StatusFound)
		return true
	})
	err := client.read(context.Background(), "/redirect", "application/octet-stream", func(io.Reader) error { return nil })
	if err == nil || strings.Contains(err.Error(), "signed-secret") || strings.Contains(err.Error(), "fixture-token") {
		t.Fatalf("release error is missing or exposes a secret: %v", err)
	}
}
