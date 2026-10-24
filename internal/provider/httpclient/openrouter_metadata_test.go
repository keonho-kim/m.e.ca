// Purpose: Verify OpenRouter model-specific reasoning limits and continuation isolation.
// Pattern: HTTP boundary contract tests.
// Usage: go test ./internal/provider/httpclient -run TestOpenRouter.
// Related: internal/provider/discovery/openrouter.go, internal/provider/httpclient/request.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestOpenRouterReasoningMetadata(t *testing.T) {
	for _, test := range []struct {
		name, reasoning string
		levels          int
		off             bool
		fail            bool
	}{
		{"mandatory", `{"supported_efforts":["high","low"],"mandatory":true}`, 2, false, false},
		{"gateway levels", `{"supported_efforts":null,"default_effort":"none"}`, 6, true, false},
		{"toggle only", `{}`, 0, true, false},
		{"invalid level", `{"supported_efforts":["high!?"]}`, 0, false, true},
		{"duplicate level", `{"supported_efforts":["high","high"]}`, 0, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"data":[{"id":"m","reasoning":%s}]}`, test.reasoning)
			})
			models, err := catalogFor(c).ListModels(context.Background())
			if test.fail {
				if err == nil {
					t.Fatal("invalid metadata accepted")
				}
				return
			}
			if err != nil || len(models) != 1 || models[0].Reasoning == nil {
				t.Fatalf("models=%+v err=%v", models, err)
			}
			r := models[0].Reasoning
			if len(r.Levels) != test.levels || r.Off != test.off || slices.Contains(r.Levels, "none") {
				t.Fatalf("options=%+v", r)
			}
		})
	}
}

func TestOpenRouterContinuationIsBoundToConnectionAndModel(t *testing.T) {
	for _, mode := range []string{"same", "other model", "other connection"} {
		t.Run(mode, func(t *testing.T) {
			c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Messages []struct {
						State json.RawMessage `json:"reasoning_details"`
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if len(body.Messages) != 1 || (len(body.Messages[0].State) > 0) != (mode == "same") {
					t.Errorf("continuation crossed model binding: %+v", body)
				}
				fmt.Fprint(w, `{"choices":[{"message":{"content":"done"},"finish_reason":"stop"}]}`)
			})
			m := modelapi.Message{Role: "assistant", Content: "earlier", ReasoningState: []byte(`[{"type":"reasoning.encrypted","data":"opaque"}]`), ProviderID: c.connection.ID, ModelID: "m"}
			if mode == "other model" {
				m.ModelID = "other"
			}
			if mode == "other connection" {
				m.ProviderID = "other"
			}
			if err := c.Generate(context.Background(), modelapi.Request{ModelID: "m", MaxOutput: 100, Messages: []modelapi.Message{m}}, func(modelapi.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if len(m.ReasoningState) == 0 {
				t.Fatal("request mutated archived continuation")
			}
		})
	}
}

func TestOpenRouterRetryDiscardsUnfinishedReasoningState(t *testing.T) {
	var attempts atomic.Int32
	c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		data := "stale"
		attempt := attempts.Add(1)
		if attempt > 1 {
			data = "fresh"
		}
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning_details\":[{\"type\":\"reasoning.encrypted\",\"data\":%q}]}}]}\n\n", data)
		if attempt > 1 {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		}
	})
	var continuation []byte
	err := c.Generate(context.Background(), modelapi.Request{ModelID: "m", MaxOutput: 100, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(event modelapi.Event) error {
		if event.Done {
			continuation = event.ReasoningState
		}
		return nil
	})
	if err != nil || attempts.Load() != 2 || !strings.Contains(string(continuation), "fresh") || strings.Contains(string(continuation), "stale") {
		t.Fatalf("attempts=%d state=%s err=%v", attempts.Load(), continuation, err)
	}
}
