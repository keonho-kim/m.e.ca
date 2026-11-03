// Purpose: Verify consumed blank prefixes and interleaved mailboxes render each reply below its real sender.
// Pattern: Virtual-terminal delivery integration test.
// Usage: go test ./internal/tui/session -run TestTeamDeliveredAttribution.
// Related: internal/tui/session/request_delivery.go, internal/tui/session/team_events.go.
package session

import (
	"context"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestTeamDeliveredAttribution(t *testing.T) {
	a := screenFixture(t)
	a.screen.SetSize(80, 40)
	a.team.enabled = true
	a.team.begin()
	delivery := requestDelivery{team: true}
	queue := make(chan notice, 16)
	for _, event := range []runrecord.Event{
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "stream", Agent: "Hana", Text: "A-public"},
		{Kind: "generation_started", Agent: "Ren"},
		{Kind: "stream", Agent: "Ren", Text: "\n\n"},
		{Kind: "stream", Agent: "Hana", Text: "A-public-complete"},
		{Kind: "stream", Agent: "Ren", Text: "\n\nB-public-complete"},
		{Kind: "team_progress", Agent: "Ren", Text: "B-public-complete"},
		{Kind: "team_progress", Agent: "Hana", Text: "A-public-complete", Final: true},
	} {
		var err error
		if event.Kind == "stream" {
			err = delivery.stream(context.Background(), queue, event)
		} else {
			err = delivery.post(context.Background(), queue, notice{Event: &event})
		}
		if err != nil {
			t.Fatal(err)
		}
		for len(queue) > 0 {
			a.receive(<-queue)
		}
		a.render()
	}
	var shown strings.Builder
	for y := range 40 {
		shown.WriteString(screenLine(a.screen, y, 80))
		shown.WriteByte('\n')
	}
	text := shown.String()
	previous := -1
	for _, marker := range []string{"Hana", "A-public-complete", "Ren", "B-public-complete"} {
		at := strings.Index(text, marker)
		if at <= previous || strings.Count(text, marker) != 1 {
			t.Fatalf("missing or crossed sender %q: %s", marker, text)
		}
		previous = at
	}
}
