// Purpose: Check streamed metadata limits, retry isolation and JSON compatibility.
// Pattern: Loopback HTTP contract tests.
// Usage: go test ./internal/provider/transport -run TestStreamingMetadata.
// Related: internal/provider/transport/metadata.go, internal/provider/transport/retry.go.
package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type tokenizerFixture struct {
	Count *int `json:"count"`
	Limit int  `json:"max_model_len"`
}

func TestStreamingMetadataRejectsOversizedValidAndMalformedBodies(t *testing.T) {
	const limit = 16 * 1024
	for _, prefix := range []string{`{"count":12,"max_model_len":100}`, "!"} {
		t.Run(prefix, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				_, _ = io.WriteString(w, prefix)
				_ = http.NewResponseController(w).Flush()
				_, _ = io.WriteString(w, strings.Repeat(" ", limit))
			}))
			t.Cleanup(server.Close)
			client := New("", "")
			t.Cleanup(client.Close)
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			target := tokenizerFixture{Limit: 77}
			err := ReadStreamingMetadata(ctx, client, http.MethodGet, server.URL, nil, limit, &target)
			if err == nil || !strings.Contains(err.Error(), "exceeds") || attempts.Load() != 1 || target != (tokenizerFixture{Limit: 77}) {
				t.Fatalf("bound attempts=%d result=%+v error=%v", attempts.Load(), target, err)
			}
		})
	}
}

func TestStreamingMetadataRetryDecodesFreshValues(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"count":19,"max_model_len":99,"tokens":[1,2]} []`)
			return
		}
		_, _ = io.WriteString(w, `{"max_model_len":64,"MAX_MODEL_LEN":128,"tokens":[3,4]}`)
	}))
	t.Cleanup(server.Close)
	client := New("", "")
	t.Cleanup(client.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	target := tokenizerFixture{Count: new(77)}
	err := ReadStreamingMetadata(ctx, client, http.MethodGet, server.URL, nil, 1024, &target)
	if err != nil || attempts.Load() != 2 || target != (tokenizerFixture{Limit: 128}) {
		t.Fatalf("retry attempts=%d result=%+v error=%v", attempts.Load(), target, err)
	}
}
