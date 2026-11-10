// Purpose: Retain bounded ordered events and wake independent network readers without engine backpressure.
// Pattern: Process-owned replay log with cursor snapshots.
// Usage: Every transport calls Next; runtime publication never performs network I/O.
// Related: internal/commands/server/projection.go, internal/commands/server/snapshot.go.
package server

import (
	"crypto/rand"
	"strconv"
	"sync"
	"time"

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
type Feed = eventLog

func NewFeed() *Feed { return newEventLog() }

type eventLog struct {
	mu          sync.Mutex
	epoch       string
	sequence    uint64
	entries     []loggedEvent
	head, count int
	bytes       int
	changed     chan struct{}
}

func newEventLog() *eventLog { return &eventLog{epoch: rand.Text()} }
func (f *eventLog) wake()    { f.mu.Lock(); defer f.mu.Unlock(); f.wakeLocked() }
func (f *eventLog) wakeLocked() {
	if f.changed != nil {
		close(f.changed)
		f.changed = nil
	}
}
func (f *eventLog) cursor() string { f.mu.Lock(); defer f.mu.Unlock(); return f.cursorLocked() }
func (f *eventLog) cursorLocked() string {
	var buffer [64]byte
	n := copy(buffer[:], f.epoch)
	buffer[n] = ':'
	return string(strconv.AppendUint(buffer[:n+1], f.sequence, 10))
}

func eventKind(kind string) record.Event { return record.Event{Kind: kind} }

// publish requires s.mu. The log owns immutable, bounded projections only.
func (s *Service) publish(value Event) {
	if value.Session == "" {
		if r := s.active[value.Task]; r != nil {
			value.Session = r.task.Session
		} else if s.conversation != nil {
			value.Session = s.conversation.Runtime.Session.ID
		}
	}
	if value.Record.OccurredAt == "" {
		value.Record.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	value = ProjectEvent(value)
	if r := s.active[value.Task]; r != nil {
		r.sequence++
		value.Sequence = r.sequence
	}
	value.Workspace = s.options.WorkspaceID
	f := s.feed
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sequence++
	value.Cursor = f.cursorLocked()
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
	f.wakeLocked()
}

func (f *eventLog) entry(index int) loggedEvent {
	return f.entries[(f.head+index)%len(f.entries)]
}
