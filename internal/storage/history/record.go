// Purpose: Represent ordered persisted conversation records without SQL types.
// Pattern: Simple Module.
// Usage: History consumers and storage adapters exchange Record values.
// Related: internal/storage/sqlite/timeline.go.
package history

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// ErrNotFound distinguishes an absent visible record from a storage failure.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrNotFound = faults.Errorf(diagcatalog.ServerHistoryRecordWasNotFound)

type Record struct {
	Images    []media.Metadata
	Truncated bool
	Sequence  int64
	Kind      string
	Message   modelapi.Message
	Event     runrecord.Event
}
