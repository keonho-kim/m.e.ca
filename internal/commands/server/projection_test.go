// Purpose: Verify replay bounds, retry resets and immutable projections independently of I/O.
// Pattern: Deterministic event contract tests and streaming benchmark.
// Usage: go test ./internal/commands/server; go test -bench BenchmarkStream -benchmem ./internal/commands/server.
// Related: internal/commands/server/projection.go, internal/commands/server/events.go.
package server

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config"
)

func projectionFixture() (*Service, *request) {
	s := &Service{ctx: context.Background(), options: Options{Config: config.Defaults()}, feed: newEventLog()}
	r := &request{task: Task{ID: "task-test"}, drafts: map[string]*draftState{}, approvals: map[string]*pendingApproval{}}
	s.active = r
	return s, r
}

func TestReplayBoundsAndExpiredCursorSnapshot(t *testing.T) {
	s, r := projectionFixture()
	old := s.State().Cursor
	text := strings.Repeat("a", maxTextBytes)
	for i := 0; i < maxEvents*2; i++ {
		s.emit(r, record.Event{Kind: "progress", Text: text})
	}
	if len(s.feed.entries) > maxEvents || s.feed.bytes > maxEventBytes {
		t.Fatal(len(s.feed.entries), s.feed.bytes)
	}
	batch, err := s.Next(t.Context(), old)
	if err != nil || batch.Snapshot == nil || batch.Snapshot.Task.ID != "task-test" {
		t.Fatal(batch, err)
	}
	if _, err := s.Next(t.Context(), "invalid"); err == nil {
		t.Fatal("malformed cursor accepted")
	}
	current := s.State().Cursor
	s.feed = newEventLog()
	batch, err = s.Next(t.Context(), current)
	if err != nil || batch.Snapshot == nil {
		t.Fatal(batch, err)
	}
	for i := 0; i < maxEvents*2; i++ {
		s.emit(r, record.Event{Kind: "progress"})
	}
	if len(s.feed.entries) != maxEvents {
		t.Fatal("count bound was not enforced", len(s.feed.entries))
	}
}

func TestDeltaRetryAndBorrowedToolBuffers(t *testing.T) {
	s, r := projectionFixture()
	cursor := s.State().Cursor
	s.emit(r, record.Event{Kind: "generation_started", Agent: "Hana"})
	s.emit(r, record.Event{Kind: "stream", Agent: "Hana", Text: "hello", Thinking: "plan"})
	s.emit(r, record.Event{Kind: "stream", Agent: "Hana", Text: "hello world", Thinking: "plan"})
	s.emit(r, record.Event{Kind: "generation_started", Agent: "Hana"})
	s.emit(r, record.Event{Kind: "stream", Agent: "Hana", Text: "replacement"})
	call := tool.Call{ID: "c", Name: "write_file", Arguments: []byte(`{"path":"a"}`)}
	s.emit(r, record.Event{Kind: "tool_succeeded", Call: &call, Output: []byte(`{"large":"original"}`), Reference: &record.EventReference{Session: "session-test", Sequence: 7}})
	call.Arguments[0] = '!'
	batch, err := s.Next(t.Context(), cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Events) != 6 {
		t.Fatal(len(batch.Events))
	}
	if !batch.Events[0].Reset || batch.Events[1].Record.Text != "hello" || batch.Events[2].Record.Text != " world" || !batch.Events[3].Reset || batch.Events[4].Generation != 2 {
		t.Fatal(batch.Events)
	}
	last := batch.Events[5]
	if !last.Truncated || len(last.Record.Output) != 0 || last.Record.Reference.Sequence != 7 || last.Record.Call.Arguments[0] != '{' {
		t.Fatal(last)
	}
}

func TestLargeDraftDoesNotRetainFullSnapshots(t *testing.T) {
	s, r := projectionFixture()
	text := strings.Repeat("界", 100000)
	s.emit(r, record.Event{Kind: "generation_started"})
	s.emit(r, record.Event{Kind: "stream", Text: text, Thinking: text})
	state := s.State()
	if len(state.Drafts) != 1 || !state.Drafts[0].Truncated || len(state.Drafts[0].Text) > maxDraftBytes || s.feed.bytes > maxEventBytes {
		t.Fatal(state.Drafts, s.feed.bytes)
	}
}

func TestReplayByteBudgetIncludesCompletionErrors(t *testing.T) {
	s, _ := projectionFixture()
	for range maxEvents {
		s.publish(Event{Status: "interrupted", Error: strings.Repeat("e", 4096), Record: record.Event{Kind: "request_finished"}})
	}
	retained := 0
	for _, entry := range s.feed.entries {
		e := entry.value
		retained += len(e.Error) + len(e.Status) + len(e.Cursor) + len(e.Task) + len(e.Record.Kind)
	}
	if retained > maxEventBytes {
		t.Fatalf("retained strings exceed byte budget: %d > %d", retained, maxEventBytes)
	}
}

func BenchmarkStreamProjection(b *testing.B) {
	// A slow or absent observer never changes this engine-side work.
	text := strings.Repeat("x", 256<<10)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		s, r := projectionFixture()
		s.emit(r, record.Event{Kind: "generation_started"})
		for n := 4096; n <= len(text); n += 4096 {
			s.emit(r, record.Event{Kind: "stream", Text: text[:n]})
		}
	}
}
