// Purpose: Hold observed generation state shared by the wire format decoders.
// Pattern: Response-owned value state.
// Usage: The protocol reader owns one State and passes it to its selected decoder.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/responses/stream.go.
package native

import (
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type State struct {
	Calls                         Calls
	Usage                         *modelapi.TokenUsage
	Finished, Streaming, Received bool
	Failure                       error
	Continuation                  []byte
}

func (s *State) EmitFailedUsage(err error, emit func(modelapi.Event) error) error {
	if s.Usage != nil {
		return errors.Join(err, emit(modelapi.Event{Usage: s.Usage, Streaming: s.Streaming}))
	}
	return err
}
