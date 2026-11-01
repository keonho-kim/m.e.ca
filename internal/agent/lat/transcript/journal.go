// Purpose: Keep one native LAT conversation local while auditing it in its parent session.
// Pattern: Journal adapter with phase-scoped history ownership.
// Usage: A LAT phase supplies this journal to a native runner and closes it after settlement.
// Related: internal/agent/record/events.go, internal/agent/run/runner.go, internal/storage/sqlite/events.go.
package transcript

import (
	"context"
	"crypto/rand"
	"fmt"
	"slices"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

// Committer owns the parent session's atomic durability boundary.
type Committer interface {
	Commit(context.Context, string, []runrecord.Entry) error
}

// Journal owns a single sequential phase; its native history is never parent history.
type Journal struct {
	parent     Committer
	session    string
	scope      runrecord.LATRun
	messages   []messageRecord
	sequence   int64
	latestUser int64
	thinking   string
	sample     contextSample
	memory     compact.Memory
}

type messageRecord struct {
	message  modelapi.Message
	sequence int64
	bytes    int
}

func New(parent Committer, session, parentRunID, phase string) (*Journal, error) {
	if parent == nil || session == "" {
		return nil, fmt.Errorf("LAT transcript requires its parent journal and session")
	}
	scope := runrecord.LATRun{ID: "lat_" + rand.Text(), ParentRunID: parentRunID, Phase: phase}
	if err := scope.Check(); err != nil {
		return nil, err
	}
	return &Journal{parent: parent, session: session, scope: scope}, nil
}

// Scope identifies this phase consistently in live output and durable history.
func (j *Journal) Scope() runrecord.LATRun { return j.scope }

func (j *Journal) ContextMessages(ctx context.Context, session string, after int64, maxBytes int, images bool) ([]modelapi.Message, error) {
	if err := j.check(ctx, session); err != nil {
		return nil, err
	}
	if after < j.memory.Through || after > j.sequence || maxBytes < 1 {
		return nil, fmt.Errorf("invalid LAT context bounds; earlier native history was compacted")
	}
	messages := make([]modelapi.Message, 0, len(j.messages))
	total := 0
	for _, record := range j.messages {
		if record.sequence <= after && (record.sequence != j.latestUser || record.sequence > j.memory.Through) {
			continue
		}
		message, size := record.message, record.bytes
		if !images && len(message.Images) > 0 {
			message.Images = nil
			message.Content += modelapi.ImageOmissionNotice
			size = message.ContextBytes()
		}
		total += size
		if total > maxBytes {
			return nil, fmt.Errorf("uncompacted LAT conversation exceeds %d bytes; no messages were omitted", maxBytes)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (j *Journal) RecordEvent(ctx context.Context, session string, event runrecord.Event) error {
	return j.Commit(ctx, session, []runrecord.Entry{{Event: &event}})
}

func (j *Journal) Commit(ctx context.Context, session string, entries []runrecord.Entry) error {
	if err := j.check(ctx, session); err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("LAT journal group must not be empty")
	}
	var messages []messageRecord
	sequence := j.sequence
	thinking := j.thinking
	records := make([]runrecord.Entry, 0, len(entries))
	for _, entry := range entries {
		var event runrecord.Event
		switch {
		case entry.Message != nil && entry.Event == nil:
			message := ownMessage(*entry.Message)
			if err := message.Check(); err != nil {
				return err
			}
			messages = append(messages, messageRecord{message: message, sequence: sequence + 1, bytes: message.ContextBytes()})
			audit := message
			if message.Role == "assistant" {
				// A completed generation already wrote this trace as a thinking
				// event. Preserve native context locally without a second DB copy.
				if message.Reasoning == thinking {
					audit.Reasoning = ""
				}
				thinking = ""
			}
			event = runrecord.Event{Kind: "lat_message", Message: &audit}
		case entry.Event != nil && entry.Message == nil:
			event = *entry.Event
			if event.Kind == "run_checkpoint" && event.Checkpoint == nil {
				return fmt.Errorf("LAT checkpoint requires a state")
			}
			if err := event.Check(); err != nil {
				return err
			}
			var keep bool
			event, keep = auditEvent(event)
			if !keep {
				continue
			}
			if event.Kind == "thinking" {
				thinking = event.Thinking
			}
		default:
			return fmt.Errorf("LAT journal record requires exactly one payload")
		}
		sequence++
		scope := j.scope
		event.Internal, event.LAT = true, &scope
		if err := event.Check(); err != nil {
			return err
		}
		records = append(records, runrecord.Entry{Event: &event})
	}
	if len(records) == 0 {
		return nil
	}
	if err := j.parent.Commit(ctx, j.session, records); err != nil {
		return err
	}
	j.messages = append(j.messages, messages...)
	j.sequence, j.thinking = sequence, thinking
	previousUser := j.latestUser
	for _, record := range messages {
		if record.message.Role == "user" {
			j.latestUser = record.sequence
		}
	}
	if previousUser != j.latestUser && previousUser <= j.memory.Through && j.memory.Through > 0 {
		j.pruneMessages()
	}
	return nil
}

// Close releases the phase's retained native history after its final durable flush.
func (j *Journal) Close() {
	clear(j.messages)
	*j = Journal{}
}

func (j *Journal) check(ctx context.Context, session string) error {
	if j.parent == nil {
		return fmt.Errorf("LAT transcript is closed")
	}
	if session != j.session {
		return fmt.Errorf("LAT transcript cannot write another session")
	}
	return ctx.Err()
}

func ownMessage(message modelapi.Message) modelapi.Message {
	// Native payload bodies follow the model's immutable history contract.
	message.Calls = slices.Clone(message.Calls)
	message.Images = slices.Clone(message.Images)
	if message.Thinking != nil {
		thinking := *message.Thinking
		message.Thinking = &thinking
	}
	return message.SealHistory()
}
