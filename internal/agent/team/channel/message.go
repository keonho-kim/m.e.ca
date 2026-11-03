// Purpose: Define committed public conversation records independently of membership and execution.
// Pattern: Domain value and consumer-owned log contract.
// Usage: Room publication and SQLite exchange the same ordered messages.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_messages.go.
package channel

import "context"

type Message struct {
	FromStream bool   `json:"-"`
	Final      bool   `json:"final,omitempty"`
	Sequence   int64  `json:"sequence"`
	From       string `json:"from"`
	Text       string `json:"text"`
}

type Repository interface {
	AppendTeamMessage(context.Context, Message) (Message, error)
	TeamMessages(context.Context, int64, int64, int) ([]Message, error)
	TeamPending(context.Context, int64, int64, string) (bool, error)
}
