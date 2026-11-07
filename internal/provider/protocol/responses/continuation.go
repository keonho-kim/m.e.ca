// Purpose: Retain completed OpenAI Responses items for stateless signed reasoning replay.
// Pattern: Response-owned continuation snapshot.
// Usage: Decoder preserves completed output after its successful terminal response only.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/response_input.go.
package responses

import (
	"bytes"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"sort"
)

func (d *Decoder) remember(index *int, raw json.RawMessage) error {
	if !d.Retain {
		return nil
	}
	if index == nil || *index < 0 || *index >= 1024 {
		return fmt.Errorf("invalid Responses output index")
	}
	if d.output == nil {
		d.output = map[int]json.RawMessage{}
	}
	d.bytes -= len(d.output[*index])
	d.bytes += len(raw)
	if d.bytes > modelapi.MaxResponseBytes {
		return &modelapi.OutputLimitError{Reason: "Responses continuation byte limit"}
	}
	// RawMessage was allocated by this frame's JSON decode. Transfer ownership;
	// the stream reader may reuse content without affecting this snapshot.
	d.output[*index] = raw
	return nil
}

func (d *Decoder) retain(raw json.RawMessage, state *native.State) error {
	if !d.Retain {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) >= 2 && trimmed[0] == '[' && len(bytes.TrimSpace(trimmed[1:len(trimmed)-1])) == 0 {
		return nil
	}
	body := raw
	if len(raw) == 0 || string(raw) == "null" {
		indices := make([]int, 0, len(d.output))
		for index := range d.output {
			indices = append(indices, index)
		}
		sort.Ints(indices)
		if len(indices) == 0 {
			return nil
		}
		output := make([]json.RawMessage, 0, len(indices))
		for _, index := range indices {
			output = append(output, d.output[index])
		}
		var err error
		body, err = json.Marshal(output)
		if err != nil {
			return err
		}
	}
	if len(body) > modelapi.MaxResponseBytes {
		return &modelapi.OutputLimitError{Reason: "Responses continuation byte limit"}
	}
	state.Continuation = body
	return nil
}
