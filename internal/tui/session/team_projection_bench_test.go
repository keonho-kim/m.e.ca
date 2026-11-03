// Purpose: Measure team event projection without terminal rendering or model I/O.
// Pattern: Fixed-size presentation workload benchmark.
// Usage: Compare BenchmarkTeamUsageProjection before and after responsibility changes.
// Related: internal/tui/session/team_events.go, internal/tui/session/team_state.go.
package session

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"testing"
	"time"
)

func BenchmarkTeamUsageProjection(b *testing.B) {
	for _, count := range []int{2, 8} {
		b.Run(fmt.Sprintf("peers%d", count), func(b *testing.B) {
			a := screenFixture(b)
			a.team.enabled = true
			a.team.begin()
			for i := range count {
				a.receiveTeam(&runrecord.Event{Kind: "team_notice", Agent: identity.Name(i), Text: "joined"})
			}
			measured := 4 * time.Second
			event := runrecord.Event{Kind: "usage_progress", Agent: "Hana", Usage: &modelapi.TokenUsage{Input: 1000, Output: 100}, MeasuredDuration: measured}
			b.ReportAllocs()
			for b.Loop() {
				a.receiveTeam(&event)
			}
			if a.chat.tokenCurrent.Output != 100 {
				b.Fatal("current usage lost")
			}
		})
	}
}
