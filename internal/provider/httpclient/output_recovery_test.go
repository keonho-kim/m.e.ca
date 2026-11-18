// Purpose: Verify a provider budget rejection reaches compaction without resubmitting the generation.
// Pattern: HTTP integration test for context-capacity signalling.
// Usage: go test ./internal/provider/httpclient -run TestRejectedOutput.
// Related: internal/provider/httpclient/request.go, internal/provider/transport/output_rejection.go.
package httpclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

// A numeric budget refusal must reach the compaction owner on the first response.
func TestRejectedOutputTriggersCompaction(t *testing.T) {
	for _, setting := range []int{0, 196608} {
		requests, retries := 0, 0
		c := mockClient(t, connection.LiteLLM, func(w http.ResponseWriter, r *http.Request) {
			requests++
			var body struct {
				Output int `json:"max_tokens"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if body.Output != 196608 {
				t.Errorf("unexpected output request: %d", body.Output)
			}
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"type":"BadRequestError","code":400,"message":"You requested 196608 output tokens and your prompt contains at least 65537 input tokens for a total of at least 262145 tokens."}}`)
		})
		err := c.Generate(t.Context(), modelapi.Request{ModelID: "alias", MaxOutput: setting, Messages: []modelapi.Message{{Role: "user", Content: "inspect this page"}}}, func(event modelapi.Event) error {
			if event.Retry != nil {
				retries++
			}
			return nil
		})
		var capacity *modelapi.ContextLimitError
		if !errors.As(err, &capacity) || capacity.Input != 0 || requests != 1 || retries != 0 {
			t.Fatalf("requests=%d retries=%d err=%v", requests, retries, err)
		}
	}
}

func TestContextFailureAfterStreamDoesNotRestart(t *testing.T) {
	requests, retries := 0, 0
	c := mockClient(t, connection.LiteLLM, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: "+`{"choices":[{"delta":{"content":"partial"}}]}`+"\n\n")
		fmt.Fprint(w, "data: "+`{"error":{"code":"context_length_exceeded","message":"generation failed"}}`+"\n\n")
	})
	err := c.Generate(t.Context(), modelapi.Request{ModelID: "alias", Messages: []modelapi.Message{{Role: "user", Content: "test"}}}, func(event modelapi.Event) error {
		if event.Retry != nil {
			retries++
		}
		return nil
	})
	var capacity *modelapi.ContextLimitError
	if !errors.As(err, &capacity) || !capacity.Partial || requests != 1 || retries != 0 {
		t.Fatalf("requests=%d retries=%d error=%v", requests, retries, err)
	}
}
