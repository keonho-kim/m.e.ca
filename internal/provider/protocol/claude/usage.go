// Purpose: Merge reported cumulative Claude counters without losing unchanged cache usage.
// Pattern: Response-owned usage projection.
// Usage: Message start, message deltas and complete JSON use Decoder.setUsage.
// Related: internal/provider/protocol/claude/stream.go, internal/provider/protocol/native/state.go.
package claude

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

type usage struct {
	Input   *int64 `json:"input_tokens"`
	Output  *int64 `json:"output_tokens"`
	Created *int64 `json:"cache_creation_input_tokens"`
	Cached  *int64 `json:"cache_read_input_tokens"`
}

func (d *Decoder) setUsage(value *usage, state *native.State) error {
	if value == nil {
		return nil
	}
	for _, pair := range []struct {
		next  *int64
		saved **int64
	}{
		{value.Input, &d.usage.Input}, {value.Output, &d.usage.Output},
		{value.Created, &d.usage.Created}, {value.Cached, &d.usage.Cached},
	} {
		if pair.next == nil {
			continue
		}
		if *pair.next < 0 || *pair.next > modelapi.MaxCallTokens {
			return fmt.Errorf("invalid Claude token usage")
		}
		*pair.saved = pair.next
	}
	if d.usage.Input == nil || d.usage.Output == nil {
		return nil
	}
	input := *d.usage.Input
	for _, counter := range []*int64{d.usage.Created, d.usage.Cached} {
		if counter != nil {
			input += *counter
		}
	}
	state.Usage = &modelapi.TokenUsage{Input: input, Output: *d.usage.Output}
	return state.Usage.Check()
}
