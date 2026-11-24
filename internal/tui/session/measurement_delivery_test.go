// Purpose: Preserve the newest model timing and token counts under interleaved status delivery.
// Pattern: Deterministic mailbox-to-screen regression tests.
// Usage: go test -race ./internal/tui/session -run TestMeasurementDelivery.
// Related: internal/tui/session/delivery.go, internal/tui/session/tokens.go.
package session

import (
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestMeasurementDeliveryPreservesFirstOutput(t *testing.T) {
	for _, team := range []bool{false, true} {
		a := screenFixture(t)
		delivery := requestDelivery{team: team}
		agent := ""
		if team {
			agent = "Ren"
		}
		started := time.Now()
		waiting := record.ModelActivity{Role: inference.Agent, Started: started}
		producing := waiting
		producing.FirstOutput = started.Add(time.Second)
		for _, event := range []record.Event{
			{Kind: "model_activity", Agent: agent, Activity: &waiting},
			{Kind: "usage_progress", Agent: agent, Activity: &waiting, Usage: &modelapi.TokenUsage{Input: 100}},
			{Kind: "model_activity", Agent: agent, Activity: &producing},
		} {
			if err := delivery.emit(a.ctx, a.ctx, a.events, event); err != nil {
				t.Fatal(err)
			}
		}
		for len(a.events) > 0 {
			a.receive(<-a.events)
		}
		_, activity := a.chat.currentAttempt()
		if activity == nil || activity.FirstOutput != producing.FirstOutput || a.chat.displayTokens().Input != 100 {
			t.Fatalf("team=%t: timing regressed or usage lost: activity=%+v usage=%+v", team, activity, a.chat.displayTokens())
		}
	}
}
