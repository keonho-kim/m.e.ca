// Purpose: Verify serving API effort encoding, OFF omission and rejection before transport.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestEffort.
// Related: internal/provider/protocol/controls.go, internal/provider/connection/reasoning_options.go.
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

func TestEffortWireAndThinkingOff(t *testing.T) {
	for _, test := range []struct {
		model, effort string
		provider      modelconn.ProviderKind
		key           string
		value         any
	}{
		{"Qwen/Qwen3.8-27B", "medium", modelconn.VLLM, "reasoning_effort", "medium"},
		{"Qwen/Qwen3.8-27B", "low", modelconn.SGLang, "reasoning_effort", "low"},
		{"nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "low", modelconn.VLLM, "reasoning_effort", "low"},
		{"nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "high", modelconn.VLLM, "reasoning_effort", "high"},
		{"deepseek-ai/DeepSeek-V4-Flash", "max", modelconn.VLLM, "reasoning_effort", "max"},
	} {
		for _, on := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/%t", test.model, test.effort, on), func(t *testing.T) {
				client := mockClient(t, test.provider, func(w http.ResponseWriter, r *http.Request) {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					template, _ := body["chat_template_kwargs"].(map[string]any)
					actual, present := body[test.key]
					if test.provider == modelconn.SGLang {
						actual, present = template[test.key]
					}
					if on && (!present || actual != test.value) {
						t.Errorf("effort missing/wrong: %+v", body)
					}
					if !on {
						if _, present := body["reasoning_effort"]; present {
							t.Error("OFF sent reasoning_effort")
						}
						if _, present := template[test.key]; present {
							t.Error("OFF sent template effort")
						}
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
				})
				err := client.Generate(context.Background(), modelapi.Request{ModelID: test.model, Thinking: &on, ReasoningEffort: test.effort, MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil })
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestEffortInvalidValuesNeverReachServer(t *testing.T) {
	for _, value := range []string{"invented", "full", "none"} {
		client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid effort reached server") })
		on := true
		err := client.Generate(context.Background(), modelapi.Request{ModelID: "models/bin/model114", Thinking: &on, ReasoningEffort: value, MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil })
		if err == nil {
			t.Fatalf("accepted effort %q", value)
		}
	}
}
