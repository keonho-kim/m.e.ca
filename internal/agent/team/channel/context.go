// Purpose: Render committed room messages as attributed context with explicit user/peer provenance.
// Pattern: Pure message projection.
// Usage: Inbox admission and shared-payload restoration use identical text and identities.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_payloads.go.
package channel

import (
	"fmt"
	"strconv"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

// Context distinguishes actual user instructions from attributed peer observations.
func (m Message) Context() modelapi.Message {
	if m.From == "user" {
		return modelapi.Message{ID: fmt.Sprintf("room:user/%d", m.Sequence), Role: "user", Content: m.Text}
	}
	var text strings.Builder
	size := len(m.Text) + len(m.From) + 64
	if note := m.Note; note != nil {
		size += len(note.ReplyAuthor) + 128
		for _, name := range note.Mentions {
			size += len(name) + 2
		}
		if work := note.Work; work != nil {
			size += len(work.Contract) + 128
			for _, path := range work.Paths {
				size += len(path) + 2
			}
		}
	}
	text.Grow(size)
	text.WriteString("Public conversation — ")
	text.WriteString(m.From)
	if m.Note != nil {
		fmt.Fprintf(&text, " [#%d; observed through #%d]", m.Sequence, m.Note.BasedOn)
		if m.Note.ReplyTo > 0 {
			fmt.Fprintf(&text, "\nReply to #%d by %s", m.Note.ReplyTo, m.Note.ReplyAuthor)
		}
		if len(m.Note.Mentions) > 0 {
			text.WriteString("\nFor attention: ")
			for i, name := range m.Note.Mentions {
				if i > 0 {
					text.WriteString(", ")
				}
				text.WriteString(name)
			}
		}
		if work := m.Note.Work; work != nil {
			fmt.Fprintf(&text, "\nSelf-reported work v%d; paths: ", work.Revision)
			for i, path := range work.Paths {
				if i > 0 {
					text.WriteString(", ")
				}
				text.WriteString(path)
			}
			if work.Contract != "" {
				text.WriteString("\nDeclared interface/behavior: ")
				text.WriteString(work.Contract)
			}
		}
	}
	text.WriteByte('\n')
	text.WriteString(m.Text)
	return modelapi.Message{ID: fmt.Sprintf("peer:%s/%d", m.From, m.Sequence), Role: "system", Content: text.String()}
}

// Sequence recognizes context records backed by the public channel.
func Sequence(id string) (int64, bool) {
	if !IsPeer(id) && !strings.HasPrefix(id, "room:") {
		return 0, false
	}
	index := strings.LastIndexByte(id, '/')
	if index < 0 {
		return 0, false
	}
	sequence, err := strconv.ParseInt(id[index+1:], 10, 64)
	return sequence, err == nil && sequence > 0
}

// IsPeer identifies the provenance marker used by private context and compaction.
func IsPeer(id string) bool { return strings.HasPrefix(id, "peer:") }
