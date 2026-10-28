// Purpose: Keep captured paths attached while requests are edited, shifted and restored to a draft.
// Pattern: Queue ownership contract tests.
// Usage: go test ./internal/tui/composer/queue.
// Related: internal/tui/composer/queue/state.go, internal/tui/composer/request.go.
package queue

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func TestQueuePreservesReferenceOwnership(t *testing.T) {
	state := State{Messages: []composer.Request{composer.Capture("review", []string{"a.txt"})}}
	var editor input.Editor
	state.Navigate(-1, &editor)
	if !state.Save("revised") || state.Messages[0].Text != "revised" || state.Messages[0].Paths[0] != "a.txt" {
		t.Fatal("editing changed the captured references")
	}
	state.RestoreEditor(&editor)
	state.Drain(&editor)
	if !strings.Contains(editor.String(), "revised") || !strings.Contains(editor.String(), "a.txt") || state.Messages != nil {
		t.Fatal("draft recovery lost references or retained queue memory")
	}
	state.Messages = []composer.Request{composer.Capture("another", []string{"b.txt"})}
	retained := state.Messages
	state.Shift()
	if retained[0].Text != "" || retained[0].Paths != nil {
		t.Fatal("dequeue retained its request snapshot")
	}
}
