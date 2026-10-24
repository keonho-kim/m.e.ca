// Purpose: Verify reasoning options depend on the serving API rather than checkpoint names.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestAliasReasoningControls.
// Related: internal/provider/protocol/controls.go, internal/provider/diagnostic/check.go.
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

func TestAliasReasoningControls(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.VLLM, connection.SGLang} {
		t.Run(string(kind), func(t *testing.T) {
			client := mockClient(t, kind, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Model    string
					Template map[string]any `json:"chat_template_kwargs"`
					Effort   string         `json:"reasoning_effort"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if body.Model != "models/bin/model114" || body.Template["enable_thinking"] != true {
					t.Errorf("controls=%+v", body)
				}
				effort := body.Effort
				if kind == connection.SGLang {
					effort, _ = body.Template["reasoning_effort"].(string)
				}
				if effort != "low" {
					t.Errorf("effort=%q", effort)
				}
				w.Header().Set("Content-Type", "application/json")
				key := "reasoning"
				if kind == connection.SGLang {
					key = "reasoning_content"
				}
				fmt.Fprintf(w, `{"choices":[{"message":{"content":"answer",%q:"brief"},"finish_reason":"stop"}]}`, key)
			})
			on := true
			var thinking string
			err := client.Generate(context.Background(), modelapi.Request{ModelID: "models/bin/model114", Thinking: &on, ReasoningEffort: "low", MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(e modelapi.Event) error { thinking += e.Thinking; return nil })
			if err != nil || thinking != "brief" {
				t.Fatalf("thinking=%q err=%v", thinking, err)
			}
		})
	}
}

func TestServerDefaultDoesNotOverrideReasoning(t *testing.T) {
	for _, provider := range connection.Providers() {
		t.Run(string(provider.Kind), func(t *testing.T) {
			client := mockClient(t, provider.Kind, func(w http.ResponseWriter, r *http.Request) {
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				for _, field := range []string{"think", "reasoning_effort", "chat_template_kwargs", "reasoning"} {
					if _, exists := body[field]; exists {
						t.Errorf("server default overridden by %s", field)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				switch provider.Kind {
				case connection.Ollama:
					fmt.Fprint(w, `{"message":{"content":"answer"},"done":true}`)
				case connection.LMStudio:
					fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"answer"}]}]}`)
				default:
					fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
				}
			})
			err := client.Generate(context.Background(), modelapi.Request{ModelID: "models/bin/model114", MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
