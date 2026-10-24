// Purpose: Verify live snapshots, final totals and a fixed uppercase token counter across screens.
// Pattern: Terminal workflow tests.
// Usage: go test ./internal/tui/session -run TestTokenCounter.
// Related: internal/tui/session/tokens.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	"strings"
	"testing"
)

func TestTokenCounterSnapshotsAndFixedPosition(t *testing.T) {
	a := fixture(t)
	for _, count := range []int64{2, 5, 5} {
		a.receive(notice{Event: &run.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Input: 20, Output: count}}})
	}
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 20 · 출력 5 · 초당 토큰 —") {
		t.Fatalf("live counter=%q", got)
	}
	a.receive(notice{Event: &run.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 12, Output: 3}}})
	a.receive(notice{Done: true})
	a.output.Append(strings.Repeat("history\n", 50), false)
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 12 · 출력 3 · 초당 토큰 —") {
		t.Fatalf("final counter=%q", got)
	}
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "text without usage"}})
	a.receive(notice{Done: true})
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 12 · 출력 3 · 초당 토큰 —") {
		t.Fatalf("unreported usage changed counter=%q", got)
	}
	a.openModel()
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.HasSuffix(got, "입력 12 · 출력 3 · 초당 토큰 —") {
		t.Fatalf("menu counter=%q", got)
	}
	a.receive(notice{Event: &run.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 8, Output: 4}}})
	a.screen.SetSize(32, 12)
	a.render()
	if got := screenLine(a.screen, 0, 32); !strings.HasSuffix(got, "입력 20 · 출력 7") {
		t.Fatalf("resized counter=%q", got)
	}
}
