// Purpose: Render committed room messages as attributed context with explicit user/peer provenance.
// Pattern: Pure message projection.
// Usage: Inbox admission and shared-payload restoration use identical text and identities.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_payloads.go.
package channel

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/team/source"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

// Context distinguishes actual user instructions from attributed peer observations.
func (m Message) Context() modelapi.Message {
	origin := &source.Public{Session: m.Session, Sequence: m.Sequence, Author: m.From, Kind: m.Kind()}
	if m.From == "user" {
		return modelapi.Message{Source: origin, ID: fmt.Sprintf("room:user/%d", m.Sequence), Role: "user", Content: m.Text, Images: m.Images}
	}
	prefix := m.ContextPrefix()
	return modelapi.Message{Source: origin, ID: fmt.Sprintf("peer:%s/%d", m.From, m.Sequence), Role: "system", Content: prefix + m.Text}
}

// ContextPrefix freezes attribution separately from the shared source text.
func (m Message) ContextPrefix() string {
	if m.Prefix != nil {
		return *m.Prefix
	}

	if m.From == "user" {
		return ""
	}
	var text strings.Builder
	size := len(m.From) + 64
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
	return text.String()
}

// Sequence returns an explicit public receive or authored-result reference.
func Sequence(message modelapi.Message) (int64, bool) {
	if message.Source == nil || message.Source.Sequence < 1 {
		return 0, false
	}
	return message.Source.Sequence, true
}

// IsPeer distinguishes attributed observations from actual user instructions.
func IsPeer(message modelapi.Message) bool {
	return message.Source != nil && message.Source.Author != "user" && message.Role == "system"
}
