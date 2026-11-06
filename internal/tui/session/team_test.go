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
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "stream", Agent: "Hana", Text: "first", Thinking: "hidden reasoning"},
		{Kind: "generation_started", Agent: "Ren"},
		{Kind: "stream", Agent: "Ren", Text: "second"},
		{Kind: "stream", Agent: "Hana", Text: "first complete"},
		{Kind: "team_progress", Agent: "Ren", Text: "second"},
		{Kind: "team_progress", Agent: "Hana", Text: "first complete"},
		{Kind: "team_message", Agent: "Ren", Text: "Please keep the interface."},
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
	if !strings.Contains(value, "Ren") {
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
		for _, id := range []string{"Hana", "Ren"} {
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

func TestTeamFooterShowsActiveNamesInJoinOrder(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		a := screenFixture(t)
		a.language = language
		a.team.enabled = true
		a.team.begin()
		a.connection.selection.Model = "test-model"
		label := "Team"
		if language == locale.Korean {
			label = "팀"
		}
		if caption := a.teamCaption(); caption != label {
			t.Fatalf("empty team caption: %q", caption)
		}
		for _, step := range []struct{ name, event, names string }{
			{"Ren", "joined", "Ren"},
			{"Hana", "joined", "Ren Hana"},
			{"Ren", "joined", "Ren Hana"},
			{"Ren", "left", "Hana"},
			{"Ren", "left", "Hana"},
			{"Ren", "joined", "Hana Ren"},
		} {
			a.receiveRuntime(&runrecord.Event{Kind: "team_notice", Agent: step.name, Text: step.event})
			want := "test-model · " + label + " " + step.names
			if got := a.footerModel(); got != want {
				t.Fatalf("%s %s: footer %q, want %q", step.name, step.event, got, want)
			}
			a.render()
			_, height := a.screen.Size()
			if shown := screenLine(a.screen, height-a.footerRows, 80); !strings.Contains(shown, want) {
				t.Fatalf("footer did not render active names: %q", shown)
			}
		}
		// A delayed runtime event must not make a departed peer active again.
		a.receiveRuntime(&runrecord.Event{Kind: "generation_started", Agent: "Arjun"})
		if a.teamCaption() != label+" Hana Ren" {
			t.Fatal("unjoined peer appeared in the footer")
		}
		a.toggleTeam()
		if a.teamCaption() != label+" Hana Ren" {
			t.Fatal("deferred mode change hid the active request")
		}
		a.team.finish(&a.chat.output)
		if a.teamCaption() != "" {
			t.Fatal("finished team retained active names")
		}
		a.team.enabled = true
		if a.teamCaption() != label {
			t.Fatal("completed request retained previous peer names")
		}
	}
}
