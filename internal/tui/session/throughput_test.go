// Purpose: Verify real throughput appears after OUTPUT and resets for a new generation.
// Pattern: Terminal event projection tests.
// Usage: go test ./internal/tui/session -run TestThroughput.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	"strings"
	"testing"
)

func TestThroughputHeaderAndLifecycle(t *testing.T) {
	a := fixture(t)
	rate := 25.5
	a.receive(notice{Event: &run.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, TokensPerSecond: &rate}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasPrefix(got, "상태 대기") || !strings.HasSuffix(got, "입력 100 · 출력 51 · 초당 토큰 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &run.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 100, Output: 51}, TokensPerSecond: &rate}})
	a.receive(notice{Done: true})
	a.openModel()
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "초당 토큰 25.5") {
		t.Fatal(got)
	}
	a.receive(notice{Event: &run.Event{Kind: "generation_started"}})
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "no reported tokens"}})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 100 · 출력 51 · 초당 토큰 —") {
		t.Fatal(got)
	}
	a.screen.SetSize(24, 12)
	a.render()
	if got := screenLine(a.screen, 0, 24); !strings.HasSuffix(got, "입력 100 · 출력 51") {
		t.Fatal("narrow header lost token counts: " + got)
	}
}
