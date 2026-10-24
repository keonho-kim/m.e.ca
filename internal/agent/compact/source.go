// Purpose: Fit one chronological summary request while preserving user messages in full.
// Pattern: Bounded source selection.
// Usage: Controller adds projected records, then sends one packet to summarize.
// Related: internal/agent/compact/controller.go, internal/storage/history/compaction.go.
package compact

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type sourceEntry struct {
	key       [32]byte
	encoded   json.RawMessage
	mandatory bool
	important bool
}
type sourceBuffer struct {
	entries                         list.List
	normal, important               []*list.Element
	bytes, required, limit, omitted int
}

func (b *sourceBuffer) add(record Record, redact func(string) string) error {
	if strings.TrimSpace(record.Text) == "" {
		return nil
	}
	mandatory := record.Role == "user"
	text := record.Text
	available := b.limit - b.required
	if !mandatory {
		limit := MaxBodyBytes
		if available >= 512 {
			limit = min(limit, available)
		} else if len(text) > available {
			b.omitted++
			return nil
		}
		text = sourceExcerpt(text, limit)
	}
	payload := struct{ Role, Text string }{record.Role, redact(text)}
	key, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if !mandatory && len(key)+1 > available && available >= 512 {
		// JSON can encode one byte as six ASCII bytes (\uXXXX). Keep both
		// ends of an oversized observation and reserve its envelope overhead.
		payload.Text = sourceExcerpt(payload.Text, (available-64)/6)
		key, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	digest := sha256.Sum256(key)
	encoded := key
	important := record.Role == "outcome" || record.Role == "failure"
	size := len(encoded) + 1
	if !mandatory && b.entries.Len() > 0 {
		last := b.entries.Back().Value.(sourceEntry)
		if !last.mandatory && last.key == digest {
			return nil
		}
	}
	if mandatory && b.required+size > b.limit {
		return fmt.Errorf("compaction user instructions exceed the available summary input budget; previous memory and original history retained; increase model context or adjust compaction input/output limits")
	}
	if !mandatory && size > b.limit-b.required {
		b.omitted++
		return nil
	}
	for b.bytes+size > b.limit {
		queue := &b.normal
		if len(*queue) == 0 {
			queue = &b.important
		}
		if len(*queue) == 0 {
			return fmt.Errorf("compaction source budget exhausted")
		}
		remove := (*queue)[0]
		entry := remove.Value.(sourceEntry)
		if !mandatory && !important && entry.important {
			b.omitted++
			return nil
		}
		b.bytes -= len(entry.encoded) + 1
		(*queue)[0] = nil
		*queue = (*queue)[1:]
		if len(*queue) == 0 {
			*queue = nil
		}
		b.entries.Remove(remove)
		b.omitted++
	}
	if mandatory {
		b.required += size
	}
	entry := b.entries.PushBack(sourceEntry{digest, encoded, mandatory, important})
	if !mandatory {
		if important {
			b.important = append(b.important, entry)
		} else {
			b.normal = append(b.normal, entry)
		}
	}
	b.bytes += size
	return nil
}

func (b *sourceBuffer) packet() []json.RawMessage {
	out := make([]json.RawMessage, 0, b.entries.Len())
	for entry := b.entries.Front(); entry != nil; entry = entry.Next() {
		out = append(out, entry.Value.(sourceEntry).encoded)
	}
	return out
}

func (b *sourceBuffer) refit(limit int, redact func(string) string) (*sourceBuffer, error) {
	next := &sourceBuffer{limit: limit, omitted: b.omitted}
	for entry := b.entries.Front(); entry != nil; entry = entry.Next() {
		var record Record
		if err := json.Unmarshal(entry.Value.(sourceEntry).encoded, &record); err != nil {
			return nil, err
		}
		if err := next.add(record, redact); err != nil {
			return nil, err
		}
	}
	return next, nil
}

func sourceExcerpt(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	marker := "\n[excerpt; full source retained in session history]\n"
	available := limit - len(marker)
	head := available / 2
	for head > 0 && !utf8.RuneStart(text[head]) {
		head--
	}
	tail := len(text) - (available - available/2)
	for tail < len(text) && !utf8.RuneStart(text[tail]) {
		tail++
	}
	return text[:head] + marker + text[tail:]
}
