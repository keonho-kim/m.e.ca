// Purpose: Preserve authenticated, replayable HTTP requests when cached history is streamed in segments.
// Pattern: Local HTTP integration tests with redirects and bounded retry recovery.
// Usage: Run TestRunGeneratorHTTPReplay without model servers or credentials.
// Related: internal/provider/httpclient/run_generator.go, internal/provider/httpclient/request.go.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestRunGeneratorHTTPReplay(t *testing.T) {
	var mu sync.Mutex
	var original []byte
	redirects := 0
	client := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, err := io.ReadAll(r.Body)
		if err != nil || int64(len(body)) != r.ContentLength || !json.Valid(body) || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("invalid body, length or authentication: length=%d declared=%d err=%v", len(body), r.ContentLength, err)
		}
		if original == nil {
			original = bytes.Clone(body)
		} else if !bytes.Equal(original, body) {
			t.Error("retry or redirect changed the request body")
		}
		if !strings.HasSuffix(r.URL.Path, "/redirect") {
			w.Header().Set("Location", r.URL.Path+"/redirect")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		redirects++
		if redirects == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"answer\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	})
	generator, release := client.NewRunGenerator()
	defer release()
	request := modelapi.Request{ModelID: "model", MaxOutput: 100, Messages: []modelapi.Message{{Role: "system", Content: strings.Repeat("x", 70*1024)}, {Role: "user", Content: "hello"}}}
	answer, retries := "", 0
	err := generator.Generate(context.Background(), request, func(event modelapi.Event) error {
		answer += event.Delta
		if event.Retry != nil {
			retries++
		}
		return nil
	})
	mu.Lock()
	defer mu.Unlock()
	if err != nil || redirects != 2 || retries != 1 || answer != "answer" {
		t.Fatalf("redirects=%d retries=%d answer=%q err=%v", redirects, retries, answer, err)
	}
}
