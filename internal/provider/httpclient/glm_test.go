// Purpose: Verify vLLM GLM Thinking switches and current reasoning fields without native tool schemas.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestVLLMGLM.
// Related: internal/provider/httpclient/thinking.go, internal/provider/httpclient/request.go, internal/provider/httpclient/wire.go.
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

func TestVLLMGLMThinkingControl(t *testing.T) {
	for _, on := range []bool{false, true} {
		client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Template struct {
					Enable *bool `json:"enable_thinking"`
				} `json:"chat_template_kwargs"`
				Tools json.RawMessage `json:"tools"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if body.Template.Enable == nil || *body.Template.Enable != on || len(body.Tools) != 0 {
				t.Errorf("incorrect staged GLM request: %+v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"reasoning":"inspect","content":"DONE\nanswer"},"finish_reason":"stop"}]}`)
		})
		supported, err := client.ThinkingControl(context.Background(), "zai-org/GLM-4.5-Air")
		if err != nil || !supported {
			t.Fatalf("GLM control: %t %v", supported, err)
		}
		var reasoning string
		err = client.Generate(context.Background(), agent.ModelRequest{ModelID: "zai-org/GLM-4.5-Air", MaxOutput: 128, Thinking: &on, Messages: []agent.Message{{Role: "user", Content: "hello"}}}, func(e agent.ModelEvent) error { reasoning += e.Thinking; return nil })
		if err != nil || reasoning != "inspect" {
			t.Fatalf("reasoning: %q %v", reasoning, err)
		}
	}
}
func TestVLLMGLMDiagnosticRejectsIgnoredThinkingOff(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"reasoning":"unexpected","content":"TOOL read_file"},"finish_reason":"stop"}]}`)
	})
	if caps, err := client.Diagnose(context.Background(), "GLM-4.5-Air"); err == nil || caps.ToolDialogue {
		t.Fatal("server ignoring Thinking OFF passed connection diagnostics")
	}
}

func TestReasoningContentIsNotConsumed(t *testing.T) {
	state := streamState{}
	var thinking string
	err := state.consume([]byte(`{"choices":[{"message":{"reasoning_content":"obsolete output","content":"hello"},"finish_reason":"stop"}]}`), false, true, func(event agent.ModelEvent) error {
		thinking += event.Thinking
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if thinking != "" {
		t.Fatalf("obsolete reasoning field consumed: %q", thinking)
	}
}
