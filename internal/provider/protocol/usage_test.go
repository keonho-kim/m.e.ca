// Purpose: Verify provider usage requests and cumulative stream snapshots, including exact zero counts.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestTokenUsage.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/request.go.
package protocol

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestTokenUsageAccompaniesTextAndDoesNotBecomeStale(t *testing.T) {
	state := streamState{provider: connection.VLLM, streaming: true}
	var events []modelapi.Event
	emit := func(e modelapi.Event) error { events = append(events, e); return nil }
	if err := state.consume([]byte(`{"choices":[{"delta":{"content":"hi"}}],"usage":{"prompt_tokens":12,"completion_tokens":1}}`), false, emit); err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if last.Delta != "hi" || last.Usage == nil {
		t.Fatal("text detached from matching usage")
	}
	if err := state.consume([]byte(`{"choices":[{"delta":{"content":" again"}}]}`), false, emit); err != nil {
		t.Fatal(err)
	}
	state.finished = true
	if err := state.complete(emit); err != nil {
		t.Fatal(err)
	}
	if events[len(events)-1].Usage != nil {
		t.Fatal("incomplete snapshot presented as final usage")
	}
}
func TestTokenUsageZeroIsReported(t *testing.T) {
	state := streamState{provider: connection.VLLM, streaming: true}
	var usage *modelapi.TokenUsage
	err := state.consume([]byte(`{"choices":[],"usage":{"prompt_tokens":0,"completion_tokens":0}}`), false, func(e modelapi.Event) error { usage = e.Usage; return nil })
	if err != nil || usage == nil || usage.Input != 0 || usage.Output != 0 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
}

func TestTokenUsageOnErrorIsDeliveredBeforeRetry(t *testing.T) {
	state := streamState{provider: connection.VLLM, streaming: true}
	var usage *modelapi.TokenUsage
	err := state.consume([]byte(`{"error":{"message":"engine failed"},"usage":{"prompt_tokens":12,"completion_tokens":3}}`), false, func(e modelapi.Event) error { usage = e.Usage; return nil })
	if err == nil || usage == nil || usage.Input != 12 || usage.Output != 3 {
		t.Fatalf("error usage=%+v err=%v", usage, err)
	}
}
