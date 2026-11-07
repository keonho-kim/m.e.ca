// Purpose: Measure completed Responses retention without real model calls.
// Pattern: Immutable stream-fixture benchmark.
// Usage: go test ./internal/provider/protocol/responses -run '^$' -bench BenchmarkCompletedResponse -benchmem.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/responses/continuation.go.
package responses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func TestFunctionItemAliasesCannotGrowWithoutBound(t *testing.T) {
	decoder := Decoder{}
	state := native.State{Streaming: true}
	for i := range toolapi.MaxCalls + 1 {
		frame := []byte(fmt.Sprintf(`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"item_%d","call_id":"same-call","name":"list_files"}}`, i))
		err := decoder.Consume(frame, false, &state, func(modelapi.Event) error { return nil })
		if i < toolapi.MaxCalls && err != nil {
			t.Fatal(err)
		}
		if i == toolapi.MaxCalls && err == nil {
			t.Fatal("unbounded item identities sharing one call index")
		}
	}
}

func TestCompletedResponseReleasesDraftsAndKeepsOpaqueOutput(t *testing.T) {
	for _, final := range []string{``, `,"output":[{"type":"message","content":[{"type":"output_text","text":"answer"}],"opaque":9007199254740993}]`} {
		decoder := Decoder{Retain: true}
		state := native.State{Streaming: true}
		item := []byte(`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","content":[{"type":"output_text","text":"answer"}],"opaque":9007199254740993}}`)
		emit := func(modelapi.Event) error { return nil }
		if err := decoder.Consume(item, false, &state, emit); err != nil {
			t.Fatal(err)
		}
		clear(item) // The borrowed stream frame can be reused immediately.
		if err := decoder.Consume([]byte(`{"type":"response.completed","response":{"status":"completed"`+final+`}}`), false, &state, emit); err != nil {
			t.Fatal(err)
		}
		if !state.Finished || !bytes.Contains(state.Continuation, []byte(`9007199254740993`)) || !bytes.Contains(state.Continuation, []byte(`answer`)) {
			t.Fatalf("lost original output: %s", state.Continuation)
		}
		if decoder.output != nil || decoder.items != nil || decoder.bytes != 0 {
			t.Fatal("completed response retained intermediate output")
		}
	}
}

func TestFailedResponseReleasesDrafts(t *testing.T) {
	decoder := Decoder{Retain: true}
	state := native.State{Streaming: true}
	emit := func(modelapi.Event) error { return nil }
	if err := decoder.Consume([]byte(`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","content":[]}}`), false, &state, emit); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Consume([]byte(`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`), false, &state, emit); err == nil {
		t.Fatal("incomplete output was accepted")
	}
	if decoder.output != nil || decoder.items != nil || decoder.bytes != 0 || len(state.Continuation) != 0 {
		t.Fatal("failed response retained output")
	}
}

func TestCompletedOutputRejectsMalformedText(t *testing.T) {
	for _, item := range []string{
		`{"type":"message","content":"invalid"}`,
		`{"type":"message","content":[{"text":42}]}`,
		`{"type":"reasoning","summary":[{"text":true}]}`,
	} {
		decoder := Decoder{Retain: true}
		state := native.State{Streaming: true}
		frame := []byte(`{"type":"response.completed","response":{"status":"completed","output":[` + item + `]}}`)
		if err := decoder.Consume(frame, false, &state, func(modelapi.Event) error { return nil }); err == nil {
			t.Fatalf("malformed output was completed: %s", item)
		}
	}
}

func TestEmptyOutputDoesNotCreateContinuation(t *testing.T) {
	decoder := Decoder{Retain: true}
	state := native.State{Streaming: true}
	if err := decoder.Consume([]byte(`{"type":"response.completed","response":{"status":"completed","output":[ ]}}`), false, &state, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(state.Continuation) != 0 {
		t.Fatal("empty native output became a replayable continuation")
	}
}

func BenchmarkCompletedResponse(b *testing.B) {
	item := map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": strings.Repeat("x", 1<<20)}}, "opaque": json.Number("9007199254740993")}
	done, _ := json.Marshal(map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})
	complete, _ := json.Marshal(map[string]any{"type": "response.completed", "response": map[string]any{"status": "completed", "output": []any{item}}})
	b.SetBytes(int64(len(done) + len(complete)))
	b.ReportAllocs()
	for b.Loop() {
		decoder := Decoder{Retain: true}
		state := native.State{Streaming: true}
		for _, frame := range [][]byte{done, complete} {
			if err := decoder.Consume(frame, false, &state, func(modelapi.Event) error { return nil }); err != nil {
				b.Fatal(err)
			}
		}
	}
}
