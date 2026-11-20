// Purpose: Verify team attempts keep their own rates across interleaving and peer departures.
// Pattern: Virtual-terminal event projection tests.
// Usage: go test ./internal/tui/session -run TestThroughputTeam.
// Related: internal/tui/chat/team/state.go, internal/tui/session/team_events.go, internal/tui/view/tokens.go.
package session

import (
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestThroughputTeamInterleavingAndDepartures(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	a.team.begin()
	for _, event := range []runrecord.Event{
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "generation_started", Agent: "Ren"},
		{Kind: "usage_progress", Agent: "Hana", Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: 2 * time.Second},
		{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 30}, MeasuredDuration: time.Second},
		{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 30}, MeasuredDuration: time.Second},
		{Kind: "usage", Agent: "Hana", Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: 2 * time.Second},
		{Kind: "team_notice", Agent: "Hana", Text: "left"},
		{Kind: "usage", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 30}, MeasuredDuration: time.Second},
		{Kind: "generation_started", Agent: "Ren"},
		{Kind: "usage", Agent: "Ren", Activity: testAttempt(100, 2*time.Second, true), Usage: &modelapi.TokenUsage{Output: 100}},
	} {
		a.receiveRuntime(&event)
	}
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.HasSuffix(line, "출력 150 · Ren 수신 TPS 50.0") {
		t.Fatalf("unmeasured tokens, repeat snapshots or departed peers changed TPS: %q", line)
	}
	a.team.finish(&a.chat.output)
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.HasSuffix(line, "Ren 수신 TPS 50.0") {
		t.Fatalf("team completion cleared the measured value: %q", line)
	}
}
