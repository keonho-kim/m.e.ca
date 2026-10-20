// Purpose: Verify exact Ornith serving names retain ON/OFF controls with opaque vLLM model roots.
// Pattern: Provider connection and generation integration test.
// Usage: go test ./internal/provider/setup -run TestThinkingOrnithLocalRoot.
// Related: internal/provider/setup/identity.go, internal/provider/protocol/controls.go.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

func TestThinkingOrnithLocalRoot(t *testing.T) {
	for _, on := range []bool{false, true} {
		t.Run(fmt.Sprintf("on=%t", on), func(t *testing.T) {
			var generated bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v1/models" {
					fmt.Fprint(w, `{"data":[{"id":"ornith-1.5-35B-A3B","root":"/models/checkpoint","owned_by":"vllm"}]}`)
					return
				}
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body struct {
					Model    string
					Template map[string]bool `json:"chat_template_kwargs"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				value, present := body.Template["enable_thinking"]
				if body.Model != "ornith-1.5-35B-A3B" || !present || value != on {
					t.Errorf("incorrect model or Thinking request: %+v", body)
				}
				generated = true
				reasoning := ""
				if on {
					reasoning = "A brief calculation"
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "391", "reasoning": reasoning}, "finish_reason": "stop"}}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			conn, err := connection.New(connection.VLLM, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := Prepare(context.Background(), conn, "", "ornith-1.5-35B-A3B", Defaults{Thinking: on}, connection.Selection{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.Client.Close()
			selected := prepared.Selection
			if generated || selected.Profile != "ornith-1.5-35b-a3b" || !selected.ThinkingSupported || selected.Thinking == nil || *selected.Thinking != on || selected.Identity.ProfileSource != "catalogue" {
				t.Fatalf("unexpected setup: generated=%t selection=%+v", generated, selected)
			}
			var answer, thinking string
			var done bool
			err = prepared.Client.Generate(context.Background(), modelapi.Request{ModelID: selected.Model, Thinking: selected.Thinking, MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "17 times 23"}}}, func(event modelapi.Event) error {
				answer += event.Delta
				thinking += event.Thinking
				done = done || event.Done
				return nil
			})
			if err != nil || !generated || !done || answer != "391" || (thinking != "") != on {
				t.Fatalf("generation: answer=%q thinking=%q done=%t err=%v", answer, thinking, done, err)
			}
		})
	}
}
