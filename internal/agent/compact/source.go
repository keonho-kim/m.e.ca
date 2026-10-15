// Purpose: Fit one chronological summary request while preserving user messages in full.
// Pattern: Bounded source selection.
// Usage: Controller adds projected records, then sends one packet to summarize.
// Related: internal/agent/compact/controller.go, internal/storage/history/compaction.go.
package compact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type sourceEntry struct {
	encoded   json.RawMessage
	mandatory bool
	important bool
}
type sourceBuffer struct {
	entries                         []sourceEntry
	bytes, required, limit, omitted int
}

func (b *sourceBuffer) add(record Record, redact func(string) string) error {
	if strings.TrimSpace(record.Text) == "" {
		return nil
	}
	mandatory := record.Role == "user"
	text := record.Text
	if !mandatory {
		text = sourceExcerpt(text, MaxBodyBytes)
	}
	encoded, err := json.Marshal(struct{ Role, Text string }{record.Role, redact(text)})
	if err != nil {
		return err
	}
	important := record.Role == "outcome" || record.Role == "failure"
	size := len(encoded) + 1
	if !mandatory && len(b.entries) > 0 {
		last := b.entries[len(b.entries)-1]
		if !last.mandatory && bytes.Equal(last.encoded, encoded) {
			return nil
		}
	}
	if mandatory && b.required+size > b.limit {
		return fmt.Errorf("compaction user instructions exceed max_input_bytes; previous memory and original history retained; increase compaction.max_input_bytes")
	}
	if !mandatory && size > b.limit-b.required {
		b.omitted++
		return nil
	}
	for b.bytes+size > b.limit {
		remove := -1
		for i, entry := range b.entries {
			if entry.mandatory {
				continue
			}
			if remove < 0 || b.entries[remove].important && !entry.important {
				remove = i
			}
		}
		if remove < 0 {
			return fmt.Errorf("compaction source budget exhausted")
		}
		if !mandatory && !important && b.entries[remove].important {
			b.omitted++
			return nil
		}
		b.bytes -= len(b.entries[remove].encoded) + 1
		copy(b.entries[remove:], b.entries[remove+1:])
		b.entries[len(b.entries)-1] = sourceEntry{}
		b.entries = b.entries[:len(b.entries)-1]
		b.omitted++
	}
	if mandatory {
		b.required += size
	}
	b.entries = append(b.entries, sourceEntry{encoded, mandatory, important})
	b.bytes += size
	return nil
}

func (b *sourceBuffer) packet() []json.RawMessage {
	out := make([]json.RawMessage, len(b.entries))
	for i, entry := range b.entries {
		out[i] = entry.encoded
	}
	return out
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
