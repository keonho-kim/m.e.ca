// Purpose: Verify LM Studio Responses preserves history, exact usage and reasoning streams.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestLMStudioResponses.
// Related: internal/provider/protocol/lmstudio.go, internal/provider/protocol/request.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/protocol"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLMStudioResponses(t *testing.T) {
	c := mockClient(t, connection.LMStudio, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/models" {
			fmt.Fprint(w, `{"models":[{"key":"weights","type":"llm","loaded_instances":[{"id":"alias","config":{"context_length":2048}}],"max_context_length":131072,"capabilities":{"reasoning":{"allowed_options":["off","low","high"],"default":"low"}}}]}`)
			return
		}
		if r.URL.Path != "/v1/responses" {
			t.Errorf("path=%s", r.URL.Path)
		}
		var body struct {
			Input      []modelapi.Message
			Store      bool
			Truncation string
			MaxOutput  int `json:"max_output_tokens"`
			Reasoning  struct{ Effort string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Input) != 4 || body.Input[2].Role != "assistant" || body.Store || body.Truncation != "disabled" || body.Reasoning.Effort != "high" || body.MaxOutput != 2047 {
			t.Errorf("request=%+v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.reasoning_text.delta\",\"delta\":\"reason\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"answer\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":20,\"output_tokens\":7}}}\n\n")
	})
	info, err := prepareModel(c, context.Background(), "alias")
	if err != nil {
		t.Fatal(err)
	}
	if info.ContextWindow != 2048 {
		t.Fatalf("limits=%+v", info)
	}
	on := true
	var answer, reason string
	var usage *modelapi.TokenUsage
	err = c.Generate(context.Background(), modelapi.Request{ModelID: "alias", MaxOutput: 4096, Thinking: &on, ReasoningEffort: "high", Messages: []modelapi.Message{{Role: "system", Content: "rules"}, {Role: "user", Content: "first"}, {Role: "assistant", Content: "old answer"}, {Role: "user", Content: "second"}}}, func(e modelapi.Event) error {
		answer += e.Delta
		reason += e.Thinking
		if e.Done {
			usage = e.Usage
		}
		return nil
	})
	if err != nil || answer != "answer" || reason != "reason" || usage == nil || usage.Input != 20 || usage.Output != 7 {
		t.Fatalf("answer=%s reason=%s usage=%+v err=%v", answer, reason, usage, err)
	}
}

func TestLMStudioIncompleteRetriesWithUsage(t *testing.T) {
	calls := 0
	c := mockClient(t, connection.LMStudio, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\ndata: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"usage\":{\"input_tokens\":20,\"output_tokens\":7}}}\n\n")
	})
	usages, retries := 0, 0
	err := c.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error {
		if e.Usage != nil {
			usages++
			if e.Usage.Input != 20 || e.Usage.Output != 7 {
				t.Errorf("usage=%+v", e.Usage)
			}
		}
		if e.Retry != nil {
			retries++
			if usages != retries {
				t.Error("retry lost preceding usage")
			}
		}
		if e.Done {
			t.Error("incomplete response completed")
		}
		return nil
	})
	if err == nil || calls != 3 || usages != 3 || !strings.Contains(err.Error(), "max_output_tokens") {
		t.Fatalf("calls=%d usages=%d err=%v", calls, usages, err)
	}
}

func TestLMStudioDoneMarkerDoesNotProveCompletion(t *testing.T) {
	for _, tail := range []string{"", "data: [DONE]\n\n"} {
		response := &http.Response{Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n" + tail))}
		if err := protocol.ReadResponse(context.Background(), response, connection.LMStudio, func(e modelapi.Event) error {
			if e.Done {
				t.Error("truncated stream completed")
			}
			return nil
		}); err == nil {
			t.Fatal("truncated Responses accepted")
		}
	}
}
