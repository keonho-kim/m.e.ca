// Purpose: Represent ordered persisted conversation records without SQL types.
// Pattern: Simple Module.
// Usage: History consumers and storage adapters exchange Record values.
// Related: internal/storage/sqlite/timeline.go.
package history

import (
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

// ErrNotFound distinguishes an absent visible record from a storage failure.
var ErrNotFound = errors.New("history record was not found")

type Record struct {
	Truncated bool
	Sequence  int64
	Kind      string
	Message   modelapi.Message
	Event     runrecord.Event
}
