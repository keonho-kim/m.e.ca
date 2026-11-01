// Purpose: Own composer input, input history, command suggestions, reservations and steering drafts.
// Pattern: Event-loop feature composition.
// Usage: Input handlers capture request values; the task worker consumes them at safe boundaries.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go, internal/tui/session/tasks.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
	"github.com/keonho-kim/meca/internal/tui/input"
)

type requestState struct {
	editor         input.Editor
	inputs         []string
	historyIndex   int
	draft          string
	commandHints   []commands.Suggestion
	commandOpen    bool
	commandIndex   int
	steering       chan composer.Request
	steeringDrafts []string
	queued         queue.State
	fileSubmission *fileSubmission
}
