// Purpose: Verify independent prefixes, coalesced work, cancellation and exact measured receipts.
// Pattern: Deterministic background-worker contract tests.
// Usage: go test -race ./internal/agent/lat/memory.
// Related: internal/agent/lat/memory/controller.go, internal/agent/lat/memory/worker.go.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type repository struct {
	state         State
	items         []Record
	reads, writes int
	fail          bool
}

func (r *repository) LATMemory(context.Context, string) (State, error) { return r.state, nil }
func (r *repository) InstallLATMemory(_ context.Context, _ string, expected int, next State) error {
	if r.fail || r.state.Revision != expected {
		return fmt.Errorf("publication rejected")
	}
	r.state = next
	r.writes++
	return nil
}
func (r *repository) LATWatermark(_ context.Context, _ string, after int64) (Mark, error) {
	mark := Mark{Through: after}
	for _, item := range r.items {
		if item.Sequence > after {
			mark.Through = item.Sequence
			mark.Bytes += int64(len(item.Text))
			if item.Kind == "user" {
				mark.LatestUser = item.Sequence
			}
		}
	}
	return mark, nil
}
func (r *repository) LATRecords(_ context.Context, _ string, after, through int64, budget int) ([]Record, int64, error) {
	r.reads++
	var out []Record
	for _, item := range r.items {
		if item.Sequence > after && item.Sequence <= through {
			out = append(out, item)
			if len(out) == 64 {
				return out, item.Sequence, nil
			}
		}
	}
	return out, through, nil
}

type generation func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (f generation) Generate(ctx context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
	return f(ctx, request, send)
}

func planned(t *testing.T, repo *repository) (*Controller, Plan) {
	t.Helper()
	c := &Controller{Repository: repo, Session: "s", Redact: func(text string) string { return text }}
	t.Cleanup(func() {
		if c.Active() {
			c.Cancel()
			<-c.Done()
			_, _ = c.Poll(context.Background())
		}
	})
	p, err := c.Prepare(context.Background(), prompt.Defaults().LATCompaction, 262144, 2048, true)
	if err != nil || p.Through == 0 {
		t.Fatal("plan", err)
	}
	return c, p
}

func TestBackgroundPrefixKeepsNewInstructionsAndOneSlot(t *testing.T) {
	r := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: strings.Repeat("first request ", 4000)}}}
	c, p := planned(t, r)
	started, release := make(chan struct{}), make(chan struct{})
	g := generation(func(ctx context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return ctx.Err()
		}
		return send(modelapi.Event{Delta: "- Initial request preserved.", Done: true})
	})
	if err := c.Start(context.Background(), p, g, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	<-started
	r.items = append(r.items, Record{Sequence: 2, Kind: "user", Text: "LATEST_CONSTRAINT"}, Record{Sequence: 3, Kind: "outcome", Text: "LATEST_RESULT"})
	reads := r.reads
	if pending, err := c.Prepare(context.Background(), prompt.Defaults().LATCompaction, 262144, 2048, true); err != nil || pending.Through != 0 || r.reads != reads {
		t.Fatal("active work was queued as duplicate snapshots", err)
	}
	if err := c.Start(context.Background(), p, g, func() bool { return true }); err == nil {
		t.Fatal("second worker was admitted")
	}
	close(release)
	<-c.Done()
	update, err := c.Poll(context.Background())
	if err != nil || !update.Done || c.Active() || r.state.Through != 1 || r.writes != 1 {
		t.Fatal("prefix publication", r.state, err)
	}
	body, err := c.Tail(context.Background(), 8192, "")
	if err != nil || !strings.Contains(string(body), "LATEST_CONSTRAINT") || !strings.Contains(string(body), "LATEST_RESULT") {
		t.Fatal("late records lost", err)
	}
	next, err := c.Prepare(context.Background(), prompt.Defaults().LATCompaction, 262144, 2048, true)
	if err != nil || next.Through != 3 {
		t.Fatal("coalesced work did not target the newest boundary", err)
	}
}

