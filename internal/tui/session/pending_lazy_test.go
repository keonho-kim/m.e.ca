// Purpose: Keep collapsed in-flight tools free of formatted detail bodies.
// Pattern: Virtual terminal display-lifecycle regression.
// Usage: go test ./internal/tui/session -run TestPendingToolDetails.
// Related: internal/tui/session/tool_activity.go, internal/tui/transcript/tools.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestPendingToolDetailsAreFormattedOnlyWhenExpanded(t *testing.T) {
	a := screenFixture(t)
	event := commandStart("pending", strings.Repeat("command line\n", 128)+"LAZY_TAIL")
	a.receiveRuntime(&event)
	if len(a.chat.toolActivity.pending) != 1 || a.chat.toolActivity.pending[0].details != "" {
		t.Fatal("collapsed tool retained a formatted detail body")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if !strings.Contains(activityTranscript(a), "LAZY_TAIL") {
		t.Fatal("expansion did not format the original")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	activityTranscript(a)
	if a.chat.toolActivity.pending[0].details != "" {
		t.Fatal("collapse retained formatted details")
	}
}
