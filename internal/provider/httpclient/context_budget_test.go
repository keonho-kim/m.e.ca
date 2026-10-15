// Purpose: Verify configured context budgets respect discovered limits and preserve serving aliases.
// Pattern: HTTP request contract tests.
// Usage: go test ./internal/provider/httpclient -run TestConfiguredContext.
// Related: internal/provider/httpclient/context_budget.go, internal/provider/httpclient/context.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"testing"
)

func TestConfiguredContextOllama(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		configured, reported, want int
	}{{"default", 0, 0, 262144}, {"override", 8192, 0, 8192}, {"smaller server", 8192, 4096, 4096}, {"smaller config", 8192, 32768, 8192}} {
		t.Run(tc.name, func(t *testing.T) {
			c := mockClient(t, connection.Ollama, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Model   string
					Options struct {
						NumCtx int `json:"num_ctx"`
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Model != "Qwen-3.8-180B" || body.Options.NumCtx != tc.want {
					t.Errorf("body=%+v", body)
				}
				fmt.Fprint(w, `{"message":{"content":"answer"},"done":true}`)
			})
			if tc.configured > 0 {
				if err := c.SetContextLength(tc.configured); err != nil {
					t.Fatal(err)
				}
			}
			if tc.reported > 0 {
				if err := c.BindModel(connection.ModelInfo{ID: "Qwen-3.8-180B", ContextWindow: tc.reported}); err != nil {
					t.Fatal(err)
				}
			}
			request := modelapi.Request{ModelID: "Qwen-3.8-180B", MaxOutput: 100, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}
			if err := c.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if c.modelInfo != nil && c.modelInfo.ContextWindow != tc.reported {
				t.Fatal("server metadata changed")
			}
		})
	}
}

func TestConfiguredContextVLLMReservation(t *testing.T) {
	c := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			fmt.Fprint(w, `{"count":1000,"max_model_len":2048}`)
			return
		}
		var body struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.MaxTokens != 24 {
			t.Errorf("output=%d", body.MaxTokens)
		}
		fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
	})
	if err := c.SetContextLength(1024); err != nil {
		t.Fatal(err)
	}
	if err := c.BindModel(connection.ModelInfo{ID: "alias", ContextWindow: 2048}); err != nil {
		t.Fatal(err)
	}
	if err := c.Generate(context.Background(), modelapi.Request{ModelID: "alias", MaxOutput: 0, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
