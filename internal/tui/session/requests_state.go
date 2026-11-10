// Purpose: Own composer input, input history, command suggestions, reservations and steering drafts.
// Pattern: Event-loop feature composition.
// Usage: Input handlers capture request values; the task worker consumes them at safe boundaries.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go, internal/tui/session/tasks.go.
package session

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/commands"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
	"github.com/keonho-kim/meca/internal/tui/input"
)

type requestState struct {
	images         []media.Image
	imageEpoch     uint64
	editor         input.Editor
	inputs         []string
	historyIndex   int
	draft          string
	commandHints   []commands.Suggestion
	commandOpen    bool
	commandIndex   int
	steering       chan requestinput.Request
	steeringReady  chan struct{}
	steeringDrafts []string
	queued         queue.State
	fileSubmission *fileSubmission
}
