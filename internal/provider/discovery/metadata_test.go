// Purpose: Verify preparation snapshots preserve retry, decoding, cancellation and ownership contracts.
// Pattern: HTTP adapter contract tests.
// Usage: go test ./internal/provider/discovery -run TestMetadata.
// Related: internal/provider/discovery/metadata.go, internal/provider/transport/metadata.go.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/transport"
)

func TestMetadataSnapshotRetriesAndCopies(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			fmt.Fprint(w, `{"values":"bad shape"}`)
			return
		}
		fmt.Fprint(w, `{"values":["original"]}`)
	}))
	t.Cleanup(server.Close)
	httpClient := transport.New("")
	t.Cleanup(httpClient.Close)
	client := Client{HTTP: httpClient}
	var first, second struct{ Values []string }
	if err := readMetadata(context.Background(), &client, server.URL, &first); err != nil {
		t.Fatal(err)
	}
	first.Values[0] = "changed"
	if err := readMetadata(context.Background(), &client, server.URL, &second); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || second.Values[0] != "original" {
		t.Fatalf("retry or ownership lost: %d %+v", calls.Load(), second)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := readMetadata(ctx, &client, server.URL, &second); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := readMetadata(context.Background(), &client, server.URL+"/fresh", &second); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatal("different endpoints shared a response")
	}
}
