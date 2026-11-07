// Purpose: Verify public progress/final equality and complete versus missing usage across tool turns.
// Pattern: Multi-generation engine contract test.
// Usage: go test ./internal/serverapi -run TestOpenAIOutputAcrossToolTurns.
// Related: internal/serverapi/openai/output.go, internal/storage/sqlite/response_output.go.
package serverapi

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestOpenAIOutputAcrossToolTurns(t *testing.T) {
	var calls atomic.Int32
	var missing atomic.Bool
	f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		if calls.Add(1)%2 == 1 {
			if err := emit(model.Event{Delta: "  checking \n", Streaming: true}); err != nil {
				return err
			}
			var usage *model.TokenUsage
			if !missing.Load() {
				usage = &model.TokenUsage{Input: 3, Output: 2}
			}
			return emit(model.Event{Calls: []tool.Call{{ID: "read", Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat source.txt"}`)}}, Usage: usage, Done: true})
		}
		return emit(model.Event{Delta: " answer \n", Streaming: true, Done: true, Usage: &model.TokenUsage{Input: 5, Output: 4}})
	}), false)
	if err := os.WriteFile(filepath.Join(f.project, "source.txt"), []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	transport := openAITransport(t, f, "sse")
	for _, unknown := range []bool{false, true} {
		missing.Store(unknown)
		reply := openAIRequest(t, transport, "POST", "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"read source"}],"stream":true,"stream_options":{"include_usage":true}}`, "")
		if reply.Code != 200 {
			t.Fatal(reply.Code, reply.Body.String())
		}
		reader := bufio.NewReader(strings.NewReader(reply.Body.String()))
		var text strings.Builder
		for {
			frame := nextCompatibleFrame(t, reader)
			if frame.Type == "done" {
				break
			}
			if frame.Error != nil {
				t.Fatal(frame.Error.Code, reply.Body.String())
			}
			for _, choice := range frame.Choices {
				text.WriteString(choice.Delta.Content)
			}
		}
		if text.String() != "checking\n\nanswer" {
			t.Fatal(text.String())
		}
		var usage json.RawMessage
		for _, line := range strings.Split(reply.Body.String(), "\n") {
			if !strings.HasPrefix(line, "data: {") {
				continue
			}
			var chunk struct {
				Choices []any
				Usage   json.RawMessage
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil {
				t.Fatal(err)
			}
			if len(chunk.Choices) == 0 {
				usage = chunk.Usage
			}
		}
		if unknown && string(usage) != "null" || !unknown && !strings.Contains(string(usage), `"total_tokens":14`) {
			t.Fatal("incorrect aggregate usage", string(usage))
		}
		plain := openAIRequest(t, transport, "POST", "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"read source"}]}`, "")
		var completion struct {
			Choices []struct{ Message struct{ Content string } }
		}
		if err := json.Unmarshal(plain.Body.Bytes(), &completion); err != nil || plain.Code != 200 || len(completion.Choices) != 1 {
			t.Fatal(plain.Code, plain.Body.String(), err)
		}
		if completion.Choices[0].Message.Content != text.String() {
			t.Fatal("stream differed from original", plain.Body.String())
		}
	}
}
