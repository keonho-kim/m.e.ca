// Purpose: Verify finalized Responses text reaches the journal once and empty completions recover within a decision.
// Pattern: Wire-to-run and scripted decision regression tests.
// Usage: go test ./internal/agent/run -run 'TestResponsesFinal|TestEmptyCompletion'.
// Related: internal/provider/protocol/responses/complete.go, internal/agent/run/generate.go, internal/agent/run/loop.go.
package run

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
)

func TestResponsesFinal(t *testing.T) {
	const delta = `{"type":"response.output_text.delta","sequence_number":1,"output_index":0,"content_index":0,"item_id":"m","delta":"Alpha"}`
	const done = `{"type":"response.output_text.done","sequence_number":3,"output_index":0,"content_index":0,"item_id":"m","text":"Alpha Beta"}`
	const final = `{"type":"response.completed","response":{"status":"completed","output":[{"type":"message","id":"m","content":[{"type":"output_text","text":"Alpha Beta"}]}]}}`
	const omitted = `{"type":"response.completed","response":{"status":"completed"}}`
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.LMStudio} {
		for _, tc := range []struct {
			name, want string
			frames     []string
		}{
			{"final only", "Alpha Beta", []string{final}},
			{"part done only", "Alpha Beta", []string{done, omitted}},
			{"item done only", "Alpha Beta", []string{`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"m","content":[{"type":"output_text","text":"Alpha Beta"}]}}`, omitted}},
			{"partial delta", "Alpha Beta", []string{delta, final}},
			{"duplicate sequence", "Alpha Beta", []string{delta, delta, done, final}},
			{"part envelope only", "Alpha Beta", []string{`{"type":"response.content_part.done","output_index":0,"content_index":0,"part":{"type":"output_text","text":"Alpha Beta"}}`, omitted}},
			{"refusal envelope only", "Refused", []string{`{"type":"response.content_part.done","output_index":0,"content_index":0,"part":{"type":"refusal","refusal":"Refused"}}`, omitted}},
			{"item removes obsolete part", "Final", []string{`{"type":"response.output_text.delta","output_index":0,"content_index":1,"delta":"Old"}`, `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","content":[{"type":"output_text","text":"Final"}]}}`, omitted}},
			{"part by bound identity", "FirstSecond", []string{`{"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"second"}}`, `{"type":"response.output_text.delta","item_id":"second","delta":"Second"}`, `{"type":"response.output_text.delta","output_index":0,"delta":"First"}`, omitted}},
			{"different terminal", "Alpha Beta", []string{strings.Replace(delta, "Alpha", "Other", 1), final}},
			{"legitimate repetition", "AlphaAlpha", []string{delta, strings.Replace(delta, `number":1`, `number":2`, 1), omitted}},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				j := &journal{}
				r := fixture(&generator{}, &executor{}, j)
				calls := 0
				r.Generator = statisticsModel(func(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
					calls++
					var body strings.Builder
					for _, frame := range tc.frames {
						fmt.Fprintf(&body, "data: %s\n\n", frame)
					}
					return protocol.ReadResponse(ctx, &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body.String()))}, kind, emit)
				})
				result, err := r.Run(t.Context(), "Answer.", approve, emit)
				if err != nil || calls != 1 || result.Text != tc.want || len(j.messages) != 2 || j.messages[1].Content != tc.want {
					t.Fatalf("calls=%d result=%q error=%v journal=%+v", calls, result.Text, err, j.messages)
				}
			})
		}
	}
}

func TestEmptyCompletionRecovery(t *testing.T) {
	for _, success := range []bool{true, false} {
		j, tools := &journal{}, &executor{}
		r := fixture(&generator{}, tools, j)
		calls := 0
		r.Generator = statisticsModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
			calls++
			if calls == 1 {
				if err := emit(modelapi.Event{Delta: "earlier draft", Streaming: true}); err != nil {
					return err
				}
				if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 4}}); err != nil {
					return err
				}
			} else if success {
				return emit(modelapi.Event{Delta: "correct answer", Done: true})
			}
			return emit(modelapi.Event{Thinking: "private thought", Done: true, Streaming: true})
		})
		result, err := r.Run(t.Context(), "Answer.", approve, emit)
		if calls != 2 || tools.executed != 0 || (err == nil) != success {
			t.Fatalf("calls=%d result=%s err=%v", calls, result.Status, err)
		}
		var answers []modelapi.Message
		for _, m := range j.messages {
			if m.Role == "assistant" {
				answers = append(answers, m)
			}
		}
		if len(answers) != 1 || answers[0].Incomplete == success {
			t.Fatalf("answers=%+v", answers)
		}
		if success && (result.Status != record.Answered || answers[0].Content != "correct answer") || !success && (result.Status != record.Stopped || answers[0].Content != "earlier draft") {
			t.Fatal(result, answers)
		}
	}
}

func TestFinalizedPartSurvivesMissingTerminal(t *testing.T) {
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	r.Generator = statisticsModel(func(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		const frame = `data: {"type":"response.output_text.done","output_index":0,"content_index":0,"text":"saved draft"}`
		response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(frame + "\n\n"))}
		return protocol.ReadResponse(ctx, response, connection.OpenAI, emit)
	})
	_, err := r.Run(t.Context(), "Answer.", approve, emit)
	if err == nil || len(j.messages) != 2 || !j.messages[1].Incomplete || j.messages[1].Content != "saved draft" {
		t.Fatalf("lost finalized part after disconnection: err=%v messages=%+v", err, j.messages)
	}
}

func TestThinkingOnlyCompletionCanRecoverWithATool(t *testing.T) {
	j, executor := &journal{}, &executor{}
	r := fixture(&generator{}, executor, j)
	decisions := 0
	r.Generator = statisticsModel(func(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		decisions++
		switch decisions {
		case 1:
			reply := `{"status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"Checking the file."}]}]}`
			wire := &http.Response{Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(reply))}
			return protocol.ReadResponse(ctx, wire, connection.OpenAI, emit)
		case 2:
			if len(request.Tools) == 0 {
				t.Fatal("empty-reply correction disabled tools")
			}
			call := `{"status":"completed","output":[{"type":"function_call","id":"fc","call_id":"unique","name":"codemode","arguments":"{\"code\":\"return await tools.inspect_fixture({path:'a'});\"}"}]}`
			wire := &http.Response{Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(call))}
			return protocol.ReadResponse(ctx, wire, connection.OpenAI, emit)
		default:
			return emit(modelapi.Event{Delta: "File checked.", Done: true})
		}
	})
	result, err := r.Run(t.Context(), "Check a", approve, emit)
	if err != nil || decisions != 3 || executor.executed != 1 || result.Text != "File checked." {
		t.Fatalf("decisions=%d executed=%d reply=%q err=%v", decisions, executor.executed, result.Text, err)
	}
}
