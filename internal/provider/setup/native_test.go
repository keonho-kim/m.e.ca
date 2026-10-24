// Purpose: Verify model-bound native effort controls survive complete setup without inventing OFF.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup -run TestNativeReasoningSelection.
// Related: internal/provider/setup/setup.go, internal/provider/discovery/model_settings.go.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeReasoningSelection(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Ollama, connection.LMStudio} {
		t.Run(string(kind), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/show":
					fmt.Fprint(w, `{"thinking":{"values":["low","high"],"default":"low"}}`)
					return
				case "/api/ps":
					fmt.Fprint(w, `{"models":[{"name":"alias","context_length":4096}]}`)
					return
				case "/api/v1/models":
					fmt.Fprint(w, `{"models":[{"type":"llm","key":"alias","loaded_instances":[{"id":"alias","config":{"context_length":4096}}],"capabilities":{"reasoning":{"allowed_options":["low","high"],"default":"low"}}}]}`)
					return
				}
				var body struct {
					Think     any
					Messages  []struct{ Content string }
					Input     []struct{ Content string }
					Reasoning struct{ Effort string }
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				messages := body.Messages
				if kind == connection.LMStudio {
					messages = body.Input
				}
				answer, reason := "Hello!", ""
				if len(messages) == 1 {
					answer, reason = "391", "calculate"
				}
				if body.Think == false || body.Reasoning.Effort == "none" {
					t.Error("unsupported OFF sent")
				}
				var response any
				if kind == connection.Ollama {
					response = map[string]any{"done": true, "message": map[string]string{"content": answer, "thinking": reason}}
				} else {
					response = map[string]any{"status": "completed", "output": []any{map[string]any{"type": "reasoning", "content": []any{map[string]string{"type": "reasoning_text", "text": reason}}}, map[string]any{"type": "message", "content": []any{map[string]string{"type": "output_text", "text": answer}}}}}
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			conn, err := connection.New(kind, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defaults, err := Prepare(context.Background(), conn, "", "alias", Defaults{}, connection.Selection{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if defaults.Selection.Thinking != nil || defaults.Selection.Effort() != "" {
				t.Fatalf("server default overridden: %+v", defaults.Selection)
			}
			defaults.Client.Close()
			on, level := true, "high"
			previous := connection.Selection{Connection: conn, Model: "alias", Thinking: &on, ReasoningEffort: &level, Details: &connection.ModelInfo{ID: "alias"}}
			prepared, err := Prepare(context.Background(), conn, "", "alias", Defaults{}, previous, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.Client.Close()
			selected := prepared.Selection
			if selected.ThinkingSupported || selected.Thinking == nil || !*selected.Thinking || selected.Effort() != "high" || selected.Details == nil || selected.Details.ContextWindow != 4096 {
				t.Fatalf("selection=%+v", selected)
			}
		})
	}
}
