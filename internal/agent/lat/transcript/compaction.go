// Purpose: Compact one LAT phase through private memory and token state without touching parent context rows.
// Pattern: In-memory compaction repository with a durable audit boundary.
// Usage: Each side runner supplies its journal to a separate compact.Controller.
// Related: internal/agent/lat/transcript/journal.go, internal/agent/compact/controller.go, internal/storage/history/boundary.go.
package transcript

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/keonho-kim/meca/internal/agent/compact"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
)

var _ compact.Repository = (*Journal)(nil)

type contextSample struct {
	count             compact.TokenCount
	sequence, through int64
}

func (j *Journal) AddContextTokens(ctx context.Context, session string, count compact.TokenCount) error {
	if err := j.check(ctx, session); err != nil {
		return err
	}
	if count.Kind != "request" || count.Amount < 0 || count.Amount > 1000000000 {
		return fmt.Errorf("invalid LAT context token sample")
	}
	j.sample = contextSample{count: count, sequence: j.sample.sequence + 1, through: j.sequence}
	return nil
}

func (j *Journal) Memory(ctx context.Context, session string) (compact.Memory, error) {
	if err := j.check(ctx, session); err != nil {
		return compact.Memory{}, err
	}
	return j.memory, nil
}

func (j *Journal) MemoryWatermark(ctx context.Context, session string, after int64) (compact.Watermark, error) {
	if err := j.check(ctx, session); err != nil {
		return compact.Watermark{}, err
	}
	if after < 0 || after > j.sample.sequence {
		return compact.Watermark{}, fmt.Errorf("invalid LAT token watermark")
	}
	mark := compact.Watermark{Usage: after, Timeline: j.sequence}
	through := j.memory.Through
	if j.sample.sequence > after {
		mark.Tokens, mark.Usage = j.sample.count.Amount, j.sample.sequence
		mark.Estimated = j.sample.count.Estimated
		through = max(through, j.sample.through)
	}
	var trailingBytes int64
	for _, record := range j.messagesAfter(through) {
		if record.message.Role == "user" || record.message.Role == "tool" || record.message.Role == "system" {
			trailingBytes += int64(len(record.message.Content))
		}
	}
	mark.Tokens += compact.EstimateBytes(trailingBytes).Amount
	mark.Estimated = mark.Estimated || trailingBytes > 0
	return mark, nil
}

func (j *Journal) CompactionBoundary(ctx context.Context, session string, through int64, keepBytes int) (int64, error) {
	if err := j.check(ctx, session); err != nil {
		return 0, err
	}
	if through < j.memory.Through || through > j.sequence || keepBytes < 0 {
		return 0, fmt.Errorf("invalid LAT compaction bounds")
	}
	selection := history.TailSelection{Target: keepBytes}
	imageBoundary := j.unreadImageBoundary(through)
	for i := len(j.messages) - 1; i >= 0; i-- {
		record := j.messages[i]
		if record.sequence > through {
			continue
		}
		if boundary, ready := selection.Keep(record.sequence, record.bytes, record.message); ready {
			return max(j.memory.Through, min(boundary, imageBoundary)), nil
		}
	}
	return j.memory.Through, nil
}

func (j *Journal) unreadImageBoundary(through int64) int64 {
	boundary, assistant := through, int64(0)
	for _, record := range j.messages {
		if record.sequence > through {
			break
		}
		message := record.message
		if message.Role == "assistant" {
			assistant = record.sequence
			if !message.Incomplete {
				boundary = through
			}
		}
		if message.Role == "tool" && len(message.Images) > 0 {
			start := assistant
			if start == 0 {
				start = record.sequence
			}
			boundary = min(boundary, start-1)
		}
	}
	return boundary
}

func (j *Journal) MemoryRecords(ctx context.Context, session string, after, through int64) ([]compact.Record, int64, error) {
	if err := j.check(ctx, session); err != nil {
		return nil, after, err
	}
	if after < j.memory.Through || through < after || through > j.sequence {
		return nil, after, fmt.Errorf("invalid LAT compaction source bounds")
	}
	var page []history.Record
	cursor, size := after, 0
	for _, record := range j.messagesAfter(after) {
		if record.sequence > through {
			break
		}
		if len(page) == 64 || size+len(record.message.Content) > compact.MaxBodyBytes && len(page) > 0 {
			return projectPage(page, cursor)
		}
		page = append(page, history.Record{Sequence: record.sequence, Kind: "message", Message: record.message})
		size += len(record.message.Content)
		cursor = record.sequence
	}
	return projectPage(page, through)
}

func (j *Journal) InstallMemory(ctx context.Context, session string, expected int, next compact.Memory) error {
	if err := j.check(ctx, session); err != nil {
		return err
	}
	if expected != j.memory.Revision || next.Revision != expected+1 || next.Through <= j.memory.Through || next.Through > j.sequence || next.UsageThrough < j.memory.UsageThrough || next.UsageThrough > j.sample.sequence {
		return fmt.Errorf("stale or invalid LAT compaction memory")
	}
	if err := next.Check(compact.MaxBodyBytes); err != nil {
		return err
	}
	metadata, err := json.Marshal(struct {
		Revision     int   `json:"revision"`
		Through      int64 `json:"through"`
		UsageThrough int64 `json:"usage_through"`
	}{next.Revision, next.Through, next.UsageThrough})
	if err != nil {
		return err
	}
	if err = j.RecordEvent(ctx, session, runrecord.Event{Kind: "lat_compaction", Text: next.Markdown, Capture: metadata}); err != nil {
		return err
	}
	j.memory = next
	j.pruneMessages()
	return nil
}

func (j *Journal) pruneMessages() {
	retained := j.messages[:0]
	for _, record := range j.messages {
		if record.sequence > j.memory.Through || record.sequence == j.latestUser {
			retained = append(retained, record)
		}
	}
	clear(j.messages[len(retained):])
	j.messages = retained
}

func projectPage(page []history.Record, through int64) ([]compact.Record, int64, error) {
	records, err := history.Compaction(page)
	return records, through, err
}

// Message sequences remain ordered even after summaries release older records.
func (j *Journal) messagesAfter(sequence int64) []messageRecord {
	index := sort.Search(len(j.messages), func(i int) bool { return j.messages[i].sequence > sequence })
	return j.messages[index:]
}
