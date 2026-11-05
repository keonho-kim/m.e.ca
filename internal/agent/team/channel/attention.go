// Purpose: Separate universally visible conversation from events that justify another peer decision.
// Pattern: Pure notification policy.
// Usage: Publication and inbox admission apply the same rule without semantic classification.
// Related: internal/agent/team/publication.go, internal/agent/team/inbox.go.
package channel

import "slices"

// Attention never assigns work or requires an answer. Ordinary speech is read at the next decision.
func (m Message) Attention(member string) bool {
	if m.From == "user" {
		return true
	}
	if m.From == member || m.Passive || m.FromStream {
		return false
	}
	if !m.Discussion || m.Note == nil {
		return true
	}
	if m.Note.ReplyAuthor == member || slices.Contains(m.Note.Mentions, member) {
		return true
	}
	// Referring to the author's earlier note is a public followup. Only
	// explicit peer mentions restrict its audience.
	return (m.Note.ReplyTo == 0 || m.Note.ReplyAuthor == m.From) && len(m.Note.Mentions) == 0
}

// OwnContext excludes native content already retained by its author's private conversation.
func (m Message) OwnContext() bool {
	return !m.FromStream && !m.Observation && !m.Declaration && !m.Discussion
}
