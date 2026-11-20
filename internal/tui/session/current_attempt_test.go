// Purpose: Verify the header displays the active attempt, first-output wait and honest rate availability.
// Pattern: Virtual terminal measurement projection.
// Usage: go test ./internal/tui/session -run TestCurrentAttemptHeader.
// Related: internal/tui/session/tokens.go, internal/tui/view/tokens.go.
package session

import (
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestCurrentAttemptHeader(t *testing.T) {
	a := screenFixture(t)
	now := time.Now()
	v := record.ModelActivity{Role: inference.Agent, Started: now.Add(-4 * time.Second)}
	check := func(want string) {
		a.receiveRuntime(&record.Event{Kind: "model_activity", Activity: &v})
		a.render()
		line := screenLine(a.screen, 0, 80)
		if !strings.Contains(line, want) {
			t.Fatalf("header=%q want=%q", line, want)
		}
	}
	check("첫 출력 대기 4")
	v.FirstOutput = now
	check("초당 토큰 —")
	v.Speed = record.OutputSpeed{Source: "received", Throughput: record.Throughput{Output: 100, Duration: time.Second}}
	check("수신 TPS 100.0")
	v.Speed = record.OutputSpeed{Source: "provider", Throughput: record.Throughput{Output: 50, Duration: time.Second}}
	check("생성 TPS 50.0")
}
