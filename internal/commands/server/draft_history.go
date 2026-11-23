// Purpose: Retire memory-backed drafts once matching text is available in durable history.
// Pattern: Versioned reference with bounded history fallback.
// Usage: Stream projection reclaims stored text; draft pagination resolves the same generation.
// Related: internal/commands/server/draft_original.go, internal/commands/server/detail.go.
package server

import (
	"bytes"
	"context"

	"github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

// Called with request output and service state held. It never reads storage.
func (r *request) reclaimStoredText(value record.Event) {
	if (value.Kind != "progress" && value.Kind != "team_progress") || value.HistorySequence <= 0 || value.Presentation == "" ||
		r.conversation == nil || r.conversation.History == nil || r.task.Session == "" {
		return
	}
	draft, original := r.drafts[value.Agent], r.originals[value.Agent]
	if draft == nil || original == nil || draft.Presentation != value.Presentation || draft.textBytes != len(value.Text) {
		return
	}
	original.mu.Lock()
	defer original.mu.Unlock()
	if original.generation != draft.Generation {
		return
	}
	text := &original.text
	switch {
	case len(text.memory) != 0 && bytes.Equal(text.memory, []byte(value.Text)):
		text.historySequence = value.HistorySequence
		text.clearMemory()
	case text.file != nil:
		// Large originals stay seekable in the bounded disk spool.
	default:
		return
	}
	draft.Text = ""
	draft.text.clear()
	draft.Truncated = draft.textBytes != 0 || draft.thinkingBytes > len(draft.Thinking)
}

func (s *Service) storedDraftWindow(ctx context.Context, session string, sequence, total, from, to int64) (string, error) {
	// Only small drafts are backed by history here. Bound escaped text and metadata too.
	const historyReadBytes = 8 * maxDraftBytes
	read, err := s.OpenDetail(ctx, DetailSource{Session: session, HistorySequence: sequence, View: DetailText})
	if err != nil {
		return "", err
	}
	defer read.Release()
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	value, err := read.Load(func(bytes int) error {
		if bytes > historyReadBytes {
			return faults.New(diagcatalog.ResourceExhausted, diagcatalog.DetailSpoolBudget)
		}
		return memory.Resize(bytes)
	})
	if err != nil {
		return "", err
	}
	if value.History == nil || int64(len(value.History.Event.Text)) != total {
		return "", faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
	}
	return value.History.Event.Text[from:to], nil
}
