// Purpose: Verify a provider rejection changes only the output budget before bounded context recovery.
// Pattern: HTTP integration test for decision retries.
// Usage: go test ./internal/provider/httpclient -run TestRejectedOutput.
// Related: internal/provider/httpclient/request.go, internal/provider/transport/output_rejection.go.
package httpclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestRejectedOutput(t *testing.T) {
	for _, tc := range []struct {
		name          string
		manual, input int
		twice         bool
		wantCalls     int
		wantCapacity  bool
	}{
		{"lower bound", 0, 65537, false, 2, false},
		{"second rejection triggers compaction", 0, 65537, true, 2, true},
		{"input already fills window", 0, 262144, false, 1, true},
		{"explicit setting", 196608, 65537, false, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var first map[string]any
			requests, retries := 0, 0
			c := mockClient(t, connection.LiteLLM, func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				output := int(body["max_tokens"].(float64))
				delete(body, "max_tokens")
				requests++
				if requests == 1 {
					first = body
					if output != 196608 {
						t.Errorf("initial budget = %d", output)
					}
				} else if !reflect.DeepEqual(first, body) || output != 8192 {
					t.Error("recovery changed history or failed to reduce output")
				}
				if requests == 1 || tc.twice {
					input := tc.input
					if requests > 1 {
						input = 262145 - output
					}
					w.WriteHeader(400)
					fmt.Fprintf(w, `{"error":{"type":"BadRequestError","code":400,"message":"You requested %d output tokens and your prompt contains at least %d input tokens for a total of at least %d tokens."}}`, output, input, output+input)
					return
				}
				fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`)
			})
			err := c.Generate(t.Context(), modelapi.Request{ModelID: "alias", MaxOutput: tc.manual, Messages: []modelapi.Message{{Role: "user", Content: "inspect this page"}}}, func(event modelapi.Event) error {
				if event.Retry != nil {
					retries++
				}
				return nil
			})
			var capacity *modelapi.ContextLimitError
			if requests != tc.wantCalls || retries != tc.wantCalls-1 || errors.As(err, &capacity) != tc.wantCapacity || (err == nil) == tc.wantCapacity {
				t.Fatalf("requests=%d retries=%d error=%v", requests, retries, err)
			}
			if capacity != nil && capacity.Input != 0 {
				t.Fatal("lower bound was reported as exact", capacity.Input)
			}
		})
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
