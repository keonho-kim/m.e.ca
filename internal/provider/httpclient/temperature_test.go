// Purpose: Verify sampling transport, explicit provider-default policy and request replay on retries.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestTemperature.
// Related: internal/provider/protocol/request.go, internal/provider/transport/retry.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestTemperatureProviderRequestsAndRetries(t *testing.T) {
	for _, provider := range connection.Providers() {
		for _, value := range []float64{0, 0.2, 0.4, 0.7} {
			t.Run(fmt.Sprintf("%s/%g", provider.Kind, value), func(t *testing.T) {
				var calls atomic.Int32
				client := mockClient(t, provider.Kind, func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Temperature *float64 `json:"temperature"`
						Generation  struct {
							Temperature *float64 `json:"temperature"`
						} `json:"generationConfig"`
						Options struct {
							Temperature *float64 `json:"temperature"`
						} `json:"options"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					got := body.Temperature
					if provider.Kind == connection.Ollama {
						got = body.Options.Temperature
						if body.Temperature != nil {
							t.Error("native temperature leaked to top level")
						}
					}
					if provider.Kind == connection.Gemini {
						got = body.Generation.Temperature
					}
					if provider.Kind == connection.OpenAI || provider.Kind == connection.Claude {
						if got != nil {
							t.Error("unsupported sampling override sent to first-party API")
						}
					} else if got == nil || *got != value {
						t.Errorf("temperature lost: %+v", body)
					}
					if calls.Add(1) == 1 {
						http.Error(w, "temporary", http.StatusServiceUnavailable)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					writeProviderSuccess(w, provider.Kind, "ok")
				})
				request := recoveryRequest()
				request.Temperature = &value
				if err := client.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 2 {
					t.Fatalf("attempts=%d", calls.Load())
				}
			})
		}
	}
}
