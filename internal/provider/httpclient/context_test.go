// Purpose: Verify exact server tokenization reserves output without truncating conversation history.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestContextCapacity.
// Related: internal/provider/httpclient/context.go, internal/provider/protocol/request.go.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"net/http"
	"strings"
	"testing"
)

func TestContextCapacity(t *testing.T) {
	for _, count := range []int{90, 100} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			calls := 0
			c := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/models" {
					fmt.Fprint(w, `{"data":[{"id":"alias","max_model_len":100}]}`)
					return
				}
				if r.URL.Path == "/tokenize" {
					fmt.Fprintf(w, `{"count":%d,"max_model_len":100}`, count)
					return
				}
				calls++
				var body struct {
					MaxTokens int `json:"max_tokens"`
					Messages  []protocol.Message
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.MaxTokens != 10 || len(body.Messages) != 1 {
					t.Errorf("request=%+v", body)
				}
				fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
			})
			if _, err := prepareModel(c, context.Background(), "alias"); err != nil {
				t.Fatal(err)
			}
			err := c.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error {
				if e.Usage != nil {
					t.Error("tokenization presented as billable usage")
				}
				return nil
			})
			if count == 100 {
				if err == nil || calls != 0 || !strings.Contains(err.Error(), "context") {
					t.Fatalf("overflow calls=%d err=%v", calls, err)
				}
				var capacity *modelapi.ContextLimitError
				if !errors.As(err, &capacity) || capacity.Input != count || capacity.Capacity != 100 {
					t.Fatal("tokenizer overflow cannot enter runtime compaction recovery:", err)
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestModelListRetriesMalformedLimit(t *testing.T) {
	calls := 0
	c := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			fmt.Fprint(w, `{"data":[{"id":"alias","max_model_len":"invalid"}]}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"alias","max_model_len":100}]}`)
	})
	models, err := catalogFor(c).ListModels(context.Background())
	if err != nil || calls != 2 || len(models) != 1 || models[0].ContextWindow != 100 {
		t.Fatalf("models=%+v calls=%d err=%v", models, calls, err)
	}
}
