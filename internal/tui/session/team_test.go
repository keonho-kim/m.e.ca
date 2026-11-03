// Purpose: Verify IRC attribution, independent streams, hidden reasoning and team-only mode controls.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestTeam.
// Related: internal/tui/session/team_events.go, internal/tui/session/team_state.go.
package session

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestTeamStreamsStayInTheirOwnMessageBlocks(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	a.team.begin()
	for _, event := range []runrecord.Event{
		{Kind: "generation_started", Agent: "agent-a"},
		{Kind: "stream", Agent: "agent-a", Text: "first", Thinking: "hidden reasoning"},
		{Kind: "generation_started", Agent: "agent-b"},
		{Kind: "stream", Agent: "agent-b", Text: "second"},
		{Kind: "stream", Agent: "agent-a", Text: "first complete"},
		{Kind: "team_progress", Agent: "agent-b", Text: "second"},
		{Kind: "team_progress", Agent: "agent-a", Text: "first complete"},
		{Kind: "team_message", Agent: "agent-b", Text: "Please keep the interface."},
	} {
		a.receiveRuntime(&event)
	}
	var text strings.Builder
	for _, block := range a.chat.output.Blocks {
		text.WriteString(block.Text)
	}
	value := text.String()
	if strings.Count(value, "first complete") != 1 || strings.Count(value, "second") != 1 || strings.Contains(value, "hidden reasoning") {
		t.Fatalf("mixed output: %s", value)
	}
	if !strings.Contains(value, "agent-b") {
		t.Fatal("public sender missing")
	}
	if strings.Contains(value, "#team") {
		t.Fatal("unnecessary broadcast recipient")
	}
	a.render()
}
func TestTeamCommandTogglesWithoutCallingModel(t *testing.T) {
	a := screenFixture(t)
	a.submitCommand(commands.Command{Kind: commands.Team})
	if !a.team.enabled || a.work.state.active() {
		t.Fatal("mode toggle started a model request")
	}
	a.submitCommand(commands.Command{Kind: commands.Team})
	if a.team.enabled {
		t.Fatal("mode did not turn off")
	}
}

func TestTeamModeChangeDuringWorkIsLocalizedAndDeferred(t *testing.T) {
	a := screenFixture(t)
	a.language = locale.English
	a.work.state = workerAgent
	a.team.active = true
	a.toggleTeam()
	if a.stopNotice != "Team mode ON · Applies to the next request" || !a.team.active {
		t.Fatalf("state changed during work or untranslated notice: %q", a.stopNotice)
	}
	a.team.finish(&a.chat.output)
	if a.team.view.Count() != 0 {
		t.Fatal("finished team retained display registries")
	}
}

func TestTeamCancellationFinishesEveryStream(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	for range 3 {
		a.team.begin()
		for _, id := range []string{"agent-a", "agent-b"} {
			a.receiveTeam(&runrecord.Event{Kind: "generation_started", Agent: id})
			a.receiveTeam(&runrecord.Event{Kind: "stream", Agent: id, Text: "**unfinished** contribution"})
		}
		a.render()
		a.team.finish(&a.chat.output)
		for _, block := range a.chat.output.Blocks {
			if block.Streaming {
				t.Fatal("cancelled stream remained active in later requests")
			}
		}
	}
}
