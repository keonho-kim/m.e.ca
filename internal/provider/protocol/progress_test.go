// Purpose: Distinguish real opaque reasoning progress from empty or repeated continuation fragments.
// Pattern: Provider stream event contract tests.
// Usage: go test ./internal/provider/protocol -run TestReasoningProgress.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/chat/reasoning.go.
package protocol

import (
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestReasoningProgressIgnoresEmptyAndRepeatedOpaqueFragments(t *testing.T) {
	s := streamState{provider: connection.OpenRouter}
	progress := 0
	emit := func(event modelapi.Event) error {
		if event.Progress {
			progress++
		}
		return nil
	}
	if err := s.consume([]byte(`{"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.text","text":""}]}}]}`), false, emit); err != nil {
		t.Fatal(err)
	}
	if progress != 0 {
		t.Fatal("empty reasoning extended semantic idle")
	}
	opaque := []byte(`{"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.encrypted","data":"opaque-reasoning"}]}}]}`)
	for range 2 {
		if err := s.consume(opaque, false, emit); err != nil {
			t.Fatal(err)
		}
	}
	if progress != 1 {
		t.Fatal("opaque progress missing or repeated", progress)
	}
}

func TestReasoningProgressIncludesNewCompletedResponsesItems(t *testing.T) {
	s := streamState{provider: connection.OpenAI}
	progress := 0
	emit := func(event modelapi.Event) error {
		if event.Progress {
			progress++
		}
		return nil
	}
	frame := []byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"rs_one","type":"reasoning","summary":[],"encrypted_content":"opaque"}}`)
	for range 2 {
		if err := s.consume(frame, false, emit); err != nil {
			t.Fatal(err)
		}
	}
	if progress != 1 {
		t.Fatal("completed opaque item did not mark one progress transition", progress)
	}
}
