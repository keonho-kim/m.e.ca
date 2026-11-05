// Purpose: Preserve folded tool sources and distinguish waiting from completed results.
// Pattern: Deterministic presentation lifecycle regressions and allocation benchmark.
// Usage: go test ./internal/tui/chat/team -run 'TestCommandPolling|TestWaiting' -bench BenchmarkTeamCommandPolling -benchmem.
// Related: internal/tui/chat/team/tools.go, internal/tui/transcript/tools.go.
package team

import (
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestCommandPollingPreservesFoldedSource(t *testing.T) {
	for _, collapsed := range []bool{true, false} {
		var state State
		output := viewport.View{Follow: true, Collapsed: collapsed}
		f := formatter()
		command := "python - <<'PY'\n" + strings.Repeat("print('working')\n", 30) + "# COMMAND_TAIL\nPY"
		arguments, _ := json.Marshal(map[string]string{"command": command})
		call := &toolapi.Call{ID: "call", Name: "execute_command", Arguments: arguments}
		ref := &runrecord.EventReference{Session: "private", Sequence: 1}
		state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
		state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call, Reference: ref,
			Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Status: "running"}},
		}, f, &output)
		id := state.tools[commandKey("Hana", "cmd_owned")].body
		if !collapsed {
			output.UpdateDetails(id, *ref, viewport.Block{Text: "captured output"})
		}
		before := output.Blocks[1]
		state.Commands("Hana", []toolapi.CommandRun{{CallID: call.ID, ID: "cmd_owned", Command: command, Status: "running", ElapsedMillis: 31000}}, f, &output)
		after := output.Blocks[1]
		if !output.HasDetails() || after.DetailRef != ref || after.Text != before.Text || after.Details != before.Details || after.Summary != before.Summary {
			t.Fatalf("collapsed=%t: command tick replaced result or source", collapsed)
		}
		if !strings.Contains(output.Blocks[0].Text, "31") {
			t.Fatal("command tick did not refresh elapsed time")
		}
	}
}

func TestWaitingToolHasNoResultSection(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	call := &toolapi.Call{ID: "waiting", Name: "write_file", Arguments: []byte(`{"path":"src/a.go","content":"code"}`)}
	state.Apply(runrecord.Event{Kind: "tool_waiting", Agent: "Hana", Call: call, Text: "Waiting for shared workspace access"}, f, &output)
	for _, block := range output.Blocks {
		if strings.Contains(block.Text, "    "+f.Language.Text("Output")+"\n") {
			t.Fatalf("waiting call was presented as output: %q", block.Text)
		}
	}
}

func TestCommandPollingRefreshesChangedCredentialMask(t *testing.T) {
	for _, reloadable := range []bool{false, true} {
		var state State
		output := viewport.View{Follow: true, Collapsed: false}
		vault := &credentials.Vault{}
		f := formatter()
		f.Redact, f.RedactionVersion = vault.Redact, vault.RedactionVersion
		call := &toolapi.Call{ID: "call", Name: "execute_command", Arguments: []byte(`{"command":"printf newly-secret"}`)}
		state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
		if reloadable {
			ref := &runrecord.EventReference{Session: "private", Sequence: 1}
			state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call, Reference: ref,
				Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Status: "running"}},
			}, f, &output)
			output.UpdateDetails(state.tools[commandKey("Hana", "cmd_owned")].body, *ref, viewport.Block{Text: "output newly-secret"})
		}
		vault.Remember("newly-secret")
		state.Commands("Hana", []toolapi.CommandRun{{CallID: call.ID, ID: "cmd_owned", Command: "printf newly-secret", Status: "running"}}, f, &output)
		body := output.Blocks[1]
		if strings.Contains(body.Text+body.Details+body.Summary, "newly-secret") || !output.HasDetails() {
			t.Fatal("credential change exposed stale command text or lost its source")
		}
		if reloadable && (body.DetailRef == nil || body.Details != "") {
			t.Fatal("reloadable detail was not invalidated for the new credential revision")
		}
	}
}

func BenchmarkTeamCommandPolling(b *testing.B) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	f := formatter()
	f.RedactionVersion = func() int { return 0 }
	command := strings.Repeat("a build command\n", 4096)
	arguments, _ := json.Marshal(map[string]string{"command": command})
	call := &toolapi.Call{ID: "call", Name: "execute_command", Arguments: arguments}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}, f, &output)
	state.Apply(runrecord.Event{Kind: "tool_succeeded", Agent: "Hana", Call: call, Reference: &runrecord.EventReference{Session: "private", Sequence: 1},
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Status: "running"}},
	}, f, &output)
	runs := []toolapi.CommandRun{{CallID: call.ID, ID: "cmd_owned", Command: command, Status: "running"}}
	state.Commands("Hana", runs, f, &output)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		runs[0].ElapsedMillis += 1000
		state.Commands("Hana", runs, f, &output)
	}
}
