// Purpose: Verify request-weighted throughput remains visible between measured generations.
// Pattern: Terminal event projection tests.
// Usage: go test ./internal/tui/session -run TestThroughput.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"testing"
	"time"
)

func TestThroughputHeaderAndLifecycle(t *testing.T) {
	a := screenFixture(t)
	measured := 2 * time.Second
	a.receive(notice{Event: &runrecord.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, MeasuredDuration: measured}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasPrefix(got, "상태 대기") || !strings.HasSuffix(got, "입력 100 · 출력 51 · 초당 토큰 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, MeasuredDuration: measured}})
	a.receive(notice{Done: true})
	a.openModel()
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "초당 토큰 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &runrecord.Event{Kind: "generation_started"}})
	a.receive(notice{Event: &runrecord.Event{Kind: "stream", Text: "no reported tokens"}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 100 · 출력 51 · 초당 토큰 25.5") {
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
	if line := screenLine(a.screen, 0, 80); strings.Contains(line, "초당 토큰") || !strings.HasSuffix(line, "입력 0 · 출력 0") {
		t.Fatalf("unmeasured TPS was shown: %q", line)
	}
}

func TestThroughputUsesWeightedRequestTotal(t *testing.T) {
	a := screenFixture(t)
	slow, fast := 2*time.Second, time.Second
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: slow})
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started"})
	for range 2 {
		a.receiveRuntime(&runrecord.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Output: 30}, MeasuredDuration: fast})
		a.render()
		if line := screenLine(a.screen, 0, 80); !strings.HasSuffix(line, "초당 토큰 16.7") {
			t.Fatalf("expected 50 tokens / 3 seconds, got %q", line)
		}
	}
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 30}, MeasuredDuration: fast})
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started"})
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 100}})
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.HasSuffix(line, "출력 150 · 초당 토큰 16.7") {
		t.Fatalf("unknown timing changed measured throughput or token totals: %q", line)
	}
}
