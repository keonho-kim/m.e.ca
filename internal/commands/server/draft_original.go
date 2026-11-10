// Purpose: Page current generation text without retaining full streamed answers in engine memory.
// Pattern: Request-owned bounded private spool with explicit generation resets.
// Usage: emit appends before publishing deltas; completion closes every retained original.
// Related: internal/commands/server/projection.go, internal/commands/server/draft_spool.go.
package server

import (
	"context"
	"errors"
	"sync"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
)

type draftOriginal struct {
	mu             sync.Mutex
	generation     uint64
	text, thinking draftText
}

type DraftPage struct {
	Window              string
	Start, End          int
	Text                string
	Generation          uint64
	Offset, Next, Total int64
	Truncated           bool
}

func (r *request) retainDraft(value record.Event, generation uint64) {
	if value.Kind == "team_notice" && value.Text == "left" {
		if d := r.originals[value.Agent]; d != nil {
			d.mu.Lock()
			r.draftCleanup = errors.Join(r.draftCleanup, d.text.close(), d.thinking.close())
			d.mu.Unlock()
			delete(r.originals, value.Agent)
		}
		return
	}
	if value.Kind != "stream" && value.Kind != "generation_started" {
		return
	}
	if r.originals == nil {
		return
	}
	d := r.originals[value.Agent]
	if d == nil {
		if len(r.originals) >= 16 {
			return
		}
		d = &draftOriginal{}
		r.originals[value.Agent] = d
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != generation {
		r.draftCleanup = errors.Join(r.draftCleanup, d.text.close(), d.thinking.close())
		d.generation = generation
	}
	if value.Kind == "stream" {
		d.text.append(value.Text)
		d.thinking.append(value.Thinking)
	}
}

func (r *request) closeDrafts() error {
	r.outputMu.Lock()
	defer r.outputMu.Unlock()
	err := r.draftCleanup
	for _, d := range r.originals {
		d.mu.Lock()
		err = errors.Join(err, d.text.close(), d.thinking.close())
		d.mu.Unlock()
	}
	r.originals = nil
	r.draftCleanup = nil
	return err
}

func (s *Service) ReadDraft(ctx context.Context, task, agent string, generation uint64, offset int64, limit int, thinking bool) (DraftPage, error) {
	if offset < 0 || limit < 4 || limit > 64<<10 || len(agent) > 128 {
		return DraftPage{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	s.mu.Lock()
	r, err := s.currentRequest(task)
	s.mu.Unlock()
	if err != nil {
		return DraftPage{}, err
	}
	r.outputMu.Lock()
	d := r.originals[agent]
	r.outputMu.Unlock()
	if d == nil {
		return DraftPage{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerPageInvalid)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if generation != d.generation {
		return DraftPage{}, faults.New(diagcatalog.Conflict, diagcatalog.ServerPageInvalid)
	}
	source := &d.text
	if thinking {
		source = &d.thinking
	}
	if offset > source.total || ctx.Err() != nil {
		return DraftPage{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	start, end := max(0, offset-redaction.MaxCredentialBytes), min(source.total, offset+int64(limit)+redaction.MaxCredentialBytes)
	window, err := source.page(start, int(end-start))
	if err != nil {
		return DraftPage{}, err
	}
	from := int(offset - start)
	to := min(len(window), from+limit)
	text := window[from:to]
	if len(text) > 0 && !utf8.RuneStart(text[0]) {
		return DraftPage{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	for len(text) > 0 && !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	return DraftPage{Window: window, Start: from, End: from + len(text), Text: text, Generation: generation, Offset: offset, Next: offset + int64(len(text)), Total: source.total}, nil
}
