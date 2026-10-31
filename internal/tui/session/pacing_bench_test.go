// Purpose: Measure input echo and drawing work under fast and deliberately slow terminal output.
// Pattern: Virtual-terminal workload benchmark with acknowledged input probes.
// Usage: Run BenchmarkTerminalPacing with -benchtime=1x; the slow case models a blocking terminal.
// Related: internal/tui/session/event_loop.go, internal/tui/session/app_test.go, internal/tui/session/delivery.go.
package session

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type echoProbe struct {
	length int
	sent   time.Time
	result chan time.Duration
}

type pacingScreen struct {
	tcell.Screen
	app     *app
	delay   time.Duration
	probes  chan echoProbe
	pending *echoProbe
	frames  int
	drawing time.Duration
}

func (s *pacingScreen) Show() {
	started := time.Now()
	if s.delay > 0 {
		// This is the measured workload, not a synchronization or pass condition.
		time.Sleep(s.delay)
	}
	s.Screen.Show()
	s.frames++
	s.drawing += time.Since(started)
	if s.pending == nil {
		select {
		case probe := <-s.probes:
			s.pending = &probe
		default:
		}
	}
	if s.pending != nil && len(s.app.editor.String()) >= s.pending.length {
		s.pending.result <- time.Since(s.pending.sent)
		s.pending = nil
	}
}

func BenchmarkTerminalPacing(b *testing.B) {
	for _, delay := range []time.Duration{0, 12 * time.Millisecond} {
		for _, typing := range []bool{false, true} {
			b.Run(fmt.Sprintf("renderDelay%s/typing%v", delay, typing), func(b *testing.B) {
				for b.Loop() {
					measurePacing(b, delay, typing)
				}
			})
		}
	}
}

func measurePacing(b *testing.B, delay time.Duration, typing bool) {
	a := fixture(b)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	a.ctx = ctx
	screen := &pacingScreen{Screen: a.screen, app: a, delay: delay, probes: make(chan echoProbe, 1)}
	a.screen = screen
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		delivery := runtimeDelivery{}
		step := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				step++
				if delivery.stream(ctx, a.events, runrecord.Event{Kind: "stream", Text: []string{"Working 0", "Working 1"}[step%2]}) != nil {
					return
				}
			}
		}
	}()
	measurements := make(chan []time.Duration, 1)
	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		var latencies []time.Duration
		intervals := [...]time.Duration{31, 43, 57, 37, 47, 61, 29, 53}
		for i := range 8 {
			interval := time.NewTimer(intervals[i] * time.Millisecond)
			select {
			case <-ctx.Done():
				interval.Stop()
				measurements <- nil
				return
			case <-interval.C:
			}
			if !typing {
				continue
			}
			probe := echoProbe{length: i + 1, sent: time.Now(), result: make(chan time.Duration, 1)}
			screen.probes <- probe
			select {
			case screen.EventQ() <- tcell.NewEventKey(tcell.KeyRune, "x", 0):
			case <-ctx.Done():
				measurements <- nil
				return
			}
			select {
			case latency := <-probe.result:
				latencies = append(latencies, latency)
			case <-ctx.Done():
				measurements <- nil
				return
			}
		}
		measurements <- latencies
		select {
		case screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0):
		case <-ctx.Done():
		}
	}()
	started := time.Now()
	if err := a.loop(); err != nil {
		b.Fatal(err)
	}
	elapsed := time.Since(started)
	cancel()
	<-streamDone
	<-inputDone
	latencies := <-measurements
	if typing && (len(latencies) != 8 || a.editor.String() != strings.Repeat("x", 8)) {
		b.Fatal("input probe did not complete")
	}
	b.ReportMetric(float64(screen.frames)/elapsed.Seconds(), "frames/s")
	b.ReportMetric(screen.drawing.Seconds()/elapsed.Seconds()*100, "drawing-percent")
	if len(latencies) > 0 {
		var total time.Duration
		for _, latency := range latencies {
			total += latency
		}
		b.ReportMetric(float64(total.Microseconds())/float64(len(latencies)), "echo-mean-us")
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		b.ReportMetric(float64(latencies[len(latencies)-1].Microseconds()), "echo-p95-us")
	}
}
