// Purpose: Define committed public conversation records independently of membership and execution.
// Pattern: Domain value and consumer-owned log contract.
// Usage: Room publication and SQLite exchange the same ordered messages.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_messages.go.
package channel

import (
	"context"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

// PageBytes bounds a receive page, independently of the accepted size of one record.
const PageBytes = 16 * 1024 * 1024

type Message struct {
	Session      string                    `json:"-"`
	Prefix       *string                   `json:"-"`
	Note         *runrecord.TeamNote       `json:"note,omitempty"`
	Reference    *runrecord.EventReference `json:"-"`
	Discussion   bool                      `json:"-"`
	Passive      bool                      `json:"passive,omitempty"`
	Declaration  bool                      `json:"-"`
	Observation  bool                      `json:"observation,omitempty"`
	ChangedPaths []string                  `json:"-"`
	FromStream   bool                      `json:"-"`
	Final        bool                      `json:"final,omitempty"`
	Sequence     int64                     `json:"sequence"`
	From         string                    `json:"from"`
	Text         string                    `json:"text"`
}

// ContextBytes conservatively sizes attribution and all metadata without formatting.
func (m Message) ContextBytes() int {
	if m.Prefix != nil {
		return len(m.Text) + len(*m.Prefix)
	}
	size := len(m.Text) + len(m.From) + 512
	if n := m.Note; n != nil {
		size += len(n.ReplyAuthor)
		for _, name := range n.Mentions {
			size += len(name) + 2
		}
		if w := n.Work; w != nil {
			size += len(w.Contract)
			for _, path := range w.Paths {
				size += len(path) + 2
			}
		}
	}
	return size
}

// Kind owns the persisted provenance shared by live delivery and log restoration.
func (m Message) Kind() string {
	switch {
	case m.From == "user":
		return "user_message"
	case m.Observation && m.Passive:
		return "team_status"
	case m.Observation:
		return "team_observation"
	case m.FromStream:
		return "team_progress"
	case m.Declaration:
		return "team_work"
	case m.Discussion:
		return "team_discussion"
	default:
		return "team_message"
	}
}

type Reader interface {
	TeamMessages(context.Context, int64, int64, int) ([]Message, error)
}

// Metadata identifies a public record without restoring its potentially large body.
type Metadata struct {
	Sequence   int64
	From, Kind string
	Bytes      int
}

type Repository interface {
	Reader
	TeamMessageMetadata(context.Context, int64) (Metadata, error)
	AppendTeamMessage(context.Context, Message) (Message, error)
}
