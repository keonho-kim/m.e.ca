// Purpose: Enforce the shared response budget on hidden block-start payloads as well as visible deltas.
// Pattern: Controlled decoder boundary tests without large allocations.
// Usage: go test ./internal/provider/protocol/claude -run TestHidden.
// Related: internal/provider/protocol/claude/stream.go, internal/provider/protocol/claude/complete.go.
package claude

import (
	"errors"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func TestHiddenBlockUsesResponseByteBudget(t *testing.T) {
	for _, block := range []string{`{"type":"redacted_thinking","data":"hidden"}`, `{"type":"thinking","thinking":"","signature":"hidden"}`, `{"type":"tool_use","id":"a","name":"read_file","input":{"path":"a"}}`} {
		d, state := Decoder{bytes: modelapi.MaxResponseBytes}, native.State{}
		err := d.Consume([]byte(`{"type":"content_block_start","index":0,"content_block":`+block+`}`), false, &state, func(modelapi.Event) error { t.Fatal("over-budget block emitted"); return nil })
		var limited *modelapi.OutputLimitError
		if !errors.As(err, &limited) || len(d.blocks) > 0 {
			t.Fatal("hidden bytes bypassed response budget", err)
		}
	}
}
