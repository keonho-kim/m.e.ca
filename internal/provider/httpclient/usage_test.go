// Purpose: Verify provider usage requests and cumulative stream snapshots, including exact zero counts.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestTokenUsage.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/request.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"testing"
)

func TestTokenUsageProviderStreams(t *testing.T) {
	for _, provider := range connection.Providers() {
		t.Run(string(provider.Kind), func(t *testing.T) {
			client := mockClient(t, provider.Kind, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Options map[string]bool `json:"stream_options"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if provider.Kind == connection.Ollama {
					if body.Options != nil {
						t.Error("compatible options sent to native API")
					}
					w.Header().Set("Content-Type", "application/x-ndjson")
					fmt.Fprintln(w, `{"message":{"content":"hi"},"done":false}`)
					fmt.Fprintln(w, `{"message":{},"done":true,"prompt_eval_count":12,"eval_count":3}`)
				} else if provider.Kind == connection.LMStudio {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":12,\"output_tokens\":3}}}\n\n")
				} else {
					continuous := provider.Kind == connection.VLLM || provider.Kind == connection.SGLang
					if body.Options["continuous_usage_stats"] != continuous {
						t.Error("continuous usage option does not match provider")
					}
					if !body.Options["include_usage"] {
						t.Error("stream usage not requested")
					}
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}],\"usage\":null}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":3}}\n\ndata: [DONE]\n\n")
				}
			})
			var final modelapi.Event
			err := client.Generate(context.Background(), modelapi.Request{ModelID: "alias", MaxOutput: 32, Messages: []modelapi.Message{{Role: "user", Content: "hi"}}}, func(e modelapi.Event) error {
				if e.Done {
					final = e
				}
				return nil
			})
			if err != nil || final.Usage == nil || final.Usage.Input != 12 || final.Usage.Output != 3 {
				t.Fatalf("event=%+v err=%v", final, err)
			}
		})
	}
}
