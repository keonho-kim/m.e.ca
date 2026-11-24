// Purpose: Verify and measure scoped team repainting through the notice delivery path.
// Pattern: Virtual terminal regression tests and fixed-workload benchmark.
// Usage: go test ./internal/tui/session -run TestTeamFrame -bench BenchmarkTeamStatusFrame.
// Related: internal/tui/session/events.go, internal/tui/session/render_header.go.
package session

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

type bodyPaintScreen struct {
	tcell.Screen
	clears int
}

func (s *bodyPaintScreen) Clear() { s.clears++; s.Screen.Clear() }

func paintNotice(a *app, n notice) {
	if a.receiveFrame(n) {
		a.render()
	} else {
		a.renderHeader()
	}
}

func TestTeamFrameStatusKeepsBodyAndFinalAnswerVisible(t *testing.T) {
	a := screenFixture(t)
	screen := &bodyPaintScreen{Screen: a.screen}
	a.screen = screen
	a.team.enabled = true
	a.team.begin()
	for _, name := range []string{"Hana", "Ren"} {
		event := record.Event{Kind: "team_notice", Agent: name, Text: "joined"}
		paintNotice(a, notice{Event: &event})
	}
	stream := record.Event{Kind: "stream", Agent: "Ren", Text: "남은 팀원의 응답"}
	paintNotice(a, notice{Event: &stream})
	before := screen.clears
	activity := record.ModelActivity{Role: inference.Agent, Started: time.Now().Add(-time.Second)}
	for _, kind := range []string{"model_activity", "usage_progress", "usage", "generation_finished"} {
		event := record.Event{Kind: kind, Agent: "Ren", Activity: &activity}
		if kind == "usage_progress" || kind == "usage" {
			event.Usage = &modelapi.TokenUsage{Output: 17}
		}
		paintNotice(a, notice{Event: &event})
	}
	if screen.clears != before || a.chat.displayTokens().Output != 17 {
		t.Fatalf("status repainted body or lost usage: clears=%d before=%d tokens=%+v", screen.clears, before, a.chat.displayTokens())
	}
	for _, event := range []record.Event{
		{Kind: "team_notice", Agent: "Hana", Text: "left"},
		{Kind: "stream", Agent: "Ren", Text: "최종 응답을 완료했습니다."},
		{Kind: "team_progress", Agent: "Ren", Text: "최종 응답을 완료했습니다.", Final: true},
		{Kind: "team_notice", Agent: "Ren", Text: "left"},
	} {
		paintNotice(a, notice{Event: &event})
	}
	paintNotice(a, notice{Done: true, RunStatus: record.Answered})
	var visible strings.Builder
	for y := range 24 {
		visible.WriteString(screenLine(a.screen, y, 80))
	}
	if !strings.Contains(visible.String(), "최종 응답을 완료했습니다.") || a.team.active {
		t.Fatalf("final answer or completion missing: %s", visible.String())
	}
}

func TestTeamFrameLoopKeepsPendingTextAndEscape(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := screenFixture(t)
		screen := &bodyPaintScreen{Screen: a.screen}
		a.screen = screen
		a.memory.read = func() (uint64, error) { return 42 << 20, nil }
		a.team.enabled = true
		a.team.begin()
		a.work.state = workerAgent
		canceled := false
		a.work.cancel = func() { canceled = true }
		done := make(chan error, 1)
		go func() { done <- a.loop() }()
		synctest.Wait()
		// Let the terminal's initial resize and its scheduled frame settle.
		time.Sleep(20 * time.Millisecond)
		synctest.Wait()
		before := screen.clears
		for range 4 {
			a.events <- notice{Event: &record.Event{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 9}}}
			synctest.Wait()
			time.Sleep(20 * time.Millisecond)
			synctest.Wait()
		}
		if screen.clears != before {
			t.Error("status-only notices rebuilt the body in the event loop")
		}
		for _, event := range []record.Event{
			{Kind: "team_notice", Agent: "Hana", Text: "left"},
			{Kind: "stream", Agent: "Ren", Text: "pending final text"},
			{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 10}},
		} {
			a.events <- notice{Event: &event}
		}
		synctest.Wait()
		time.Sleep(20 * time.Millisecond)
		synctest.Wait()
		var visible strings.Builder
		for y := range 24 {
			visible.WriteString(screenLine(a.screen, y, 80))
		}
		if !strings.Contains(visible.String(), "pending final text") {
			t.Error("status update dropped the pending final response frame")
		}
		a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyEscape, "", 0)
		synctest.Wait()
		if !canceled || !a.work.state.stopping() || a.exit {
			t.Error("Escape failed to cancel the team while preserving the session")
		}
		a.cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestTeamFrameUsageClearsVisibleRetry(t *testing.T) {
	a := screenFixture(t)
	retry := record.Event{Kind: "retry", Agent: "Ren", Text: "retry pending"}
	paintNotice(a, notice{Event: &retry})
	usage := record.Event{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 1}}
	paintNotice(a, notice{Event: &usage})
	for y := range 24 {
		if strings.Contains(screenLine(a.screen, y, 80), "retry pending") {
			t.Fatal("resumed output left the old retry notice visible")
		}
	}
}

func BenchmarkTeamStatusFrame(b *testing.B) {
	for _, scoped := range []bool{false, true} {
		name := "full"
		if scoped {
			name = "scoped"
		}
		b.Run(name, func(b *testing.B) {
			a := screenFixture(b)
			a.team.enabled = true
			a.team.begin()
			for _, agent := range []string{"Hana", "Ren"} {
				a.receiveRuntime(&record.Event{Kind: "team_notice", Agent: agent, Text: "joined"})
				a.receiveRuntime(&record.Event{Kind: "stream", Agent: agent, Text: strings.Repeat("- **확인 결과**를 공유합니다.\n", 200)})
			}
			a.render()
			event := record.Event{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Input: 1000, Output: 100}}
			b.ReportAllocs()
			for b.Loop() {
				if scoped {
					paintNotice(a, notice{Event: &event})
				} else {
					a.receive(notice{Event: &event})
					a.render()
				}
			}
		})
	}
}
