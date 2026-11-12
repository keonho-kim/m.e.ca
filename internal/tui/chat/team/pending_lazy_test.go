// Purpose: Preserve lazy in-flight team tool details through explicit fold transitions.
// Pattern: Pure team presentation lifecycle test.
// Usage: go test ./internal/tui/chat/team -run TestPendingTeamTool.
// Related: internal/tui/chat/team/tools.go, internal/tui/transcript/tools.go.
package team

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestPendingTeamToolDoesNotRetainCollapsedDetails(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	state.Apply(record.Event{Kind: "tool_started", Agent: "Hana", Call: &tool.Call{ID: "call", Name: "execute_command", Arguments: []byte(`{"command":"first\nTEAM_TAIL"}`)}}, formatter(), &output)
	if len(output.Blocks) != 2 || output.Blocks[1].Details != "" {
		t.Fatal("collapsed team tool retained formatted details")
	}
	output.ToggleDetails()
	state.RefreshPending(formatter(), &output)
	if !strings.Contains(output.Blocks[1].Details, "TEAM_TAIL") {
		t.Fatal("team expansion lost the original")
	}
	output.ToggleDetails()
	state.RefreshPending(formatter(), &output)
	if output.Blocks[1].Details != "" {
		t.Fatal("team collapse retained the body")
	}
}
