// Purpose: Join a pending LAT-memory call and commit its last measured usage on shutdown.
// Pattern: Channel-controlled runtime lifecycle test.
// Usage: go test -race ./internal/agent/run -run TestLATMemoryShutdown.
// Related: internal/agent/run/lat_memory.go, internal/agent/run/lat_memory_usage.go.
package run

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestLATMemoryShutdownJoinsAndRetainsActualUsage(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	j := &journal{}
	if err := j.AppendMessage(ctx, "s", modelapi.Message{Role: "user", Content: strings.Repeat("required constraint ", 4000)}); err != nil {
		t.Fatal(err)
	}
	r := fixture(&generator{}, &executor{}, j)
	r.LAT = true
	r.ReviewMemory = &journalMemory{journal: j}
	ready, joined := make(chan struct{}), make(chan struct{})
	g := statisticsModel(func(ctx context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if err := send(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 20, Output: 3}}); err != nil {
			return err
		}
		close(ready)
		<-ctx.Done()
		close(joined)
		return ctx.Err()
	})
	s := execution{runner: r, reviewGenerator: g, latObserved: true, limits: DefaultLimits(), usage: &requestBudget{}, statistics: runStatistics{started: time.Now()}, emit: emit, latMemory: &reviewMemory{ctx: ctx, stop: stop, dirty: true, controller: latmemory.Controller{Repository: r.ReviewMemory, Session: "s", Redact: r.Redact}}}
	s.usage.begin(s.limits)
	if err := s.maintainLATMemory(ctx, false); err != nil {
		t.Fatal(err)
	}
	<-ready
	if err := s.closeLATMemory(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-joined:
	default:
		t.Fatal("summary worker outlived request shutdown")
	}
	if s.activeLATMemory() || !s.latMemory.closed {
		t.Fatal("closed worker was retained")
	}
	statistics := s.statistics.snapshot(time.Now())
	if statistics.Usage == nil || statistics.Usage.Input != 20 || statistics.Usage.Output != 3 {
		t.Fatal("cancel lost received usage", statistics)
	}
	found := false
	for _, event := range j.events {
		if event.Generation != nil && event.Generation.Role == inference.LATCompaction {
			found = true
		}
	}
	if !found {
		t.Fatal("cancelled background attempt was not persisted")
	}
	if err := s.maintainLATMemory(ctx, true); err != nil || s.activeLATMemory() {
		t.Fatal("shutdown restarted a worker", err)
	}
}
