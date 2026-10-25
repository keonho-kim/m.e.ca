// Purpose: Verify actual OpenRouter charges survive streaming and complete HTTP responses.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestReportedCost.
// Related: internal/provider/protocol/chunk.go, internal/agent/model/usage.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestReportedCostProviderResponses(t *testing.T) {
	for _, tc := range []struct {
		name, cost string
		provider   connection.ProviderKind
		stream     bool
		want       *float64
	}{
		{"stream", "0.012345", connection.OpenRouter, true, new(0.012345)},
		{"complete", "0.012345", connection.OpenRouter, false, new(0.012345)},
		{"free", "0", connection.OpenRouter, true, new(0.0)},
		{"missing", "null", connection.OpenRouter, true, nil},
		{"other provider", "0.012345", connection.VLLM, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := mockClient(t, tc.provider, func(w http.ResponseWriter, _ *http.Request) {
				usage := fmt.Sprintf(`"usage":{"prompt_tokens":12,"completion_tokens":3,"cost":%s}`, tc.cost)
				if tc.stream {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"choices\":[],%s}\n\ndata: [DONE]\n\n", usage)
				} else {
					fmt.Fprintf(w, `{"choices":[{"message":{"content":"hi"},"finish_reason":"stop"}],%s}`, usage)
				}
			})
			var final modelapi.Event
			err := client.Generate(context.Background(), modelapi.Request{ModelID: "alias", MaxOutput: 32, Messages: []modelapi.Message{{Role: "user", Content: "hi"}}}, func(e modelapi.Event) error {
				if e.Done {
					final = e
				}
				return nil
			})
			if err != nil || final.Usage == nil {
				t.Fatalf("usage=%+v err=%v", final.Usage, err)
			}
			body, err := json.Marshal(final.Usage)
			var got struct {
				Cost *float64 `json:"cost"`
			}
			if err != nil || json.Unmarshal(body, &got) != nil {
				t.Fatal("invalid usage serialization")
			}
			if (got.Cost == nil) != (tc.want == nil) || got.Cost != nil && *got.Cost != *tc.want {
				t.Fatalf("cost was lost or invented: %s", body)
			}
		})
	}
}
