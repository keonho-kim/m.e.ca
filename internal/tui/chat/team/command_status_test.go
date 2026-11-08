// Purpose: Show completed sibling commands before earlier native results become durable.
// Pattern: Deterministic interleaved team presentation test.
// Usage: go test ./internal/tui/chat/team -run TestCompletedCommandStatus.
// Related: internal/tui/chat/team/tools.go, internal/tui/transcript/command_view.go.
package team

import (
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestCompletedCommandStatusDoesNotWaitForEarlierReply(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	f := formatter()
	for _, id := range []string{"first", "second"} {
		call := &toolapi.Call{ID: id, Name: "execute_command", Arguments: []byte(`{"command":"sleep 60"}`)}
		state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
	}
	zero := 0
	state.Commands("Hana", []toolapi.CommandRun{
		{CallID: "first", ID: "cmd_first", Command: "sleep 60", Status: "running"},
		{CallID: "second", ID: "cmd_second", Command: "printf done", Status: "exited", ExitCode: &zero},
	}, f, &output)
	var text strings.Builder
	for _, block := range output.Blocks {
		text.WriteString(block.Text)
	}
	if !strings.Contains(text.String(), f.Language.Text("Command succeeded")) || !strings.Contains(text.String(), f.Language.Text("Command running")) {
		t.Fatal("sibling completion was hidden by earlier work", text.String())
	}
}

func TestActiveCommandsKeepSeparateCardsWhenCallIDsAreReused(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	f := formatter()
	for _, part := range []string{"old", "new"} {
		id := "cmd_" + part
		arguments, _ := json.Marshal(map[string]string{"command": part + "-command"})
		call := &toolapi.Call{ID: "reused", Name: "execute_command", Arguments: arguments}
		state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
		state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call, Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: id, Status: "running"}}}, f, &output)
	}
	state.Commands("Hana", []toolapi.CommandRun{{CallID: "reused", ID: "cmd_old", Command: "old-command", Status: "running"}, {CallID: "reused", ID: "cmd_new", Command: "new-command", Status: "running"}}, f, &output)
	var text strings.Builder
	for _, block := range output.Blocks {
		text.WriteString(block.Text)
	}
	if !strings.Contains(text.String(), "old-command") || !strings.Contains(text.String(), "new-command") {
		t.Fatal("native call ID reuse merged independent command cards", text.String())
	}
}
