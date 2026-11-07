// Purpose: Measure fragmented Gemini continuation ownership and verify terminal release.
// Pattern: Deterministic native stream fixtures.
// Usage: go test ./internal/provider/protocol/gemini -run TestContinuation -bench BenchmarkContinuation.
// Related: internal/provider/protocol/gemini/stream.go, internal/provider/protocol/gemini/types.go.
package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func BenchmarkContinuation(b *testing.B) {
	frame := []byte(`{"candidates":[{"content":{"parts":[{"text":"` + strings.Repeat("x", 1024) + `"}]}}]}`)
	final := []byte(`{"candidates":[{"finishReason":"STOP"}]}`)
	b.ReportAllocs()
	b.SetBytes(1 << 20)
	for b.Loop() {
		decoder := Decoder{}
		state := native.State{Streaming: true}
		for range 1024 {
			if err := decoder.Consume(frame, false, &state, func(modelapi.Event) error { return nil }); err != nil {
				b.Fatal(err)
			}
		}
		if err := decoder.Consume(final, false, &state, func(modelapi.Event) error { return nil }); err != nil {
			b.Fatal(err)
		}
	}
}

func TestContinuationKeepsPartsAndLocalCallIDs(t *testing.T) {
	decoder := Decoder{}
	state := native.State{Streaming: true}
	emit := func(modelapi.Event) error { return nil }
	frames := []string{
		`{"candidates":[{"content":{"parts":[{"text":"thought","thought":true,"thoughtSignature":"signed"}]}}]}`,
		`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"list_files","args":{"n":9007199254740993}},"thoughtSignature":"call-signature"}]}}]}`,
		`{"candidates":[{"finishReason":"STOP"}]}`,
	}
	for _, frame := range frames {
		if err := decoder.Consume([]byte(frame), false, &state, emit); err != nil {
			t.Fatal(err)
		}
	}
	var continuation Continuation
	if err := json.Unmarshal(state.Continuation, &continuation); err != nil {
		t.Fatal(err)
	}
	calls, err := state.Calls.Complete()
	if err != nil || len(calls) != 1 || len(continuation.CallIDs) != 1 || calls[0].ID != continuation.CallIDs[0] || len(continuation.Parts) != 2 || continuation.Parts[1].Signature != "call-signature" || string(continuation.Parts[1].Call.Args) != `{"n":9007199254740993}` {
		t.Fatalf("native part or identity changed: %s, %v", state.Continuation, err)
	}
}
