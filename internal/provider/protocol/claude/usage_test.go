// Purpose: Preserve independent cumulative Claude token counters across partial usage deltas.
// Pattern: Native stream state contract tests.
// Usage: go test ./internal/provider/protocol/claude.
// Related: internal/provider/protocol/claude/stream.go.
package claude

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"testing"
)

func TestUsageMergesOnlyReportedCounters(t *testing.T) {
	for _, value := range []string{`"input_tokens":12`, `"cache_creation_input_tokens":21,"cache_read_input_tokens":31`} {
		d, state := Decoder{}, native.State{}
		for _, frame := range []string{
			`{"type":"message_start","message":{"usage":{"input_tokens":10,"cache_creation_input_tokens":20,"cache_read_input_tokens":30,"output_tokens":0}}}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{` + value + `,"output_tokens":4}}`,
			`{"type":"message_stop"}`,
		} {
			if err := d.Consume([]byte(frame), false, &state, func(modelapi.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
		if state.Usage == nil || state.Usage.Input != 62 || state.Usage.Output != 4 {
			t.Fatalf("lost cumulative counters: %+v", state.Usage)
		}
	}
}
