// Purpose: Verify boolean Thinking discovery, wire settings and separate output.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestThinking.
// Related: internal/provider/httpclient/thinking.go, internal/provider/httpclient/wire.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func TestThinking(t *testing.T) {
	client := mockClient(t, modelconn.Ollama, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/show" {
			fmt.Fprint(w, `{"thinking":{"values":[true,false]}}`)
			return
		}
		var payload map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if string(payload["think"]) != "false" {
			t.Errorf("think = %s", payload["think"])
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"message":{"thinking":"inspect","content":"answer"},"done":true}`)
	})
	supported, err := client.ThinkingControl(context.Background(), "model")
	if err != nil || !supported {
		t.Fatalf("discovery: %v %v", supported, err)
	}
	off := false
	reason := ""
	err = client.Generate(context.Background(), agent.ModelRequest{ModelID: "model", MaxOutput: 128, Thinking: &off, Messages: []agent.Message{{Role: "user", Content: "hello"}}}, func(e agent.ModelEvent) error { reason += e.Thinking; return nil })
	if err != nil || reason != "inspect" {
		t.Fatalf("thinking: %q %v", reason, err)
	}
}
