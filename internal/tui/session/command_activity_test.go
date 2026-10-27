// Purpose: Verify parallel tool activity remains visible through ticks and sibling completion.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run 'TestParallelToolActivity|TestRunningCommandActivity'.
// Related: internal/tui/session/command_activity.go, internal/tui/session/runtime_events.go.
package session

import (
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type activityTools struct {
	Tools
	toolapi.CommandExecutor
	runs []toolapi.CommandRun
}

func (t *activityTools) CommandActivity() []toolapi.CommandRun { return t.runs }

func commandStart(id, command string) runrecord.Event {
	args, _ := json.Marshal(map[string]string{"command": command})
	return runrecord.Event{Kind: "tool_started", CallID: id, Call: &toolapi.Call{ID: id, Name: "execute_command", Arguments: args}}
}

func activityTranscript(a *app) string {
	layout := a.chatLayout(100)
	var text strings.Builder
	for _, line := range layout.Window(0, layout.Count()) {
		text.WriteString(line.Text + "\n")
	}
	return text.String()
}

func TestParallelToolActivityPreservesSiblings(t *testing.T) {
	a := fixture(t)
	first := commandStart("read", "cat docs/guide.md")
	second := commandStart("ruff", "ruff check src")
	third := commandStart("ty", "ty check src")
	for _, event := range []*runrecord.Event{&first, &second, &third} {
		a.receiveRuntime(event)
	}
	shown := activityTranscript(a)
	for _, command := range []string{"cat docs/guide.md", "ruff check src", "ty check src"} {
		if !strings.Contains(shown, command) {
			t.Fatalf("parallel command %q disappeared: %s", command, shown)
		}
	}
	first.Kind, first.Output = "tool_succeeded", []byte(`{"Stdout":"guide contents","Status":"exited","ExitCode":0}`)
	a.receiveRuntime(&first)
	shown = activityTranscript(a)
	for _, text := range []string{"guide contents", "ruff check src", "ty check src"} {
		if !strings.Contains(shown, text) {
			t.Fatalf("sibling completion removed %q: %s", text, shown)
		}
	}
	if len(a.output.Blocks) != 1 {
		t.Fatal("live activity was persisted as completed tool output")
	}
	// Ordered completion must release every transient tool row.
	a.finishWorker(notice{})
	if shown := activityTranscript(a); strings.Contains(shown, "ruff check src") || strings.Contains(shown, "ty check src") {
		t.Fatalf("finished worker retained pending commands: %s", shown)
	}
}

func TestRunningCommandActivityPreservesFullCommandAndNotices(t *testing.T) {
	a := fixture(t)
	command := "printf '%s' '" + strings.Repeat("long-command-", 20) + "' && cat docs/final-section.md"
	a.opts.Tools = &activityTools{Tools: a.opts.Tools, runs: []toolapi.CommandRun{{ID: "cmd_check", Command: command, Status: "running"}}}
	file := runrecord.Event{Kind: "tool_started", CallID: "write", Call: &toolapi.Call{ID: "write", Name: "write_file", Arguments: []byte(`{"path":"pending.txt","content":"source"}`)}}
	a.receiveRuntime(&file)
	a.status = "separate notice"
	a.updateCommandActivity()
	shown := activityTranscript(a)
	for _, text := range []string{"pending.txt", "cat docs/final-section.md", "separate notice"} {
		if !strings.Contains(strings.Join(strings.Fields(shown), ""), strings.Join(strings.Fields(text), "")) {
			t.Fatalf("command tick removed %q: %s", text, shown)
		}
	}
}

func TestRunningCommandActivityShowsFourCallsWithoutDuplicates(t *testing.T) {
	a := fixture(t)
	tools := &activityTools{Tools: a.opts.Tools}
	a.opts.Tools = tools
	for _, command := range []string{"cat README.md", "cat src/main.py", "ruff check src", "ty check src"} {
		event := commandStart(command, command)
		a.receiveRuntime(&event)
		tools.runs = append(tools.runs, toolapi.CommandRun{CallID: command, ID: "cmd_" + command, Command: command, Status: "running"})
	}
	for _, tick := range []bool{false, true} {
		if tick {
			a.updateCommandActivity()
		}
		a.render()
		shown := strings.Join(screenRows(a, 24-a.footerRows), "\n")
		for _, command := range tools.runs {
			if strings.Count(shown, command.Command) != 1 {
				t.Fatalf("tick=%v: command %q missing or repeated: %s", tick, command.Command, shown)
			}
		}
	}
}
