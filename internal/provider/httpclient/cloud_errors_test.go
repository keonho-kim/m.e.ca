// Purpose: Request visible thinking and stop repeated native safety, context and argument failures.
// Pattern: Local HTTP capability and terminal-state contract tests.
// Usage: go test ./internal/provider/httpclient -run TestCloud.
// Related: internal/provider/httpclient/request.go, internal/provider/transport/errors.go.
package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestCloudRequestsVisibleThinkingAndObservesUnsupportedModels(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			var count atomic.Int32
			client := mockClient(t, kind, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Reasoning  map[string]any
					Generation struct {
						Thinking map[string]any `json:"thinkingConfig"`
					} `json:"generationConfig"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				visible := body.Reasoning["summary"] == "auto"
				if kind == connection.Gemini {
					visible = body.Generation.Thinking["includeThoughts"] == true
				}
				request := count.Add(1)
				if request == 1 {
					if !visible {
						t.Error("visible thinking was not requested")
					}
					w.WriteHeader(http.StatusBadRequest)
					if kind == connection.OpenAI {
						fmt.Fprint(w, `{"error":{"code":"unsupported_parameter","param":"reasoning.summary","message":"reasoning.summary is not supported with this model"}}`)
					} else {
						fmt.Fprint(w, `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"Thinking is not supported for this model"}}`)
					}
					return
				}
				if visible {
					t.Error("unsupported summary option was replayed")
				}
				w.Header().Set("Content-Type", "application/json")
				writeProviderSuccess(w, kind, "answer")
			})
			for range 2 {
				if err := client.Generate(t.Context(), recoveryRequest(), func(modelapi.Event) error { return nil }); err != nil {
					t.Fatal(err)
				}
			}
			if count.Load() != 3 {
				t.Fatalf("capability result was not reused: %d", count.Load())
			}
		})
	}
}

func TestCloudHonorsDeclaredNoThinking(t *testing.T) {
	client := mockClient(t, connection.Gemini, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Config map[string]json.RawMessage `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, exists := body.Config["thinkingConfig"]; exists {
			t.Error("thought summary requested from a declared unsupported model")
		}
		w.Header().Set("Content-Type", "application/json")
		writeProviderSuccess(w, connection.Gemini, "answer")
	})
	if err := client.BindModel(connection.ModelInfo{ID: "alias", ThoughtSummary: new(false)}); err != nil {
		t.Fatal(err)
	}
	if err := client.Generate(t.Context(), recoveryRequest(), func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestCloudPermanentTerminalsDoNotRetry(t *testing.T) {
	for _, tc := range []struct {
		kind connection.ProviderKind
		body string
		sse  bool
	}{
		{connection.Claude, `{"type":"message","stop_reason":"model_context_window_exceeded","content":[],"usage":{"input_tokens":4,"output_tokens":2}}`, false},
		{connection.Claude, `{"type":"message","stop_reason":"refusal","content":[],"usage":{"input_tokens":4,"output_tokens":2}}`, false},
		{connection.Gemini, `{"promptFeedback":{"blockReason":"SAFETY"}}`, false},
		{connection.Gemini, `{"candidates":[{"finishReason":"SAFETY"}]}`, false},
		{connection.Gemini, `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"bad arguments"}}`, true},
	} {
		var count atomic.Int32
		client := mockClient(t, tc.kind, func(w http.ResponseWriter, r *http.Request) {
			count.Add(1)
			if tc.sse {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: "+tc.body+"\n\n")
			} else {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.body)
			}
		})
		if err := client.Generate(t.Context(), recoveryRequest(), func(modelapi.Event) error { return nil }); err == nil {
			t.Error("terminal failure was ignored")
		}
		if count.Load() != 1 {
			t.Errorf("permanent terminal was submitted %d times", count.Load())
		}
	}
}
