// Purpose: Preserve structured provider errors beyond the former four-kilobyte response cutoff.
// Pattern: HTTP boundary regression test.
// Usage: go test ./internal/provider/transport -run TestLargeError.
// Related: internal/provider/transport/transport.go, internal/provider/transport/errors.go.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLargeErrorPreservesCapacityClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{"padding": strings.Repeat("x", 70*1024), "error": map[string]string{"code": "context_length_exceeded", "message": "too long"}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := New("")
	defer client.Close()
	request, err := client.NewRequest(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendOnce(request, time.Second)
	var capacity interface{ ContextLimit() bool }
	if !errors.As(err, &capacity) || !capacity.ContextLimit() {
		t.Fatalf("large error lost its type: %v", err)
	}
}
