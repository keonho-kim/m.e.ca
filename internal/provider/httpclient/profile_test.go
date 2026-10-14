// Purpose: Verify family controls and explicit aliases preserve served IDs and reject guessed toggles.
// Pattern: Provider HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestProfile.
// Related: internal/provider/profile/profile.go, internal/provider/diagnostic/control.go, internal/provider/protocol/request.go.
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

func TestProfileThinkingWireContract(t *testing.T) {
	for _, model := range []string{"Qwen/Qwen3.6-27B", "ornith-ai/Ornith-1.5-35B-A3B", "nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "deepseek-ai/DeepSeek-V4-Flash", "LGAI-EXAONE/EXAONE-4.5-33B", "google/gemma-4-31B-it"} {
		for _, on := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", model, on), func(t *testing.T) {
				client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Model    string `json:"model"`
						Template struct {
							Enable *bool `json:"enable_thinking"`
						} `json:"chat_template_kwargs"`
						Tools json.RawMessage `json:"tools"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					if body.Model != model || body.Template.Enable == nil || *body.Template.Enable != on || len(body.Tools) > 0 {
						t.Errorf("incorrect profile request: %+v", body)
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"choices":[{"message":{"content":"DONE\nanswer"},"finish_reason":"stop"}]}`)
				})
				if ok, err := thinkingControl(client, context.Background(), model); err != nil || !ok {
					t.Fatalf("control=%t %v", ok, err)
				}
				if err := client.Generate(context.Background(), modelapi.Request{ModelID: model, MaxOutput: 128, Thinking: &on, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestProfileExplicitAliasKeepsRequestModel(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model    string          `json:"model"`
			Template map[string]bool `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Model != "local-coder" || body.Template["enable_thinking"] {
			t.Errorf("alias changed or Thinking enabled: %+v", body)
		}
		if _, ok := body.Template["enable_thinking"]; !ok {
			t.Error("explicit profile was not applied")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
	})
	client.profileID = "ornith-1.5-9b"
	off := false
	if err := client.Generate(context.Background(), modelapi.Request{ModelID: "local-coder", MaxOutput: 128, Thinking: &off, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestProfileUnknownAliasDoesNotGuessThinking(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) { t.Error("unknown control reached server") })
	if ok, err := thinkingControl(client, context.Background(), "local-coder"); err != nil || ok {
		t.Fatalf("guessed control=%t %v", ok, err)
	}
	off := false
	if err := client.Generate(context.Background(), modelapi.Request{ModelID: "local-coder", MaxOutput: 128, Thinking: &off, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err == nil {
		t.Fatal("unknown Thinking control accepted")
	}
}

func TestProfileDiagnosticVerifiesBothThinkingStates(t *testing.T) {
	for _, onProducesReasoning := range []bool{false, true} {
		t.Run(fmt.Sprint(onProducesReasoning), func(t *testing.T) {
			client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Messages []modelapi.Message `json:"messages"`
					Template map[string]bool    `json:"chat_template_kwargs"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				reply := `{"tool":"read_file"}`
				reasoning := ""
				if body.Template["enable_thinking"] {
					reply = "391"
					if onProducesReasoning {
						reasoning = "17 times 23 is 391"
					}
				} else if len(body.Messages) > 2 {
					reply = "AGENTS.md"
				}
				w.Header().Set("Content-Type", "application/json")
				response := map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply, "reasoning": reasoning}, "finish_reason": "stop"}}}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			})
			caps, err := diagnose(client, context.Background(), "Qwen/Qwen3.6-27B")
			if err != nil || !caps.ToolDialogue || caps.Thinking != onProducesReasoning {
				t.Fatalf("capabilities=%+v err=%v", caps, err)
			}
		})
	}
}

func TestProfileSGLangUsesItsDeclaredReasoningField(t *testing.T) {
	client := mockClient(t, modelconn.SGLang, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Template map[string]bool `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if !body.Template["enable_thinking"] {
			t.Error("SGLang control missing")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"answer\",\"reasoning_content\":\"inspect\"}}]}\n\ndata: [DONE]\n\n")
	})
	on := true
	var reasoning string
	err := client.Generate(context.Background(), modelapi.Request{ModelID: "ornith-ai/Ornith-1.5-9B", Thinking: &on, MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(e modelapi.Event) error { reasoning += e.Thinking; return nil })
	if err != nil || reasoning != "inspect" {
		t.Fatalf("SGLang reasoning=%q err=%v", reasoning, err)
	}
}

func TestProfileDiagnosticRejectsIgnoredThinkingOff(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"reasoning":"unexpected","content":"{\"tool\":\"read_file\"}"},"finish_reason":"stop"}]}`)
	})
	if caps, err := diagnose(client, context.Background(), "Qwen/Qwen3.6-27B"); err == nil || caps.ToolDialogue {
		t.Fatal("server ignoring Thinking OFF passed connection diagnostics")
	}
}
