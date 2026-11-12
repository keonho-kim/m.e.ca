// Purpose: Preserve oversized interrupted inputs as separate requests instead of overflowing the editor.
// Pattern: Pure recovery-boundary regression tests.
// Usage: go test ./internal/tui/composer/queue -run TestOversizedRecovery.
// Related: internal/tui/composer/queue/state.go, internal/tui/input/editor.go.
package queue

import (
	"strings"
	"testing"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func TestOversizedRecoveryKeepsRequestsAndDraft(t *testing.T) {
	body := strings.Repeat("x", 9<<20)
	state := State{Messages: []requestinput.Request{requestinput.Capture(body, []string{"first.txt"}), requestinput.Capture(body, nil)}}
	var editor input.Editor
	editor.Set("keep draft")
	state.Drain(&editor)
	if editor.String() != "keep draft" || len(state.Messages) != 2 || state.Messages[0].Paths[0] != "first.txt" {
		t.Fatal("oversized recovery merged or lost accepted input")
	}
}
