// Purpose: Own request admission, pending reservations and undelivered steering drafts together.
// Pattern: Event-loop feature composition.
// Usage: Input handlers capture request values; the task worker consumes them at safe boundaries.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go, internal/tui/session/tasks.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
)

type requestState struct {
	steering       chan composer.Request
	steeringDrafts []string
	queued         queue.State
	fileSubmission *fileSubmission
}
