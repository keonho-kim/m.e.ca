// Purpose: Verify one connection preparation reuses a models response and refreshes on the next preparation.
// Pattern: HTTP integration test.
// Usage: go test ./internal/provider/setup -run TestPreparationMetadata.
// Related: internal/provider/setup/setup.go, internal/provider/discovery/client.go.
package setup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestPreparationMetadataReadsCatalogOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		calls.Add(1)
		fmt.Fprint(w, `{"data":[{"id":"sample","canonical_slug":"publisher/original","context_length":32768}]}`)
	}))
	t.Cleanup(server.Close)
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	for want := int32(1); want <= 2; want++ {
		prepared, err := Prepare(context.Background(), conn, "", "sample", Defaults{}, connection.Selection{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prepared.Client.Close()
		if calls.Load() != want {
			t.Fatalf("requests=%d, want %d; catalog was repeated or stale across preparations", calls.Load(), want)
		}
		if prepared.Selection.Details.ContextWindow != 32768 || prepared.Selection.Identity.Unavailable {
			t.Fatal("metadata projection lost")
		}
	}
}
