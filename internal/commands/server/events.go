// Purpose: Retain bounded ordered events and wake independent network readers without engine backpressure.
// Pattern: Process-owned replay log with cursor snapshots.
// Usage: Every transport calls Next; runtime publication never performs network I/O.
// Related: internal/commands/server/projection.go, internal/commands/server/snapshot.go.
package server

import (
	"context"
	"crypto/rand"
	"strconv"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/record"
)

const (
	maxEvents     = 2048
	maxEventBytes = 8 << 20
	maxTextBytes  = 16 << 10
	maxDraftBytes = 128 << 10
)

type loggedEvent struct {
	value    Event
	bytes    int
	sequence uint64
}
type eventLog struct {
	epoch       string
	sequence    uint64
	entries     []loggedEvent
	head, count int
	bytes       int
	changed     chan struct{}
}

func newEventLog() eventLog { return eventLog{epoch: rand.Text()} }
func (f *eventLog) wake() {
	if f.changed != nil {
		close(f.changed)
		f.changed = nil
	}
}
func (f *eventLog) cursor() string {
	var buffer [64]byte
	n := copy(buffer[:], f.epoch)
	buffer[n] = ':'
	return string(strconv.AppendUint(buffer[:n+1], f.sequence, 10))
}

func eventKind(kind string) record.Event { return record.Event{Kind: kind} }

// publish requires s.mu. The log owns immutable, bounded projections only.
func (s *Service) publish(value Event) {
	if value.Session == "" {
		if s.active != nil && value.Task == s.active.task.ID {
			value.Session = s.active.task.Session
		} else if s.conversation != nil {
			value.Session = s.conversation.Runtime.Session.ID
		}
	}
	if value.Task == "" && s.active != nil {
		value.Task = s.active.task.ID
	}
	value = ProjectEvent(value)
	f := &s.feed
	f.sequence++
	value.Cursor = f.cursor()
	size := eventBytes(value)
	for f.count > 0 && (f.count >= maxEvents || f.bytes+size > maxEventBytes) {
		f.bytes -= f.entries[f.head].bytes
		f.entries[f.head] = loggedEvent{}
		f.head = (f.head + 1) % len(f.entries)
		f.count--
	}
	if f.count == len(f.entries) {
		entries := make([]loggedEvent, min(maxEvents, max(1, 2*len(f.entries))))
		for i := 0; i < f.count; i++ {
			entries[i] = f.entry(i)
		}
		f.entries, f.head = entries, 0
	}
	f.entries[(f.head+f.count)%len(f.entries)] = loggedEvent{value, size, f.sequence}
	f.count++
	f.bytes += size
	f.wake()
}

func (s *Service) Next(ctx context.Context, cursor string) (Batch, error) {
	for {
		s.mu.Lock()
		batch, ready, err := s.readEvents(cursor)
		if err != nil || ready {
			s.mu.Unlock()
			return batch, err
		}
		if s.closed {
			s.mu.Unlock()
			return Batch{}, failure("unavailable", "server is shutting down")
		}
		if s.feed.changed == nil {
			s.feed.changed = make(chan struct{})
		}
		changed := s.feed.changed
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return Batch{}, ctx.Err()
		case <-changed:
		}
	}
}

// Replay validates a cursor and reads available events without waiting. SSE
// uses it to establish response headers even when the subscriber is caught up.
func (s *Service) Replay(cursor string) (Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, _, err := s.readEvents(cursor)
	return batch, err
}

func (s *Service) readEvents(cursor string) (Batch, bool, error) {
	f := &s.feed
	if cursor == "" {
		snapshot := s.snapshot()
		return Batch{Snapshot: &snapshot}, true, nil
	}
	epoch, raw, ok := strings.Cut(cursor, ":")
	sequence, err := strconv.ParseUint(raw, 10, 64)
	if !ok || err != nil || len(epoch) != 26 {
		return Batch{}, false, failure("invalid_argument", "invalid event cursor")
	}
	if epoch != f.epoch || (f.count > 0 && sequence < f.entry(0).sequence-1) {
		snapshot := s.snapshot()
		return Batch{Snapshot: &snapshot}, true, nil
	}
	if sequence > f.sequence {
		return Batch{}, false, failure("invalid_argument", "event cursor is ahead of this server")
	}
	if sequence == f.sequence {
		return Batch{}, false, nil
	}
	// Retained sequences are contiguous, so a cursor locates its next entry
	// directly regardless of how much older history remains in the log.
	start := int(sequence + 1 - f.entry(0).sequence)
	end := min(start+32, f.count)
	values := make([]Event, end-start)
	for i := range values {
		values[i] = f.entry(start + i).value
	}
	return Batch{Events: values}, len(values) > 0, nil
}

func (f *eventLog) entry(index int) loggedEvent {
	return f.entries[(f.head+index)%len(f.entries)]
}