func TestCancelledMemoryPreservesPriorStateAndReceivedUsage(t *testing.T) {
	r := &repository{state: State{Revision: 1, Through: 1, Text: "prior memo"}, items: []Record{{Sequence: 2, Kind: "user", Text: "new requirement"}}}
	c, p := planned(t, r)
	ready := make(chan struct{})
	g := generation(func(ctx context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if err := send(modelapi.Event{Delta: "partial", Usage: &modelapi.TokenUsage{Input: 10, Output: 2}}); err != nil {
			return err
		}
		close(ready)
		<-ctx.Done()
		return ctx.Err()
	})
	if err := c.Start(context.Background(), p, g, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	<-ready
	progress, err := c.Poll(context.Background())
	if err != nil || progress.Progress == nil || progress.Progress.Usage.Output != 2 {
		t.Fatal("actual progress lost", err)
	}
	c.Cancel()
	<-c.Done()
	update, err := c.Poll(context.Background())
	if !errors.Is(err, context.Canceled) || len(update.Receipts) != 1 || update.Receipts[0].Usage.Input != 10 || r.state.Text != "prior memo" || r.state.Through != 1 || r.writes != 0 || c.Active() {
		t.Fatal("cancel destroyed state or measurements", r.state, err)
	}
	if _, err = c.Prepare(context.Background(), prompt.Defaults().LATCompaction, 262144, 2048, true); err != nil {
		t.Fatal("intentional cancellation poisoned later work", err)
	}
}

func TestMemoryRetryReplacesDraftAndAccountsEveryAttempt(t *testing.T) {
	r := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: "request"}}}
	c, p := planned(t, r)
	reserved := 0
	g := generation(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		for _, event := range []modelapi.Event{{Delta: "obsolete", Usage: &modelapi.TokenUsage{Input: 10, Output: 2}}, {Retry: &modelapi.Retry{Attempt: 2}}, {Delta: "final memo in free prose", Usage: &modelapi.TokenUsage{Input: 11, Output: 3}, Done: true}} {
			if err := send(event); err != nil {
				return err
			}
		}
		return nil
	})
	if err := c.Start(context.Background(), p, g, func() bool { reserved++; return true }); err != nil {
		t.Fatal(err)
	}
	<-c.Done()
	update, err := c.Poll(context.Background())
	if err != nil || reserved != 2 || len(update.Receipts) != 2 || update.Receipts[0].Usage.Input != 10 || update.Receipts[1].Usage.Input != 11 || r.state.Text != "final memo in free prose" {
		t.Fatal("retry receipt or draft mismatch", r.state, err)
	}
}

func TestFailedMemoryLeavesOriginalTailAvailable(t *testing.T) {
	for _, publishFailure := range []bool{false, true} {
		r := &repository{state: State{Revision: 1, Through: 1, Text: "prior"}, items: []Record{{Sequence: 2, Kind: "user", Text: "required change"}}, fail: publishFailure}
		c, p := planned(t, r)
		g := generation(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
			if !publishFailure {
				return errors.New("provider failed after retries")
			}
			return send(modelapi.Event{Delta: "new memo", Done: true})
		})
		if err := c.Start(context.Background(), p, g, func() bool { return true }); err != nil {
			t.Fatal(err)
		}
		<-c.Done()
		if _, err := c.Poll(context.Background()); err == nil {
			t.Fatal("summary failure hidden")
		}
		if r.state.Text != "prior" || r.state.Through != 1 {
			t.Fatal("previous state changed")
		}
		body, err := c.Tail(context.Background(), 8192, "")
		if err != nil || !strings.Contains(string(body), "required change") {
			t.Fatal("failed summary consumed original records", err)
		}
	}
}

func TestMemoryRetryDeniedByBudgetDoesNotRecordAnUnsentAttempt(t *testing.T) {
	r := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: "request"}}}
	c, p := planned(t, r)
	reserved := 0
	g := generation(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if err := send(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}}); err != nil {
			return err
		}
		return send(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2}})
	})
	if err := c.Start(context.Background(), p, g, func() bool { reserved++; return reserved == 1 }); err != nil {
		t.Fatal(err)
	}
	<-c.Done()
	update, err := c.Poll(context.Background())
	if err == nil || len(update.Receipts) != 1 || update.Receipts[0].Usage.Input != 10 || r.writes != 0 {
		t.Fatal("an unsent attempt was recorded", update, err)
	}
}

func TestMemoryMasksOriginalsPriorMemoAndGeneratedMemo(t *testing.T) {
	r := &repository{state: State{Revision: 1, Through: 1, Text: "prior secret-value"}, items: []Record{{Sequence: 2, Kind: "user", Text: "requirement secret-value"}}}
	c := &Controller{Repository: r, Session: "s", Redact: func(text string) string { return strings.ReplaceAll(text, "secret-value", "[redacted]") }}
	p, err := c.Prepare(context.Background(), prompt.Defaults().LATCompaction, 262144, 2048, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.Request.Messages[1].Content, "secret-value") {
		t.Fatal("stored secret reached a model request")
	}
	g := generation(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		return send(modelapi.Event{Delta: "generated secret-value", Done: true})
	})
	if err = c.Start(context.Background(), p, g, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	<-c.Done()
	if _, err = c.Poll(context.Background()); err != nil || strings.Contains(r.state.Text, "secret-value") {
		t.Fatal("generated secret persisted", err)
	}
}
