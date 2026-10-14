// Purpose: Verify boolean Thinking discovery, wire settings and separate output.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestThinking.
// Related: internal/provider/diagnostic/control.go, internal/provider/protocol/wire.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"testing"
)

func TestThinking(t *testing.T) {
	client := mockClient(t, modelconn.Ollama, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ps" {
			fmt.Fprint(w, `{"models":[]}`)
			return
		}
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
	supported, err := thinkingControl(client, context.Background(), "model")
	if err != nil || !supported {
		t.Fatalf("discovery: %v %v", supported, err)
	}
	off := false
	reason := ""
	err = client.Generate(context.Background(), modelapi.Request{ModelID: "model", MaxOutput: 128, Thinking: &off, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(e modelapi.Event) error { reason += e.Thinking; return nil })
	if err != nil || reason != "inspect" {
		t.Fatalf("thinking: %q %v", reason, err)
	}
}
