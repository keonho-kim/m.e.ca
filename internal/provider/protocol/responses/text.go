// Purpose: Reconcile streamed Responses content with authoritative per-part and terminal text.
// Pattern: Attempt-owned ordered content accumulator.
// Usage: Decoder tracks displayed deltas and produces the final update at completion or interruption.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/responses/complete.go.
package responses

import (
	"cmp"
	"slices"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type partKey struct{ output, kind, index int }

const (
	answerPart = iota
	reasoningPart
	summaryPart
)

type span struct{ start, end int }
type textPart struct {
	size      int
	spans     []span
	final     string
	finalized bool
}

// Bound metadata independently of payload bytes for alternating single-byte parts.
const maxPartEntries = 64 * 1024

type outputText struct {
	hasSnapshot                    bool
	items                          map[int]map[partKey]*textPart
	partCount, spanCount, retained int
	answer, reasoning              strings.Builder
}

func (t *outputText) set(key partKey, value string, final bool) error {
	if key.output < 0 || key.output >= 1024 || key.index < 0 || key.index >= maxPartEntries {
		return faults.Errorf(diag.ResponsesInvalidResponsesOutputIndex)
	}
	group := t.items[key.output]
	part := group[key]
	if part == nil {
		if t.partCount >= maxPartEntries {
			return &modelapi.OutputLimitError{Reason: "Responses content part limit"}
		}
		part = &textPart{}
	}
	size := len(value)
	if !final {
		size += part.size
	}
	total := t.retained - part.size + size
	if total > modelapi.MaxResponseBytes || !final && t.answer.Len()+t.reasoning.Len()+len(value) > modelapi.MaxResponseBytes {
		return &modelapi.OutputLimitError{Reason: "Responses text buffer limit"}
	}
	if group == nil {
		if t.items == nil {
			t.items = make(map[int]map[partKey]*textPart)
		}
		group = make(map[partKey]*textPart)
		t.items[key.output] = group
	}
	if group[key] == nil {
		group[key] = part
		t.partCount++
	}
	if final {
		t.hasSnapshot = true
		part.final, part.finalized = value, true
		t.spanCount -= len(part.spans)
		part.spans = nil
	} else if value != "" {
		live := &t.answer
		if key.kind != answerPart {
			live = &t.reasoning
		}
		start := live.Len()
		end := start + len(value)
		if n := len(part.spans); n > 0 && part.spans[n-1].end == start {
			part.spans[n-1].end = end
		} else {
			if t.spanCount >= maxPartEntries {
				return &modelapi.OutputLimitError{Reason: "Responses interleaved content limit"}
			}
			part.spans = append(part.spans, span{start, end})
			t.spanCount++
		}
		live.WriteString(value)
	}
	t.retained, part.size = total, size
	return nil
}

func (t *outputText) observeItem(index int, item outputItem) error {
	// A message item owns all its content. Reasoning can be returned as
	// opaque encrypted state with no display text in the terminal snapshot.
	t.removeParts(index, func(k partKey) bool {
		switch k.kind {
		case answerPart:
			return item.Type == "message" && k.index >= len(item.Content)
		case reasoningPart:
			return len(item.Content) > 0 && k.index >= len(item.Content)
		default:
			return len(item.Summary) > 0 && k.index >= len(item.Summary)
		}
	})
	for i, p := range item.Content {
		kind := answerPart
		if item.Type == "reasoning" {
			kind = reasoningPart
		}
		if err := t.set(partKey{index, kind, i}, p.value(), true); err != nil {
			return err
		}
	}
	for i, p := range item.Summary {
		if err := t.set(partKey{index, summaryPart, i}, p.Text, true); err != nil {
			return err
		}
	}
	return nil
}

func (t *outputText) removeParts(index int, match func(partKey) bool) {
	group := t.items[index]
	for k, p := range group {
		if match(k) {
			delete(group, k)
			t.retained -= p.size
			t.spanCount -= len(p.spans)
			t.partCount--
		}
	}
	if len(group) == 0 {
		delete(t.items, index)
	}
}

func (t *outputText) completion() modelapi.Event {
	keys := make([]partKey, 0, t.partCount)
	for _, group := range t.items {
		for k := range group {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b partKey) int {
		return cmp.Or(cmp.Compare(a.output, b.output), cmp.Compare(a.kind, b.kind), cmp.Compare(a.index, b.index))
	})
	var texts, thoughts []string
	for _, k := range keys {
		values, live := &texts, t.answer.String()
		if k.kind != answerPart {
			values, live = &thoughts, t.reasoning.String()
		}
		p := t.items[k.output][k]
		if p.finalized {
			*values = append(*values, p.final)
			continue
		}
		for _, s := range p.spans {
			*values = append(*values, live[s.start:s.end])
		}
	}
	text, thought := strings.Join(texts, ""), strings.Join(thoughts, "")
	delta, ok := strings.CutPrefix(text, t.answer.String())
	thinking, okThinking := strings.CutPrefix(thought, t.reasoning.String())
	if ok && okThinking {
		return modelapi.Event{Delta: delta, Thinking: thinking}
	}
	return modelapi.Event{ReplaceOutput: true, Delta: text, Thinking: thought}
}

// EmitPartial publishes text confirmed by a part or item when the stream ends
// without a successful completion. It never releases calls or opaque state.
func (s *Decoder) EmitPartial(streaming bool, emit func(modelapi.Event) error) error {
	if !s.text.hasSnapshot {
		return nil
	}
	update := s.text.completion()
	if !update.ReplaceOutput && update.Delta == "" && update.Thinking == "" {
		return nil
	}
	update.Streaming = streaming
	return emit(update)
}
