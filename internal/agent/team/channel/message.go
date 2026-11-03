// Purpose: Define committed public conversation records independently of membership and execution.
// Pattern: Domain value and consumer-owned log contract.
// Usage: Room publication and SQLite exchange the same ordered messages.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_messages.go.
package channel

import (
	"context"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type Message struct {
	Reference    *runrecord.EventReference `json:"-"`
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
	default:
		return "team_message"
	}
}

type Repository interface {
	AppendTeamMessage(context.Context, Message) (Message, error)
	TeamMessages(context.Context, int64, int64, int) ([]Message, error)
	TeamPending(context.Context, int64, int64, string) (bool, error)
}
