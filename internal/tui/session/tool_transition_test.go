// Purpose: Prevent automatic tool folding from flashing command bodies or moving the reading position.
// Pattern: Virtual terminal lifecycle regression tests.
// Usage: go test ./internal/tui/session -run 'TestToolTransition|TestLiveTool'.
// Related: internal/tui/session/tool_activity.go, internal/tui/session/keys.go, internal/tui/session/chat_projection.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestToolTransitionKeepsCollapsedHeight(t *testing.T) {
	a := screenFixture(t)
	a.chat.output.Append(strings.Repeat("prior response\n", 30), false)
	old := commandStart("old", "echo OLD_TOOL_RECORD")
	old.Kind, old.Output = "tool_succeeded", []byte(`{"Stdout":"old output","Status":"exited","ExitCode":0}`)
	a.receiveRuntime(&old)
	event := commandStart("fast", "cat <<'EOF'\n"+strings.Repeat("LONG_COMMAND_BODY\n", 25)+"PENDING_TAIL\nEOF")
	a.receiveRuntime(&event)
	a.render()
	before := a.chat.chatProjection.layout.Count()
	first := a.chat.chatVisible[0]
	if shown := strings.Join(screenRows(a, 22), "\n"); strings.Contains(shown, "PENDING_TAIL") {
		t.Fatal("collapsed start flashed the full command")
	}
	event.Kind, event.Output = "tool_succeeded", []byte(`{"Stdout":"ACTUAL_OUTPUT","Status":"exited","ExitCode":0}`)
	a.receiveRuntime(&event)
	a.render()
	if after := a.chat.chatProjection.layout.Count(); after != before {
		t.Fatalf("automatic completion changed transcript height: %d -> %d", before, after)
	}
	if got := a.chat.chatVisible[0]; got.ID != first.ID || got.Row != first.Row {
		t.Fatal("completion shifted the top visible history row")
	}
	if shown := strings.Join(screenRows(a, 22), "\n"); strings.Contains(shown, "ACTUAL_OUTPUT") || strings.Contains(shown, "PENDING_TAIL") {
		t.Fatal("completion bypassed the fold preference")
	}
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Text: "next response"})
	if len(a.chat.output.Blocks) != 3 || !strings.Contains(a.chat.output.Blocks[2].Summary, "LONG_COMMAND_BODY") {
		t.Fatal("next generation lost the completed tool record")
	}
}

func TestLiveToolDetailsToggleBeforeAnyResult(t *testing.T) {
	a := screenFixture(t)
	event := commandStart("running", "cat <<'EOF'\n"+strings.Repeat("command body\n", 25)+"COMMAND_TAIL\nEOF")
	a.receiveRuntime(&event)
	a.render()
	if keymapLabels(t, a)["^O"] != "도구 펴기" || strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("pending tool lacks a compact display and an expand action")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if a.chat.output.Collapsed || !strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("Ctrl+O could not expand a pending tool before its first result")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if !a.chat.output.Collapsed || strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("pending tool retained stale expanded text after folding")
	}
	event.Kind, event.Output = "tool_succeeded", []byte(`{"Stdout":"result","Status":"exited","ExitCode":0}`)
	a.receiveRuntime(&event)
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if shown := activityTranscript(a); !strings.Contains(shown, "COMMAND_TAIL") || !strings.Contains(shown, "result") {
		t.Fatal("completion lost the command or original output")
	}
}

