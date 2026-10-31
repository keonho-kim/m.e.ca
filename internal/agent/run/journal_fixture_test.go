// Purpose: Preserve ordered original records and atomic groups in scripted runtime tests.
// Pattern: In-memory journal fixture.
// Usage: Runtime tests share this journal; LAT sources read its chronological timeline.
// Related: internal/agent/run/run_test.go, internal/agent/run/lat_memory_fixture_test.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type journal struct {
	failKind string
	messages []modelapi.Message
	events   []runrecord.Event
	timeline []history.Record
}

func (j *journal) ContextMessages(_ context.Context, _ string, after int64, maxBytes int, _ bool) ([]modelapi.Message, error) {
	if after != 0 {
		return nil, fmt.Errorf("test journal does not support compaction")
	}
	bytes := 0
	for _, m := range j.messages {
		bytes += len(m.Content) + len(m.Calls)
	}
	if bytes > maxBytes {
		return nil, fmt.Errorf("conversation limit exceeded")
	}
	return append([]modelapi.Message(nil), j.messages...), nil
}
func (j *journal) AppendMessage(_ context.Context, _ string, m modelapi.Message) error {
	if len(j.timeline) == 0 {
		for _, previous := range j.messages {
			j.timeline = append(j.timeline, history.Record{Sequence: int64(len(j.timeline) + 1), Kind: "message", Message: previous})
		}
	}
	j.messages = append(j.messages, m)
	j.timeline = append(j.timeline, history.Record{Sequence: int64(len(j.timeline) + 1), Kind: "message", Message: m})
	return nil
}
func (j *journal) RecordEvent(_ context.Context, _ string, e runrecord.Event) error {
	if j.failKind != "" && e.Kind == j.failKind {
		return fmt.Errorf("simulated journal failure")
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	var copy runrecord.Event
	if err = json.Unmarshal(body, &copy); err != nil {
		return err
	}
	j.events = append(j.events, copy)
	j.timeline = append(j.timeline, history.Record{Sequence: int64(len(j.timeline) + 1), Kind: "event", Event: copy})
	return nil
}
func (j *journal) Commit(ctx context.Context, session string, records []runrecord.Entry) error {
	copy := &journal{failKind: j.failKind, messages: append([]modelapi.Message(nil), j.messages...), events: append([]runrecord.Event(nil), j.events...), timeline: append([]history.Record(nil), j.timeline...)}
	for _, r := range records {
		if r.Message != nil {
			if err := copy.AppendMessage(ctx, session, *r.Message); err != nil {
				return err
			}
		}
		if r.Event != nil {
			if err := copy.RecordEvent(ctx, session, *r.Event); err != nil {
				return err
			}
		}
	}
	j.messages, j.events, j.timeline = copy.messages, copy.events, copy.timeline
	return nil
}
