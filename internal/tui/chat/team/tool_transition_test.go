// Purpose: Keep team tool transitions under the same explicit fold preference as single-mode tools.
// Pattern: Deterministic presentation lifecycle regression.
// Usage: go test ./internal/tui/chat/team -run TestToolTransition.
// Related: internal/tui/chat/team/tools.go, internal/tui/transcript/tools.go, internal/tui/viewport/details.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestToolTransitionPreservesFoldPreferenceAndBodyIdentity(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	call := &toolapi.Call{ID: "tool", Name: "execute_command", Arguments: []byte(`{"command":"echo start\n` + strings.Repeat(`long command\n`, 30) + `COMMAND_TAIL"}`)}
	event := runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}
	state.Apply(event, f, &output)
	body := output.Blocks[1].ID
	if strings.Contains(output.Blocks[1].Text, "COMMAND_TAIL") {
		t.Fatal("team tool start exposed its full command")
	}
	output.ToggleDetails()
	state.RefreshPending(f, &output)
	if !strings.Contains(output.Blocks[1].Text, "COMMAND_TAIL") {
		t.Fatal("explicit expansion lost pending arguments")
	}
	event.Kind, event.Output = "tool_succeeded", []byte(`{"Stdout":"actual output","Status":"exited","ExitCode":0}`)
	state.Apply(event, f, &output)
	if len(output.Blocks) != 2 || output.Blocks[1].ID != body || !strings.Contains(output.Blocks[1].Text, "actual output") {
		t.Fatal("completion replaced its identity, duplicated the card or lost expansion")
	}
	output.ToggleDetails()
	if strings.Contains(output.Blocks[1].Text, "COMMAND_TAIL") || strings.Contains(output.Blocks[1].Text, "actual output") {
		t.Fatal("team completion ignored explicit folding")
	}
}

func TestToolTransitionCommandControlKeepsExistingCard(t *testing.T) {
	for _, control := range []string{"wait_command", "cancel_command"} {
		t.Run(control, func(t *testing.T) {
			var state State
			output := viewport.View{Follow: true, Collapsed: true}
			f := formatter()
			call := &toolapi.Call{ID: "start", Name: "execute_command", Arguments: []byte(`{"command":"slow-build"}`)}
			state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
			state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call,
				Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "slow-build", Status: "running"}},
			}, f, &output)
			before := output.Next
			controlCall := &toolapi.Call{ID: "control", Name: control, Arguments: []byte(`{"command_id":"cmd_owned"}`)}
			state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: controlCall}, f, &output)
			if output.Next != before {
				t.Fatal("command control appended a card that disappears on completion")
			}
			state.Apply(runrecord.Event{Kind: "tool_failed", Agent: "Hana", Call: controlCall, Text: "control failed"}, f, &output)
			if output.Next == before || !strings.Contains(output.Blocks[len(output.Blocks)-1].Text, "control failed") {
				t.Fatal("control failure without a process outcome was lost")
			}
		})
	}
}

func TestToolTransitionCommandControlDoesNotBorrowAnotherPeerCard(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	call := &toolapi.Call{ID: "start", Name: "execute_command", Arguments: []byte(`{"command":"slow-build"}`)}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
	state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call,
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "slow-build", Status: "running"}},
	}, f, &output)
	before := output.Next
	control := &toolapi.Call{ID: "control", Name: "wait_command", Arguments: []byte(`{"command_id":"cmd_owned"}`)}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Ren", Call: control}, f, &output)
	if output.Next != before+2 || !strings.Contains(output.Blocks[len(output.Blocks)-2].Text, "Ren") || !strings.Contains(output.Blocks[1].Text, "slow-build") {
		t.Fatal("unmatched control hid its sender or changed another peer's command")
	}
}
