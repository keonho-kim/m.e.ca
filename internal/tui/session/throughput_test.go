// Purpose: Verify current call timing and token totals follow their independent lifetimes.
// Pattern: Terminal event projection tests.
// Usage: go test ./internal/tui/session -run TestThroughput.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func testAttempt(out int64, duration time.Duration, finished bool) *runrecord.ModelActivity {
	now := time.Now()
	return &runrecord.ModelActivity{Role: inference.Agent, Started: now.Add(-duration), FirstOutput: now.Add(-duration), Finished: finished, Speed: runrecord.OutputSpeed{Source: "received", Throughput: runrecord.Throughput{Output: out, Duration: duration}}}
}
func TestThroughputHeaderAndLifecycle(t *testing.T) {
	a := screenFixture(t)
	measured := 2 * time.Second
	a.receive(notice{Event: &runrecord.Event{Kind: "usage_progress", Activity: testAttempt(51, measured, false), Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, MeasuredDuration: measured}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasPrefix(got, "상태 대기") || !strings.HasSuffix(got, "입력 100 · 출력 51 · 수신 TPS 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &runrecord.Event{Kind: "usage", Activity: testAttempt(51, measured, true), Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, MeasuredDuration: measured}})
	a.receive(notice{Done: true})
	a.openModel()
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "수신 TPS 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &runrecord.Event{Kind: "generation_started"}})
	a.receive(notice{Event: &runrecord.Event{Kind: "stream", Text: "no reported tokens"}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 100 · 출력 51") {
		t.Fatal(got)
	}
	a.screen.SetSize(24, 12)
	a.render()
	if got := screenLine(a.screen, 0, 24); !strings.HasSuffix(got, "입력 100 · 출력 51") {
		t.Fatal("narrow header lost token counts: " + got)
	}
}

func TestThroughputHiddenUntilFirstMeasurement(t *testing.T) {
	a := screenFixture(t)
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started"})
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Text: "Receiving text without reported usage"})
	a.render()
	if line := screenLine(a.screen, 0, 80); strings.Contains(line, "25.5") {
		t.Fatal("stale TPS on next generation", line)
	}
}

func TestThroughputFinalOnlyReport(t *testing.T) {
	a := screenFixture(t)
	first := testAttempt(20, time.Second, true)
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 20}, Activity: first})
	unknown := testAttempt(0, 0, true)
	unknown.Speed = runrecord.OutputSpeed{}
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 100}, Activity: unknown})
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.HasSuffix(line, "출력 120 · 초당 토큰 —") {
		t.Fatal(line)
	}
}
