// Purpose: Keep command cards aligned with control results and observed status transitions.
// Pattern: Deterministic interleaved lifecycle regressions.
// Usage: go test ./internal/tui/chat/team -run 'TestCommandControl|TestCommandStatusTransition'.
// Related: internal/tui/chat/team/tools.go, internal/tui/transcript/commands.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestCommandControlUpdatesOwnedCardAndReleasesTracking(t *testing.T) {
	for _, control := range []string{"wait_command", "cancel_command"} {
		for _, collapsed := range []bool{true, false} {
			t.Run(control+map[bool]string{true: "/collapsed", false: "/expanded"}[collapsed], func(t *testing.T) {
				var state State
				output := viewport.View{Follow: true, Collapsed: collapsed}
				f := formatter()
				call := &toolapi.Call{ID: "start", Name: "execute_command", Arguments: []byte(`{"command":"slow-build"}`)}
				state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
				state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call,
					Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "slow-build", Status: "running"}},
				}, f, &output)
				owned := state.tools[commandKey("Hana", "cmd_owned")].messageDisplay
				controlCall := &toolapi.Call{ID: "control", Name: control, Arguments: []byte(`{"command_id":"cmd_owned"}`)}
				state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: controlCall}, f, &output)
				status, label := "exited", "Command succeeded"
				if control == "cancel_command" {
					status, label = "interrupted", "Command interrupted"
				}
				state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: controlCall,
					Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "slow-build", Status: status}},
					Output:    []byte(`{"Stdout":"final output","ExitCode":0,"Status":"` + status + `"}`),
				}, f, &output)
				state.Commands("Hana", nil, f, &output)
				var shown strings.Builder
				for _, block := range output.Blocks {
					shown.WriteString(block.Text)
				}
				if len(state.tools) != 0 || strings.Contains(shown.String(), f.Language.Text("Command running")) {
					t.Fatalf("terminal control left active presentation: %d cards, %q", len(state.tools), shown.String())
				}
				if !strings.Contains(shown.String(), "slow-build") || !strings.Contains(shown.String(), f.Language.Text(label)) {
					t.Fatalf("terminal control lost its command or outcome: %q", shown.String())
				}
				if output.Blocks[1].ID != owned.body || !strings.Contains(output.Blocks[1].Text, f.Language.Text(control)) {
					t.Fatal("control did not update the owned card using its real result source")
				}
				if !collapsed && strings.Count(shown.String(), "final output") != 1 {
					t.Fatal("terminal output was duplicated or lost")
				}
			})
		}
	}
}

func TestCommandStatusTransitionKeepsSummaryAndSource(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	call := &toolapi.Call{ID: "start", Name: "execute_command", Arguments: []byte(`{"command":"slow-build"}`)}
	ref := &runrecord.EventReference{Session: "private", Sequence: 1}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
	state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call, Reference: ref,
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Status: "waiting"}},
	}, f, &output)
	zero := 0
	for _, step := range []struct{ status, label string }{
		{"running", "Command running"}, {"finalizing", "Finalizing command output"}, {"exited", "Command succeeded"},
	} {
		state.Commands("Hana", []toolapi.CommandRun{{ID: "cmd_owned", CallID: "start", Command: "slow-build", Status: step.status, ExitCode: &zero}}, f, &output)
		if !strings.Contains(output.Blocks[0].Text, f.Language.Text(step.label)) || !strings.Contains(output.Blocks[1].Text, f.Language.Text(step.label)) {
			t.Fatalf("%s title and summary disagree: %q / %q", step.status, output.Blocks[0].Text, output.Blocks[1].Text)
		}
		if output.Blocks[1].DetailRef != ref || !output.HasDetails() {
			t.Fatal("status transition discarded the stored result reference")
		}
	}
}

func TestCommandControlKeepsNewestSourceAndOtherPeers(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	start := &toolapi.Call{ID: "same", Name: "execute_command", Arguments: []byte(`{"command":"build"}`)}
	for _, agent := range []string{"Hana", "Ren"} {
		state.Apply(runrecord.Event{Kind: "tool_started", Agent: agent, Call: start}, f, &output)
		state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: agent, Call: start,
			Reference: &runrecord.EventReference{Session: agent, Sequence: 1},
			Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "build", Status: "running"}},
		}, f, &output)
	}
	control := &toolapi.Call{ID: "control", Name: "wait_command", Arguments: []byte(`{"command_id":"cmd_owned"}`)}
	ref := &runrecord.EventReference{Session: "Hana", Sequence: 2}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: control}, f, &output)
	state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: control, Reference: ref,
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "build", Status: "running"}},
	}, f, &output)
	if len(state.tools) != 2 || state.tools[commandKey("Ren", "cmd_owned")].body == 0 {
		t.Fatal("control changed another peer or retained its provisional identity")
	}
	owned := state.tools[commandKey("Hana", "cmd_owned")]
	if output.Blocks[1].ID != owned.body || output.Blocks[1].DetailRef != ref {
		t.Fatal("control did not retain the newest expandable original")
	}
	zero := 0
	state.Commands("Hana", []toolapi.CommandRun{{ID: "cmd_owned", Status: "exited", Command: "build", ExitCode: &zero}}, f, &output)
	before := output.Blocks[1]
	state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: control,
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "build", Status: "running"}},
	}, f, &output)
	if output.Blocks[1].Text != before.Text || output.Blocks[1].DetailRef != ref {
		t.Fatal("delayed active result overwrote the observed terminal status or source")
	}
}