func TestLiveToolMaskChangesInvalidateCachedText(t *testing.T) {
	a := screenFixture(t)
	event := commandStart("pending", "echo newly-secret")
	a.receiveRuntime(&event)
	if !strings.Contains(activityTranscript(a), "newly-secret") {
		t.Fatal("initial public argument missing")
	}
	a.opts.Vault.Remember("newly-secret")
	if shown := activityTranscript(a); strings.Contains(shown, "newly-secret") || !strings.Contains(shown, "[redacted]") {
		t.Fatal("pending display reused text from an older credential revision")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if strings.Contains(activityTranscript(a), "newly-secret") {
		t.Fatal("expansion exposed a stale pending detail")
	}
}

func TestLiveToolMaskChangesRebuildTruncatedSummary(t *testing.T) {
	a := screenFixture(t)
	secret := "new-key-" + strings.Repeat("secret", 40)
	event := commandStart("pending", "echo "+secret)
	a.receiveRuntime(&event)
	if !strings.Contains(activityTranscript(a), secret[:60]) {
		t.Fatal("initial public command was not summarized")
	}
	if err := a.opts.Vault.Remember(secret); err != nil {
		t.Fatal(err)
	}
	if shown := activityTranscript(a); strings.Contains(shown, secret[:60]) || !strings.Contains(shown, "[redacted]") {
		t.Fatal("truncated summary retained a fragment of a newly private value")
	}
}

func TestLiveToolCommandSnapshotsKeepFoldingAndReadingAnchor(t *testing.T) {
	a := screenFixture(t)
	a.chat.output.Append(strings.Repeat("reading history\n", 30), false)
	event := commandStart("pending", "cat <<'EOF'\n"+strings.Repeat("command body\n", 25)+"COMMAND_TAIL\nEOF")
	a.receiveRuntime(&event)
	tools := &activityTools{runs: []toolapi.CommandRun{{CallID: "pending", ID: "cmd_owned", Command: "cat <<'EOF'\n" + strings.Repeat("command body\n", 25) + "COMMAND_TAIL\nEOF", Status: "running"}}}
	a.opts.Tools = tools
	a.updateCommandActivity()
	a.render()
	if strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("command snapshot reopened a folded pending tool")
	}
	layout := a.chatLayout(74)
	layout.Seek(&a.chat.output, 4, 18)
	anchorID, anchorRow := a.chat.output.AnchorID, a.chat.output.AnchorRow
	event.Kind, event.Output = "tool_succeeded", []byte(`{"Stdout":"done","Status":"exited","ExitCode":0}`)
	tools.runs = nil
	a.receiveRuntime(&event)
	a.render()
	if a.chat.output.Follow || a.chat.output.AnchorID != anchorID || a.chat.output.AnchorRow != anchorRow {
		t.Fatal("tool completion moved a user reading older history")
	}
}

func TestLiveToolCommandSnapshotTogglesWithoutClockTick(t *testing.T) {
	a := screenFixture(t)
	command := "echo newly-secret\n" + strings.Repeat("command body\n", 25) + "COMMAND_TAIL"
	tools := &activityTools{runs: []toolapi.CommandRun{{ID: "cmd_owned", Command: command, Status: "running"}}}
	a.opts.Tools = tools
	a.updateCommandActivity()
	if strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("command-only snapshot ignored initial folding")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if !strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("snapshot cache waited for a clock tick before expanding")
	}
	a.opts.Vault.Remember("newly-secret")
	if shown := activityTranscript(a); strings.Contains(shown, "newly-secret") || !strings.Contains(shown, "[redacted]") {
		t.Fatal("command-only snapshot kept an outdated mask")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
	if strings.Contains(activityTranscript(a), "COMMAND_TAIL") {
		t.Fatal("snapshot cache waited for a clock tick before folding")
	}
	tools.runs = nil
	a.updateCommandActivity()
	if shown := activityTranscript(a); shown != "" || len(a.chat.toolActivity.views) != 0 || keymapLabels(t, a)["^O"] != "" {
		t.Fatal("completed command retained its live display or shortcut")
	}
}

func TestLiveToolCommandControlsDoNotAddTransientDuplicates(t *testing.T) {
	for _, control := range []string{"wait_command", "cancel_command"} {
		t.Run(control, func(t *testing.T) {
			a := screenFixture(t)
			a.opts.Tools = &activityTools{runs: []toolapi.CommandRun{{ID: "cmd_owned", Command: "slow-build", Status: "running"}}}
			a.updateCommandActivity()
			before := activityTranscript(a)
			event := runrecord.Event{Kind: "tool_started", CallID: "control", Call: &toolapi.Call{ID: "control", Name: control, Arguments: []byte(`{"command_id":"cmd_owned"}`)}}
			a.receiveRuntime(&event)
			if shown := activityTranscript(a); shown != before {
				t.Fatal("command control introduced a transient duplicate beside its owned command")
			}
			event.Kind, event.Text = "tool_failed", "control failed"
			a.receiveRuntime(&event)
			if shown := activityTranscript(a); !strings.Contains(shown, "control failed") || !strings.Contains(shown, "slow-build") {
				t.Fatal("control failure lost the command or the durable failure result")
			}
		})
	}
}
