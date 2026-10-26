// Purpose: Represent ordered persisted conversation records without SQL types.
// Pattern: Simple Module.
// Usage: History consumers and storage adapters exchange Record values.
// Related: internal/storage/sqlite/timeline.go.
package history

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type Record struct {
	Sequence int64
	Kind     string
	Message  modelapi.Message
	Event    runrecord.Event
}
